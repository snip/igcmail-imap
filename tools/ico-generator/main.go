package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input.png> <output.ico>\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	// Read the input PNG file
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Decode the PNG
	srcImg, err := png.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding PNG: %v\n", err)
		os.Exit(1)
	}

	// Standard ICO sizes for Windows (including high DPI sizes)
	sizes := []int{16, 24, 32, 48, 64, 128, 256}

	// Generate ICO file
	err = createICOFile(outputFile, srcImg, sizes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating ICO file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created ICO file with sizes: %v\n", sizes)
}

// resizeImage resizes an image to the specified width and height using bilinear interpolation
func resizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// imageToBMP converts an image to BMP format suitable for ICO files
// ICO files expect 32-bit BGRA format with rows stored bottom-to-top
func imageToBMP(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// BMP header for 32-bit BGRA
	bmpHeaderSize := 40
	imageSize := width * height * 4
	fileSize := 14 + bmpHeaderSize + imageSize

	// BMP file header (14 bytes)
	fileHeader := make([]byte, 14)
	fileHeader[0] = 'B'  // Signature
	fileHeader[1] = 'M'
	binary.LittleEndian.PutUint32(fileHeader[2:], uint32(fileSize)) // File size
	binary.LittleEndian.PutUint32(fileHeader[10:], uint32(14+bmpHeaderSize)) // Data offset

	// BMP info header (40 bytes)
	infoHeader := make([]byte, 40)
	binary.LittleEndian.PutUint32(infoHeader[0:], 40) // Header size
	binary.LittleEndian.PutUint32(infoHeader[4:], uint32(width))  // Width
	binary.LittleEndian.PutUint32(infoHeader[8:], uint32(height)) // Height
	binary.LittleEndian.PutUint16(infoHeader[12:], 1)  // Planes
	binary.LittleEndian.PutUint16(infoHeader[14:], 32) // Bits per pixel
	binary.LittleEndian.PutUint32(infoHeader[16:], 0)  // Compression
	binary.LittleEndian.PutUint32(infoHeader[20:], uint32(imageSize)) // Image size
	binary.LittleEndian.PutUint32(infoHeader[24:], 2835) // X pixels per meter
	binary.LittleEndian.PutUint32(infoHeader[28:], 2835) // Y pixels per meter

	// Convert image to BGRA format, bottom-to-top
	imageData := make([]byte, imageSize)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// ICO stores rows bottom-to-top
			srcY := height - 1 - y
			r, g, b, a := img.At(x, srcY).RGBA()
			offset := (y*width + x) * 4
			imageData[offset] = byte(b >> 8)     // B
			imageData[offset+1] = byte(g >> 8)   // G
			imageData[offset+2] = byte(r >> 8)   // R
			imageData[offset+3] = byte(a >> 8)   // A
		}
	}

	// Combine headers and data
	result := make([]byte, 0, fileSize)
	result = append(result, fileHeader...)
	result = append(result, infoHeader...)
	result = append(result, imageData...)

	return result, nil
}

// createICOFile creates an ICO file with multiple sizes from a source image
func createICOFile(filename string, srcImg image.Image, sizes []int) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// ICO file header (6 bytes)
	// Reserved: 0
	// Type: 1 (ICO)
	// Count: number of images
	header := struct {
		Reserved uint16
		Type     uint16
		Count    uint16
	}{
		Reserved: 0,
		Type:     1,
		Count:    uint16(len(sizes)),
	}

	if err := binary.Write(file, binary.LittleEndian, header); err != nil {
		return err
	}

	// Prepare image data and directory entries
	var dirEntries []ICODirEntry
	var imageData []byte
	offset := uint32(6 + len(sizes)*16) // Header + directory entries

	for _, size := range sizes {
		resized := resizeImage(srcImg, size, size)

		// Convert to BMP format for traditional ICO storage
		bmpData, err := imageToBMP(resized)
		if err != nil {
			return err
		}

		dirEntries = append(dirEntries, ICODirEntry{
			Width:       uint8(size),
			Height:      uint8(size),
			ColorCount:  0, // 0 = more than 256 colors
			Reserved:    0,
			Planes:      1,
			BPP:         32,
			Size:        uint32(len(bmpData)),
			Offset:      offset,
		})

		imageData = append(imageData, bmpData...)
		offset += uint32(len(bmpData))
	}

	// Write directory entries
	for _, entry := range dirEntries {
		if err := binary.Write(file, binary.LittleEndian, entry); err != nil {
			return err
		}
	}

	// Write image data
	if _, err := file.Write(imageData); err != nil {
		return err
	}

	return nil
}

// ICODirEntry represents a directory entry in an ICO file
type ICODirEntry struct {
	Width      uint8
	Height     uint8
	ColorCount uint8
	Reserved   uint8
	Planes     uint16
	BPP        uint16
	Size       uint32
	Offset     uint32
}