package main

import (
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

func TestMoveRandomFiles_MovesAtLeastOne(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	names := []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt"}
	createTempFiles(t, src, names)

	moved, err := MoveRandomFiles(3, src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if moved < 1 || moved > 3 {
		t.Errorf("expected 1..3 files moved, got %d", moved)
	}

	// Verify files actually exist in dst and are gone from src.
	srcEntries, _ := os.ReadDir(src)
	dstEntries, _ := os.ReadDir(dst)

	if len(dstEntries) != moved {
		t.Errorf("expected %d files in dst, got %d", moved, len(dstEntries))
	}
	if len(srcEntries)+moved != len(names) {
		t.Errorf("src+dst file count mismatch: src=%d moved=%d total=%d", len(srcEntries), moved, len(names))
	}
}

func TestMoveRandomFiles_MaxFilesGreaterThanAvailable(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	names := []string{"x.txt", "y.txt"}
	createTempFiles(t, src, names)

	moved, err := MoveRandomFiles(100, src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if moved < 1 || moved > 2 {
		t.Errorf("expected 1 or 2 files moved, got %d", moved)
	}
}

func TestMoveRandomFiles_EmptySource(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	_, err := MoveRandomFiles(5, src, dst)
	if err == nil {
		t.Error("expected error for empty source directory, got nil")
	}
}

func TestMoveRandomFiles_NonExistentSource(t *testing.T) {
	_, err := MoveRandomFiles(5, "/nonexistent/path/xyz", t.TempDir())
	if err == nil {
		t.Error("expected error for non-existent source, got nil")
	}
}

func TestMoveRandomFiles_CreatesDstIfAbsent(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "new_subdir")

	createTempFiles(t, src, []string{"file.txt"})

	moved, err := MoveRandomFiles(1, src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if moved != 1 {
		t.Errorf("expected 1 file moved, got %d", moved)
	}
	if _, err := os.Stat(filepath.Join(dst, "file.txt")); os.IsNotExist(err) {
		t.Error("expected file.txt in newly created dst dir")
	}
}

// TestMoveFile_CopyFallbackNoError is a regression test for the bug where
// moveFile always returned an error on the copy-then-delete fallback path
// because fmt.Errorf wrapped os.Remove's nil return value unconditionally.
// It exercises moveFile directly and verifies that a successful move returns
// no error.
func TestMoveFile_CopyFallbackNoError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(dir, "sub", "dest.txt")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := moveFile(src, dst); err != nil {
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

	moved, err := MoveRandomFiles(1, src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if moved != 1 {
		t.Errorf("expected 1 file moved, got %d", moved)
	}

	got, err := os.ReadFile(filepath.Join(dst, "test.txt"))
	if err != nil {
		t.Fatalf("cannot read moved file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}
