package comm

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"
)

// LogReader provides incremental file reading with offset tracking and carry buffer.
// It is safe for concurrent use.
type LogReader struct {
	FilePath string
	offset   int64  // current read position
	carry    string // incomplete line from last read
	mu       sync.Mutex
}

// NewLogReader creates a new LogReader for the given file path.
func NewLogReader(path string) *LogReader {
	return &LogReader{FilePath: path}
}

// ReadNew reads lines appended since the last call.
// Incomplete trailing lines are buffered in carry and returned on the next call.
func (r *LogReader) ReadNew() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := os.Open(r.FilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Check file size; if file was truncated, reset offset
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() < r.offset {
		r.offset = 0
		r.carry = ""
	}

	if info.Size() == r.offset {
		return nil, nil // no new data
	}

	if _, err := f.Seek(r.offset, io.SeekStart); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	r.offset += int64(len(data))

	text := r.carry + string(data)
	r.carry = ""

	// Split into lines; last element may be incomplete
	parts := strings.Split(text, "\n")
	if len(parts) == 0 {
		return nil, nil
	}

	// The last element after split is either empty (if text ends with \n)
	// or an incomplete line. Either way, store it as carry.
	r.carry = parts[len(parts)-1]
	parts = parts[:len(parts)-1]

	var lines []string
	for _, p := range parts {
		line := strings.TrimRight(p, "\r")
		lines = append(lines, line)
	}

	return lines, nil
}

// ReadAll reads all lines from the file, resetting the offset to the end.
func (r *LogReader) ReadAll() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.FilePath)
	if err != nil {
		return nil, err
	}

	r.offset = int64(len(data))
	r.carry = ""

	text := string(data)
	if text == "" {
		return nil, nil
	}

	raw := strings.Split(text, "\n")
	var lines []string
	for _, p := range raw {
		line := strings.TrimRight(p, "\r")
		if line != "" || len(lines) > 0 { // preserve internal blank lines
			lines = append(lines, line)
		}
	}

	// Trim trailing empty line from final \n
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines, nil
}

// ReadTail reads the last n lines from the file without changing the offset.
func (r *LogReader) ReadTail(n int) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if n <= 0 {
		return nil, nil
	}

	f, err := os.Open(r.FilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read all lines (for moderate files) or use reverse reader for large ones
	var allLines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(allLines) <= n {
		return allLines, nil
	}
	return allLines[len(allLines)-n:], nil
}

// Reset resets the offset to the beginning of the file.
func (r *LogReader) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.offset = 0
	r.carry = ""
}

// SeekEnd moves the offset to the current end of the file.
// Subsequent ReadNew calls will only return data appended after this point.
func (r *LogReader) SeekEnd() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, err := os.Stat(r.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			r.offset = 0
			r.carry = ""
			return nil
		}
		return err
	}

	r.offset = info.Size()
	r.carry = ""
	return nil
}

// Offset returns the current read offset (for diagnostics).
func (r *LogReader) Offset() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.offset
}
