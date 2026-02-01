package extract

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
)

const igcExt = ".igc"

// ExtractResult holds information about a single extracted file
type ExtractResult struct {
	Filename string // The filename that was saved
	Path     string // Full path to the saved file
}

// IGCOnly returns true if the filename (after lowercasing extension) is .igc.
func IGCOnly(filename string) bool {
	return strings.EqualFold(filepath.Ext(filename), igcExt)
}

// SaveDir holds the output directory and tracks existing filenames for duplicate handling.
type SaveDir struct {
	Dir       string
	usedNames map[string]struct{}
}

// NewSaveDir returns a SaveDir for the given output directory.
func NewSaveDir(dir string) *SaveDir {
	return &SaveDir{
		Dir:       dir,
		usedNames: make(map[string]struct{}),
	}
}

// SavePath returns the path to use for an IGC attachment. If the base name was already used,
// returns a path with timestamp prefix + "duplicate" as requested.
func (d *SaveDir) SavePath(baseName string) string {
	if !IGCOnly(baseName) {
		return ""
	}
	name := baseName
	if !strings.HasSuffix(strings.ToLower(name), igcExt) {
		name = name + igcExt
	}
	fullPath := filepath.Join(d.Dir, name)
	if _, used := d.usedNames[name]; used {
		ts := time.Now().Format("20060102150405")
		name = ts + "_duplicate_" + name
		fullPath = filepath.Join(d.Dir, name)
	}
	d.usedNames[name] = struct{}{}
	return fullPath
}

// ExtractIGCAttachments parses the raw RFC822 message body and saves each .igc attachment
// into the given SaveDir. Returns the extracted files and any error (e.g. from writing).
func ExtractIGCAttachments(raw []byte, out *SaveDir) ([]ExtractResult, error) {
	if out == nil || out.Dir == "" {
		return nil, nil
	}
	m, err := message.Read(strings.NewReader(string(raw)))
	if err != nil && !message.IsUnknownCharset(err) {
		return nil, err
	}
	mr := m.MultipartReader()
	if mr == nil {
		// Single part: check if the whole body is IGC-named (e.g. Content-Disposition filename)
		disp, _, _ := m.Header.ContentDisposition()
		if disp == "attachment" {
			if filename, _ := m.Header.Text("Content-Disposition"); filename != "" {
				// Parse filename from Content-Disposition (simplified: look for filename=)
				if name := parseFilenameFromDisposition(filename); name != "" && IGCOnly(name) {
					path := out.SavePath(name)
					if path != "" {
						if wErr := writePartToFile(m.Body, path); wErr != nil {
							return nil, wErr
						}
						filename := filepath.Base(path)
						return []ExtractResult{{Filename: filename, Path: path}}, nil
					}
				}
			}
		}
		return nil, nil
	}

	var results []ExtractResult
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		_, params, _ := part.Header.ContentDisposition()
		filename := ""
		if params != nil {
			filename = params["filename"]
		}
		if filename == "" {
			if params == nil {
				params = make(map[string]string)
			}
			_, ctParams, _ := part.Header.ContentType()
			if ctParams != nil && ctParams["name"] != "" {
				filename = ctParams["name"]
			}
		}
		if filename == "" || !IGCOnly(filename) {
			continue
		}
		path := out.SavePath(filename)
		if path == "" {
			continue
		}
		if wErr := writePartToFile(part.Body, path); wErr != nil {
			return results, wErr
		}
		results = append(results, ExtractResult{
			Filename: filepath.Base(path),
			Path:     path,
		})
	}
	return results, nil
}

func parseFilenameFromDisposition(disp string) string {
	// Very simple: look for filename="..." or filename=...
	i := strings.Index(strings.ToLower(disp), "filename=")
	if i < 0 {
		return ""
	}
	disp = disp[i+len("filename="):]
	disp = strings.TrimSpace(disp)
	if len(disp) >= 2 && (disp[0] == '"' || disp[0] == '\'') {
		end := strings.IndexAny(disp[1:], `"'`)
		if end >= 0 {
			return disp[1 : 1+end]
		}
	}
	// Unquoted: take until space or semicolon
	j := 0
	for j < len(disp) && disp[j] != ' ' && disp[j] != ';' && disp[j] != '\r' && disp[j] != '\n' {
		j++
	}
	return disp[:j]
}

func writePartToFile(r io.Reader, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}
