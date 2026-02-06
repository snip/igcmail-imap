package main

import (
	"bytes"
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

	// Standard ICO sizes for Windows
	sizes := []int{16, 32, 48, 64}

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

		// Convert to PNG format for storage in ICO
		var buf bytes.Buffer
		if err := png.Encode(&buf, resized); err != nil {
			return err
		}

		pngData := buf.Bytes()

		dirEntries = append(dirEntries, ICODirEntry{
			Width:       uint8(size),
			Height:      uint8(size),
			ColorCount:  0, // 0 = more than 256 colors
			Reserved:    0,
			Planes:      1,
			BPP:         32,
			Size:        uint32(len(pngData)),
			Offset:      offset,
		})

		imageData = append(imageData, pngData...)
		offset += uint32(len(pngData))
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