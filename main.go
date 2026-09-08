package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	ext := flag.String("ext", "", "only move files with this extension (e.g. .txt)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <num_files> <source_path> <dest_path>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) != 3 {
		flag.Usage()
		os.Exit(1)
	}

	numFiles, err := strconv.Atoi(args[0])
	if err != nil || numFiles <= 0 {
		fmt.Fprintf(os.Stderr, "Error: <num_files> must be a positive integer\n")
		os.Exit(1)
	}

	srcPath := args[1]
	dstPath := args[2]

	moved, err := MoveRandomFiles(numFiles, srcPath, dstPath, *ext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Moved %d file(s) from %q to %q:\n", len(moved), srcPath, dstPath)
	for _, name := range moved {
		fmt.Printf("  %s\n", name)
	}
}

// MoveRandomFiles moves up to numFiles randomly selected files from srcPath
// to dstPath and returns the list of moved filenames. If ext is non-empty,
// only files with that extension are considered (e.g. ".txt").
// If the source directory contains fewer than numFiles matching files,
// all available files are moved.
func MoveRandomFiles(numFiles int, srcPath, dstPath, ext string) ([]string, error) {
	srcPath = filepath.Clean(srcPath)
	dstPath = filepath.Clean(dstPath)

	if srcPath == dstPath {
		return nil, fmt.Errorf("source and destination are the same directory %q", srcPath)
	}

	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read source directory %q: %w", srcPath, err)
	}

	ext = strings.ToLower(ext)
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			if ext == "" || strings.ToLower(filepath.Ext(e.Name())) == ext {
				files = append(files, e.Name())
			}
		}
	}

	if len(files) == 0 {
		if ext != "" {
			return nil, fmt.Errorf("no files matching extension %q in source directory %q", ext, srcPath)
		}
		return nil, fmt.Errorf("no files found in source directory %q", srcPath)
	}

	if numFiles > len(files) {
		numFiles = len(files)
	}

	if err := os.MkdirAll(dstPath, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create destination directory %q: %w", dstPath, err)
	}

	// Shuffle and take exactly numFiles files.
	rand.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
	selected := files[:numFiles]

	for _, name := range selected {
		src := filepath.Join(srcPath, name)
		dst := filepath.Join(dstPath, name)
		if err := moveFile(src, dst); err != nil {
			return nil, fmt.Errorf("failed to move %q: %w", name, err)
		}
	}

	return selected, nil
}

// moveFile moves a file from src to dst.
// It tries os.Rename first (fast, same filesystem); if that fails it falls
// back to a copy-then-delete approach (cross-filesystem support).
func moveFile(src, dst string) error {
	return moveFileWithRename(src, dst, os.Rename)
}

func moveFileWithRename(src, dst string, rename func(string, string) error) error {
	if err := rename(src, dst); err == nil {
		return nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %q: %w", src, err)
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat %q: %w", src, err)
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("create %q: %w", dst, err)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		os.Remove(dst)
		return fmt.Errorf("copy %q to %q: %w", src, dst, err)
	}

	if err := dstFile.Sync(); err != nil {
		dstFile.Close()
		os.Remove(dst)
		return fmt.Errorf("sync %q: %w", dst, err)
	}

	if err := dstFile.Close(); err != nil {
		os.Remove(dst)
		return fmt.Errorf("close %q: %w", dst, err)
	}

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("remove source %q after copy: %w", src, err)
	}

	return nil
}
