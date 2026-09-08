# rmove

A CLI utility that randomly selects and moves files from one directory to another.

## Usage

```
rmove [options] <num_files> <source_path> <dest_path>
```

### Arguments

| Argument | Description |
|---|---|
| `num_files` | Number of files to randomly select and move (positive integer) |
| `source_path` | Directory to read files from |
| `dest_path` | Directory to move files into (created if it doesn't exist) |

### Options

| Flag | Description |
|---|---|
| `-ext` | Only move files with this extension, e.g. `.txt` or `txt` (case-insensitive) |

### Examples

```bash
# Move 3 random files from ./photos to ./archive
rmove 3 ./photos ./archive

# Move 5 random .jpg files from ./camera to ./selected
rmove -ext .jpg 5 ./camera ./selected

# Move all .txt files (if fewer than requested)
rmove -ext txt 100 ./docs ./trash
```

## How it works

1. Reads the source directory and filters regular files (skips subdirectories)
2. If `-ext` is provided, filters to only matching extensions (case-insensitive)
3. Shuffles the file list and selects the requested number
4. Moves each file using `os.Rename` (fast, same filesystem) with a copy-then-delete fallback for cross-filesystem moves

## Build

```bash
make build
```

## Install

```bash
make install
```

## Test

```bash
make test
```
