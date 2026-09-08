package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// createTempFiles creates n files with given names in dir.
func createTempFiles(t *testing.T, dir string, names []string) {
	t.Helper()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatalf("failed to create temp file %q: %v", name, err)
		}
	}
}

func TestMoveRandomFiles_MovesExactCount(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	names := []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt"}
	createTempFiles(t, src, names)

	moved, err := MoveRandomFiles(3, src, dst, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moved) != 3 {
		t.Errorf("expected exactly 3 files moved, got %d", len(moved))
	}

	// Verify files actually exist in dst and are gone from src.
	srcEntries, _ := os.ReadDir(src)
	dstEntries, _ := os.ReadDir(dst)

	if len(dstEntries) != len(moved) {
		t.Errorf("expected %d files in dst, got %d", len(moved), len(dstEntries))
	}
	if len(srcEntries)+len(moved) != len(names) {
		t.Errorf("src+dst file count mismatch: src=%d moved=%d total=%d", len(srcEntries), len(moved), len(names))
	}
}

func TestMoveRandomFiles_MaxFilesGreaterThanAvailable(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	names := []string{"x.txt", "y.txt"}
	createTempFiles(t, src, names)

	moved, err := MoveRandomFiles(100, src, dst, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moved) != 2 {
		t.Errorf("expected all 2 files moved, got %d", len(moved))
	}
	dstEntries, _ := os.ReadDir(dst)
	if len(dstEntries) != 2 {
		t.Errorf("expected 2 files in dst, got %d", len(dstEntries))
	}
}

func TestMoveRandomFiles_EmptySource(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	_, err := MoveRandomFiles(5, src, dst, "")
	if err == nil {
		t.Error("expected error for empty source directory, got nil")
	}
}

func TestMoveRandomFiles_NonExistentSource(t *testing.T) {
	_, err := MoveRandomFiles(5, "/nonexistent/path/xyz", t.TempDir(), "")
	if err == nil {
		t.Error("expected error for non-existent source, got nil")
	}
}

func TestMoveRandomFiles_CreatesDstIfAbsent(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "new_subdir")

	createTempFiles(t, src, []string{"file.txt"})

	moved, err := MoveRandomFiles(1, src, dst, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moved) != 1 {
		t.Errorf("expected 1 file moved, got %d", len(moved))
	}
	if _, err := os.Stat(filepath.Join(dst, "file.txt")); os.IsNotExist(err) {
		t.Error("expected file.txt in newly created dst dir")
	}
}

// TestMoveFile_CopyFallback exercises the copy-then-delete fallback path
// by injecting a rename function that always fails, forcing the fallback.
func TestMoveFile_CopyFallback(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(dir, "sub", "dest.txt")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}

	failingRename := func(_, _ string) error {
		return fmt.Errorf("simulated rename failure")
	}

	if err := moveFileWithRename(src, dst, failingRename); err != nil {
		t.Fatalf("moveFile returned unexpected error: %v", err)
	}

	// Source must be gone.
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source file still exists after move")
	}

	// Destination must have the correct content.
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("cannot read destination file: %v", err)
	}
	if string(got) != "content" {
		t.Errorf("content mismatch: got %q", got)
	}
}

func TestMoveRandomFiles_FileContentsPreserved(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	content := []byte("hello world")
	if err := os.WriteFile(filepath.Join(src, "test.txt"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	moved, err := MoveRandomFiles(1, src, dst, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moved) != 1 {
		t.Errorf("expected 1 file moved, got %d", len(moved))
	}

	got, err := os.ReadFile(filepath.Join(dst, "test.txt"))
	if err != nil {
		t.Fatalf("cannot read moved file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestMoveRandomFiles_FiltersByExtension(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	createTempFiles(t, src, []string{"a.txt", "b.txt", "c.go", "d.go", "e.md"})

	moved, err := MoveRandomFiles(2, src, dst, ".go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moved) != 2 {
		t.Errorf("expected 2 files moved, got %d", len(moved))
	}
	for _, name := range moved {
		if filepath.Ext(name) != ".go" {
			t.Errorf("expected only .go files, got %q", name)
		}
	}

	// .txt and .md files should remain in src.
	srcEntries, _ := os.ReadDir(src)
	remaining := make(map[string]bool)
	for _, e := range srcEntries {
		remaining[e.Name()] = true
	}
	for _, want := range []string{"a.txt", "b.txt", "e.md"} {
		if !remaining[want] {
			t.Errorf("expected %q to remain in source", want)
		}
	}
}

func TestMoveRandomFiles_ExtensionWithoutDot(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	createTempFiles(t, src, []string{"a.txt", "b.go"})

	moved, err := MoveRandomFiles(1, src, dst, "txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moved) != 1 {
		t.Errorf("expected 1 file moved, got %d", len(moved))
	}
	if filepath.Ext(moved[0]) != ".txt" {
		t.Errorf("expected .txt file, got %q", moved[0])
	}
}

func TestMoveRandomFiles_ExtensionCaseInsensitive(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	createTempFiles(t, src, []string{"a.TXT", "b.txt"})

	moved, err := MoveRandomFiles(2, src, dst, ".txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moved) != 2 {
		t.Errorf("expected 2 files moved (case-insensitive match), got %d", len(moved))
	}
}

func TestMoveRandomFiles_NoMatchingExtension(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	createTempFiles(t, src, []string{"a.txt", "b.txt"})

	_, err := MoveRandomFiles(1, src, dst, ".go")
	if err == nil {
		t.Error("expected error when no files match extension, got nil")
	}
}

func TestMoveRandomFiles_EmptySourceWithExtension(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	_, err := MoveRandomFiles(1, src, dst, ".txt")
	if err == nil {
		t.Error("expected error for empty source directory with extension filter, got nil")
	}
}

func TestMoveRandomFiles_SameSourceAndDest(t *testing.T) {
	dir := t.TempDir()
	createTempFiles(t, dir, []string{"a.txt"})

	_, err := MoveRandomFiles(1, dir, dir, "")
	if err == nil {
		t.Error("expected error when source and destination are the same, got nil")
	}
}
