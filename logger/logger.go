package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const logFileName = "igcmailimap.log"

// Logger handles logging to a file in the output directory
type Logger struct {
	enabled bool
	logFile *os.File
	logger  *log.Logger
}

// New creates a new logger that writes to the specified output directory
func New(outputDir string, enabled bool) (*Logger, error) {
	l := &Logger{
		enabled: enabled,
	}

	if enabled && outputDir != "" {
		logPath := filepath.Join(outputDir, logFileName)

		// Create the output directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}

		// Open log file in append mode, create if it doesn't exist
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		l.logFile = file
		l.logger = log.New(file, "", log.LstdFlags)
	}

	return l, nil
}

// Close closes the log file if it's open
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Info logs an info message
func (l *Logger) Info(message string) {
	if l.enabled && l.logger != nil {
		l.logger.Println("[INFO] " + message)
	}
}

// Error logs an error message
func (l *Logger) Error(message string) {
	if l.enabled && l.logger != nil {
		l.logger.Println("[ERROR] " + message)
	}
}

// Warning logs a warning message
func (l *Logger) Warning(message string) {
	if l.enabled && l.logger != nil {
		l.logger.Println("[WARNING] " + message)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	if l.enabled && l.logger != nil {
		l.logger.Println("[DEBUG] " + message)
	}
}

// LogFetch logs information about a fetch operation
func (l *Logger) LogFetch(messagesFound int, outputDir string, uids []uint32) {
	if l.enabled && l.logger != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		message := fmt.Sprintf("[%s] IMAP fetch completed: %d new messages found (UIDs: %v), output directory: %s",
			timestamp, messagesFound, uids, outputDir)
		l.logger.Println("[FETCH] " + message)
	}
}

// LogMessageDetails logs details about fetched messages
func (l *Logger) LogMessageDetails(uid uint32, subject, from string) {
	if l.enabled && l.logger != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		message := fmt.Sprintf("[%s] Fetched message UID %d - Subject: '%s', From: '%s'",
			timestamp, uid, subject, from)
		l.logger.Println("[MESSAGE] " + message)
	}
}

// ExtractResult holds information about a single extracted file
type ExtractResult struct {
	Filename string // The filename that was saved
	Path     string // Full path to the saved file
}

// LogMessageExtract logs details about files extracted from a message
func (l *Logger) LogMessageExtract(uid uint32, subject, from string, results []ExtractResult, outputDir string) {
	if l.enabled && l.logger != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		filenames := make([]string, len(results))
		for i, result := range results {
			filenames[i] = result.Filename
		}
		message := fmt.Sprintf("[%s] Extracted %d files from UID %d (Subject: '%s', From: '%s') - Files: %v",
			timestamp, len(results), uid, subject, from, filenames)
		l.logger.Println("[EXTRACT] " + message)
	}
}

// LogExtract logs information about file extraction
func (l *Logger) LogExtract(filesSaved int, outputDir string) {
	if l.enabled && l.logger != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		message := fmt.Sprintf("[%s] IGC extraction completed: %d files saved to %s",
			timestamp, filesSaved, outputDir)
		l.logger.Println("[EXTRACT] " + message)
	}
}
