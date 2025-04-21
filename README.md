[![Build and Test](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/build.yml/badge.svg)](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/build.yml)
[![Release](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/release.yml/badge.svg)](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/release.yml)

[日本語](README_ja.md)

# DotfilesLinker (Go Version)

DotfilesLinker is a simple tool for creating symbolic links from your dotfiles repository to your home directory. This Go version is a port of the original [DotfilesLinker](https://github.com/guitarrapc/DotfilesLinker) written in C# NativeAOT.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Features

- Automatically link files starting with `.` in the repository root to your home directory
- Link files in the `HOME/` directory to the same relative path in `$HOME`
- Link files in the `ROOT/` directory to the same relative path in the root directory (`/`) (Linux/macOS only)
- Option to overwrite existing files or directories
- Verbose logging option
- Exclude specific files using a `dotfiles_ignore` file

## Installation

### Download Binary

Download the appropriate binary for your platform from [GitHub Releases](https://github.com/guitarrapc/dotfileslinker-go/releases).

### Build from Source

```bash
git clone https://github.com/guitarrapc/dotfileslinker-go.git
cd dotfileslinker-go
go build ./cmd/dotfileslinker
```

## Usage

### Basic Usage

Simply run it in the root directory of your dotfiles repository:

```bash
dotfileslinker
```

### Command Line Options

```
Dotfiles Linker - A utility to link dotfiles from a repository to your home directory

Usage: dotfileslinker [options]

Options:
  --help, -h         Display help message
  --force=y          Overwrite existing files or directories
  --verbose, -v      Display detailed information during execution
  --version          Display version information
```

### Environment Variables

- `DOTFILES_ROOT` - Directory containing dotfiles (default: current directory)
- `DOTFILES_HOME` - Target home directory (default: user's home directory)
- `DOTFILES_IGNORE_FILE` - Name of ignore file (default: `dotfiles_ignore`)

## Directory Structure

DotfilesLinker expects the following directory structure:

```
dotfiles/                 # Root of dotfiles repository
├── .gitconfig            # Will be linked to home directory
├── .bashrc               # Will be linked to home directory
├── dotfiles_ignore       # List of files to exclude from linking
├── HOME/                 # $HOME directory structure
│   ├── .config/          # Will be linked to $HOME/.config
│   │   └── nvim/
│   │       └── init.vim  # Will be linked to $HOME/.config/nvim/init.vim
│   └── bin/
│       └── script.sh     # Will be linked to $HOME/bin/script.sh
└── ROOT/                 # Root directory structure (Linux/macOS only)
    └── etc/
        └── hosts         # Will be linked to /etc/hosts (requires admin privileges)
```

## dotfiles_ignore File

The `dotfiles_ignore` file should list filenames that you want to exclude from linking, one per line:

```
LICENSE
README.md
README_ja.md
dotfiles_ignore
.git
.github
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
