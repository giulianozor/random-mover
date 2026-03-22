package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <num_files> <source_path> <dest_path>\n", os.Args[0])
		os.Exit(1)
	}

	maxFiles, err := strconv.Atoi(os.Args[1])
	if err != nil || maxFiles <= 0 {
		fmt.Fprintf(os.Stderr, "Error: <num_files> must be a positive integer\n")
		os.Exit(1)
	}

	srcPath := os.Args[2]
	dstPath := os.Args[3]

	moved, err := MoveRandomFiles(maxFiles, srcPath, dstPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Moved %d file(s) from %q to %q\n", moved, srcPath, dstPath)
}

// MoveRandomFiles moves a random number (between 1 and maxFiles) of files
// from srcPath to dstPath and returns the number of files actually moved.
func MoveRandomFiles(maxFiles int, srcPath, dstPath string) (int, error) {
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return 0, fmt.Errorf("cannot read source directory %q: %w", srcPath, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}

	if len(files) == 0 {
		return 0, fmt.Errorf("no files found in source directory %q", srcPath)
	}

	if err := os.MkdirAll(dstPath, 0o755); err != nil {
		return 0, fmt.Errorf("cannot create destination directory %q: %w", dstPath, err)
	}

	limit := maxFiles
	if limit > len(files) {
		limit = len(files)
	}

	// Pick a random count between 1 and limit (inclusive).
	// Since Go 1.20 the global math/rand source is automatically seeded, so no
	// explicit seeding is required.
	count := rand.Intn(limit) + 1

	// Shuffle and take the first `count` files.
	rand.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
	selected := files[:count]

	for _, name := range selected {
		src := filepath.Join(srcPath, name)
		dst := filepath.Join(dstPath, name)
		if err := moveFile(src, dst); err != nil {
			return 0, fmt.Errorf("failed to move %q: %w", name, err)
		}
	}

	return count, nil
}

// moveFile moves a file from src to dst.
// It tries os.Rename first (fast, same filesystem); if that fails it falls
// back to a copy-then-delete approach (cross-filesystem support).
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %q: %w", src, err)
	}

	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %q: %w", src, err)
	}

	if err := os.WriteFile(dst, data, info.Mode()); err != nil {
		return fmt.Errorf("write %q: %w", dst, err)
	}

	return fmt.Errorf("remove source %q after copy: %w", src, os.Remove(src))
}
