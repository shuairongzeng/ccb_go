package comm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReverseReaderReadLastLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("line1\nline2\nline3\nline4\nline5\n"), 0644)

	r := NewReverseReader(path)

	lines, err := r.ReadLastLines(3)
	if err != nil {
		t.Fatalf("ReadLastLines: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "line3" || lines[1] != "line4" || lines[2] != "line5" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestReverseReaderReadAllLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("a\nb\nc\n"), 0644)

	r := NewReverseReader(path)

	lines, err := r.ReadLastLines(100)
	if err != nil {
		t.Fatalf("ReadLastLines(100): %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestReverseReaderEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte(""), 0644)

	r := NewReverseReader(path)
	lines, err := r.ReadLastLines(5)
	if err != nil {
		t.Fatalf("ReadLastLines empty: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}
}

func TestReverseReaderSingleLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("only\n"), 0644)

	r := NewReverseReader(path)
	lines, err := r.ReadLastLines(1)
	if err != nil {
		t.Fatalf("ReadLastLines single: %v", err)
	}
	if len(lines) != 1 || lines[0] != "only" {
		t.Fatalf("expected ['only'], got %v", lines)
	}
}

func TestReverseReaderNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("line1\nline2\nline3"), 0644)

	r := NewReverseReader(path)
	lines, err := r.ReadLastLines(2)
	if err != nil {
		t.Fatalf("ReadLastLines no trailing: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "line2" || lines[1] != "line3" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestReverseReaderSmallChunkSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("line1\nline2\nline3\nline4\nline5\n"), 0644)

	r := NewReverseReader(path)
	r.ChunkSize = 4 // very small chunks to test boundary handling

	lines, err := r.ReadLastLines(3)
	if err != nil {
		t.Fatalf("ReadLastLines small chunk: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "line3" || lines[1] != "line4" || lines[2] != "line5" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestReverseReaderZeroLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("line1\n"), 0644)

	r := NewReverseReader(path)
	lines, err := r.ReadLastLines(0)
	if err != nil {
		t.Fatalf("ReadLastLines(0): %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}
}

func TestReverseReaderFindLast(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("apple\nbanana\ncherry\nbanana\ndate\n"), 0644)

	r := NewReverseReader(path)

	line, idx, err := r.FindLast(func(s string) bool {
		return strings.Contains(s, "banana")
	})
	if err != nil {
		t.Fatalf("FindLast: %v", err)
	}
	if line != "banana" {
		t.Fatalf("expected 'banana', got %q", line)
	}
	if idx != 3 {
		t.Fatalf("expected index 3, got %d", idx)
	}
}

func TestReverseReaderFindLastNoMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("apple\nbanana\n"), 0644)

	r := NewReverseReader(path)

	line, idx, err := r.FindLast(func(s string) bool {
		return strings.Contains(s, "cherry")
	})
	if err != nil {
		t.Fatalf("FindLast no match: %v", err)
	}
	if line != "" || idx != -1 {
		t.Fatalf("expected empty/-1, got %q/%d", line, idx)
	}
}

func TestReverseReaderLargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Create a file with 1000 lines
	var b strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b, "line-%04d\n", i)
	}
	os.WriteFile(path, []byte(b.String()), 0644)

	r := NewReverseReader(path)
	lines, err := r.ReadLastLines(5)
	if err != nil {
		t.Fatalf("ReadLastLines large: %v", err)
	}
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
	if lines[0] != "line-0995" || lines[4] != "line-0999" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestReverseReaderCRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	os.WriteFile(path, []byte("line1\r\nline2\r\nline3\r\n"), 0644)

	r := NewReverseReader(path)
	lines, err := r.ReadLastLines(2)
	if err != nil {
		t.Fatalf("ReadLastLines CRLF: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	for _, l := range lines {
		if strings.ContainsRune(l, '\r') {
			t.Fatalf("line still contains \\r: %q", l)
		}
	}
}

func TestReverseReaderNonExistent(t *testing.T) {
	r := NewReverseReader("/nonexistent/path/file.log")
	_, err := r.ReadLastLines(5)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
