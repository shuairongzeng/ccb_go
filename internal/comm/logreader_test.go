package comm

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestLogReaderReadNew(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Write initial content
	os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0644)

	r := NewLogReader(path)

	lines, err := r.ReadNew()
	if err != nil {
		t.Fatalf("ReadNew: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Fatalf("unexpected lines: %v", lines)
	}

	// No new data
	lines, err = r.ReadNew()
	if err != nil {
		t.Fatalf("ReadNew (no new): %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}

	// Append more data
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("line4\nline5\n")
	f.Close()

	lines, err = r.ReadNew()
	if err != nil {
		t.Fatalf("ReadNew (appended): %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "line4" || lines[1] != "line5" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestLogReaderCarryBuffer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Write content without trailing newline (incomplete line)
	os.WriteFile(path, []byte("line1\npartial"), 0644)

	r := NewLogReader(path)

	lines, err := r.ReadNew()
	if err != nil {
		t.Fatalf("ReadNew: %v", err)
	}
	// "partial" should be in carry, not returned
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "line1" {
		t.Fatalf("expected 'line1', got %q", lines[0])
	}

	// Append the rest of the partial line
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("_complete\nline3\n")
	f.Close()

	lines, err = r.ReadNew()
	if err != nil {
		t.Fatalf("ReadNew (carry): %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "partial_complete" {
		t.Fatalf("expected 'partial_complete', got %q", lines[0])
	}
	if lines[1] != "line3" {
		t.Fatalf("expected 'line3', got %q", lines[1])
	}
}

func TestLogReaderFileTruncation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0644)

	r := NewLogReader(path)
	r.ReadNew() // consume all

	// Truncate and write new content
	os.WriteFile(path, []byte("new1\n"), 0644)

	lines, err := r.ReadNew()
	if err != nil {
		t.Fatalf("ReadNew after truncation: %v", err)
	}
	if len(lines) != 1 || lines[0] != "new1" {
		t.Fatalf("expected ['new1'], got %v", lines)
	}
}

func TestLogReaderReadAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("a\nb\nc\n"), 0644)

	r := NewLogReader(path)
	lines, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	// Offset should be at end
	if r.Offset() != 6 { // "a\nb\nc\n" = 6 bytes
		t.Fatalf("expected offset 6, got %d", r.Offset())
	}
}

func TestLogReaderReadTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("1\n2\n3\n4\n5\n"), 0644)

	r := NewLogReader(path)
	lines, err := r.ReadTail(3)
	if err != nil {
		t.Fatalf("ReadTail: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "3" || lines[1] != "4" || lines[2] != "5" {
		t.Fatalf("unexpected tail: %v", lines)
	}
}

func TestLogReaderSeekEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("old1\nold2\n"), 0644)

	r := NewLogReader(path)
	r.SeekEnd()

	// Append new data
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("new1\n")
	f.Close()

	lines, err := r.ReadNew()
	if err != nil {
		t.Fatalf("ReadNew after SeekEnd: %v", err)
	}
	if len(lines) != 1 || lines[0] != "new1" {
		t.Fatalf("expected ['new1'], got %v", lines)
	}
}

func TestLogReaderReset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("a\nb\n"), 0644)

	r := NewLogReader(path)
	r.ReadNew() // consume all

	r.Reset()
	lines, err := r.ReadNew()
	if err != nil {
		t.Fatalf("ReadNew after Reset: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines after reset, got %d", len(lines))
	}
}

func TestLogReaderConcurrentSafety(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("init\n"), 0644)

	r := NewLogReader(path)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.ReadNew()
			r.ReadAll()
			r.ReadTail(5)
			r.SeekEnd()
			r.Offset()
		}()
	}
	wg.Wait()
}

func TestLogReaderCRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("line1\r\nline2\r\nline3\r\n"), 0644)

	r := NewLogReader(path)
	lines, err := r.ReadNew()
	if err != nil {
		t.Fatalf("ReadNew CRLF: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	for _, l := range lines {
		if strings.ContainsRune(l, '\r') {
			t.Fatalf("line still contains \\r: %q", l)
		}
	}
}

func TestLogReaderEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte(""), 0644)

	r := NewLogReader(path)
	lines, err := r.ReadNew()
	if err != nil {
		t.Fatalf("ReadNew empty: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}
}

func TestLogReaderNonExistentFile(t *testing.T) {
	r := NewLogReader("/nonexistent/path/file.log")
	_, err := r.ReadNew()
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
