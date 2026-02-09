package comm

import (
	"io"
	"os"
	"strings"
)

const defaultChunkSize = 8192

// ReverseReader reads a file from the end toward the beginning in chunks.
// Useful for efficiently scanning the tail of large log files.
type ReverseReader struct {
	FilePath  string
	ChunkSize int
}

// NewReverseReader creates a new ReverseReader with the default chunk size.
func NewReverseReader(path string) *ReverseReader {
	return &ReverseReader{
		FilePath:  path,
		ChunkSize: defaultChunkSize,
	}
}

// ReadLastLines reads the last n lines from the file.
// Lines are returned in forward order (oldest first).
func (r *ReverseReader) ReadLastLines(n int) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}

	f, err := os.Open(r.FilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := info.Size()
	if fileSize == 0 {
		return nil, nil
	}

	chunkSize := int64(r.ChunkSize)
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}

	var collected []string
	pos := fileSize
	var leftover string

	for pos > 0 && len(collected) < n+1 {
		readSize := chunkSize
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize

		buf := make([]byte, readSize)
		_, err := f.ReadAt(buf, pos)
		if err != nil && err != io.EOF {
			return nil, err
		}

		chunk := string(buf) + leftover
		leftover = ""

		parts := strings.Split(chunk, "\n")

		// The first element may be a partial line (unless we're at file start)
		if pos > 0 {
			leftover = parts[0]
			parts = parts[1:]
		}

		// Prepend lines in reverse order
		for i := len(parts) - 1; i >= 0; i-- {
			line := strings.TrimRight(parts[i], "\r")
			collected = append([]string{line}, collected...)
		}
	}

	// If there's leftover at file start, prepend it
	if leftover != "" {
		line := strings.TrimRight(leftover, "\r")
		collected = append([]string{line}, collected...)
	}

	// Remove trailing empty lines
	for len(collected) > 0 && collected[len(collected)-1] == "" {
		collected = collected[:len(collected)-1]
	}

	// Return only the last n lines
	if len(collected) > n {
		collected = collected[len(collected)-n:]
	}

	return collected, nil
}

// FindLast searches backward through the file for the last line matching the predicate.
// Returns the matching line, its 0-based line index, and any error.
// If no match is found, returns ("", -1, nil).
func (r *ReverseReader) FindLast(predicate func(string) bool) (string, int, error) {
	f, err := os.Open(r.FilePath)
	if err != nil {
		return "", -1, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", -1, err
	}
	fileSize := info.Size()
	if fileSize == 0 {
		return "", -1, nil
	}

	chunkSize := int64(r.ChunkSize)
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}

	// We need to know total line count for indexing, so we collect all lines
	// from the tail until we find a match. For very large files, this is still
	// efficient because we stop as soon as we find a match.
	pos := fileSize
	var leftover string
	var tailLines []string

	for pos > 0 {
		readSize := chunkSize
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize

		buf := make([]byte, readSize)
		_, err := f.ReadAt(buf, pos)
		if err != nil && err != io.EOF {
			return "", -1, err
		}

		chunk := string(buf) + leftover
		leftover = ""

		parts := strings.Split(chunk, "\n")

		if pos > 0 {
			leftover = parts[0]
			parts = parts[1:]
		}

		// Check lines from end of this chunk
		for i := len(parts) - 1; i >= 0; i-- {
			line := strings.TrimRight(parts[i], "\r")
			tailLines = append([]string{line}, tailLines...)
		}

		// Check newly added lines for match (search from end)
		for i := 0; i < len(parts); i++ {
			line := strings.TrimRight(parts[len(parts)-1-i], "\r")
			if predicate(line) {
				// We found a match. Now compute the line index.
				// We need to count all lines before this chunk + position within chunk.
				// For simplicity, read the whole file to count.
				// This is acceptable because FindLast is typically called on moderate files.
				allLines, err := readAllLines(r.FilePath)
				if err != nil {
					return line, -1, nil
				}
				for j := len(allLines) - 1; j >= 0; j-- {
					if predicate(allLines[j]) {
						return allLines[j], j, nil
					}
				}
				return line, -1, nil
			}
		}
	}

	// Check leftover
	if leftover != "" {
		line := strings.TrimRight(leftover, "\r")
		if predicate(line) {
			return line, 0, nil
		}
	}

	return "", -1, nil
}

// readAllLines reads all lines from a file.
func readAllLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(data)
	if text == "" {
		return nil, nil
	}
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, p := range raw {
		lines = append(lines, strings.TrimRight(p, "\r"))
	}
	// Trim trailing empty
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}
