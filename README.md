[![Build and Test](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/build.yaml/badge.svg)](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/build.yaml)
[![Release](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/release.yaml/badge.svg)](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/release.yaml)

[日本語](README_ja.md)

# DotfilesLinker (Go Version)

Fast Go utility to create symbolic links from dotfiles to your home directory. This is a port of the original [DotfilesLinker](https://github.com/guitarrapc/DotfilesLinker) written in C# NativeAOT. Supports Windows, Linux, and macOS while respecting your dotfiles repository structure. It's implemented in pure Go and distributed as a statically linked single binary without any dependency on external libraries like libc.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
# Table of Contents

- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Windows Security Notes](#windows-security-notes)
- [License](#license)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Quick Start

1. Download the latest binary from the [GitHub Releases page](https://github.com/guitarrapc/dotfileslinker-go/releases/latest) and place it in a directory that is in your PATH.
2. Run executable file `dotfileslinker` in your terminal.

```sh
# Safe mode, do not overwrite existing files
$ dotfileslinker

# use --force=y to overwrite destination files
$ dotfileslinker --force=y
```

## How It Works

dotfileslinker creates symbolic links based on your dotfiles repository structure:

- Dotfiles in the root directory → linked to `$HOME`
- Files in the `HOME` directory → linked to the corresponding path in `$HOME`
- Files in the `ROOT` directory → linked to the corresponding path in the root directory (`/`) (Linux and macOS only)

## Installation

### Scoop (Windows)

Install DotfilesLinker using [Scoop](https://scoop.sh/):

```sh
$ scoop bucket add guitarrapc https://github.com/guitarrapc/scoop-bucket.git
$ scoop install dotfileslinker-go
```

### Download Binary

Download the latest binary from the [GitHub Releases page](https://github.com/guitarrapc/dotfileslinker-go/releases) and place it in a directory that is in your PATH.

Available platforms:
- Windows (x64, ARM64)
- Linux (x64, ARM64)
- macOS (x64, ARM64)

### Build from Source

```bash
git clone https://github.com/guitarrapc/dotfileslinker-go.git
cd dotfileslinker-go
go build ./cmd/dotfileslinker
go test ./...
golangci-lint run
```

## Usage

1. Prepare your dotfiles repository structure as shown below.

<details><summary>Linux example</summary>

```sh
dotfiles
├─.bashrc_custom             # link to $HOME/.bashrc_custom
├─.gitignore_global          # link to $HOME/.gitignore_global
├─.gitconfig                 # link to $HOME/.gitconfig
├─aqua.yaml                  # non-dotfiles file automatically ignore
├─dotfiles_ignore            # ignore list for dotfiles link
├─.github
│  └─workflows               # automatically ignore
├─HOME
│  ├─.config
│  │  └─aquaproj-aqua
│  │     └─aqua.yaml         # link to $HOME/.config/aquaproj-aqua/aqua.yaml
│  └─.ssh
│     └─config               # link to $HOME/.ssh/config
└─ROOT
    └─etc
        └─profile.d
           └─profile_foo.sh  # link to /etc/profile.d/profile_foo.sh
```

</details>

<details><summary>Windows example</summary>

```sh
dotfiles
├─dotfiles_ignore            # ignore list for dotfiles link
├─.gitignore_global          # link to $HOME/.gitignore_global
├─.gitconfig                 # link to $HOME/.gitconfig
├─.textlintrc.json           # link to $HOME/.textlintrc.json
├─.wslconfig                 # link to $HOME/.wslconfig
├─aqua.yaml                  # non-dotfiles file automatically ignore
├─.github
│  └─workflows               # automatically ignore
└─HOME
    ├─.config
    │  └─git
    │     └─config           # link to $HOME/.config/git/config
    │     └─ignore           # link to $HOME/.config/git/ignore
    ├─.ssh
    │  ├─config              # link to $HOME/.ssh/config
    │  └─conf.d
    │     └─github           # link to $HOME/.ssh/conf.d/github
    └─AppData
       ├─Local
       │  └─Packages
       │      └─Microsoft.WindowsTerminal_8wekyb3d8bbwe
       │          └─LocalState
       │              └─settings.json   # link to $HOME/AppData/Local/Packages/Microsoft.WindowsTerminal_8wekyb3d8bbwe/LocalState/settings.json
       └─Roaming
           └─Code
               └─User
                  └─settings.json   # link to $HOME/AppData/Roaming/Code/User/settings.json
```

</details>

2. Run the dotfileslinker command. The `--force=y` option is required to overwrite existing files.

```sh
$ dotfileslinker --force=y
[o] Skipping already linked: /home/user/.bashrc_custom -> /home/user/dotfiles/.bashrc_custom
[o] Skipping already linked: /home/user/.gitconfig -> /home/user/dotfiles/.gitconfig
[o] Creating symbolic link: /home/user/.gitignore_global -> /home/user/dotfiles/.gitignore_global
[o] Creating symbolic link: /home/user/.config/aquaproj-aqua/aqua.yaml -> /home/user/dotfiles/HOME/.config/aquaproj-aqua/aqua.yaml
[o] Creating symbolic link: /home/user/.ssh/config -> /home/user/dotfiles/HOME/.ssh/config
[o] All operations completed.
```

3. Verify the symbolic links created by dotfileslinker.

```sh
$ ls -la $HOME
total 24
drwxr-x--- 5 user user 4096 Apr 21 10:30 .
drwxr-xr-x 3 root root 4096 Apr 21 10:00 ..
lrwxrwxrwx 1 user user   45 Apr 21 10:30 .bashrc_custom -> /home/user/dotfiles/.bashrc_custom
lrwxrwxrwx 1 user user   41 Apr 21 10:30 .gitconfig -> /home/user/dotfiles/.gitconfig
lrwxrwxrwx 1 user user   48 Apr 21 10:30 .gitignore_global -> /home/user/dotfiles/.gitignore_global
drwxr-xr-x 3 user user 4096 Apr 21 10:30 .config
drwxr-xr-x 2 user user 4096 Apr 21 10:30 .ssh
```

4. Run the following command to see all available options:

```bash
dotfileslinker --help
```

## Configuration

### Command Options

All options are optional. The default behavior is to create symbolic links for all dotfiles in the repository.

| Option | Description |
| --- | --- |
| `--help`, `-h` | Display help information |
| `--version` | Display version information |
| `--force=y` | Overwrite existing files or directories |
| `--verbose`, `-v` | Display detailed information during execution |
| `--dry-run`, `-d` | Simulate operations without making any changes |

### Environment Variables

dotfiles can be configured using the following environment variables:

| Variable | Description | Default |
| --- | --- | --- |
| `DOTFILES_ROOT` | Root directory of your dotfiles repository | Current directory |
| `DOTFILES_HOME` | User's home directory | User profile directory (`$HOME`) |
| `DOTFILES_IGNORE_FILE` | Name of the ignore file | `dotfiles_ignore` |

Example usage with environment variables:

```sh
# Set custom dotfiles repository path
export DOTFILES_ROOT=/path/to/my/dotfiles

# Set custom home directory
export DOTFILES_HOME=/custom/home/path

# Run with custom settings
dotfileslinker --force=y
```

### dotfiles_ignore File

You can specify files or directories to be excluded from linking in the `dotfiles_ignore` file:

```
# Example dotfiles_ignore
.git
.github
README.md
LICENSE
```

#### Supported Pattern Types

DotfilesLinker supports the following pattern types in the `dotfiles_ignore` file:

```
# Simple filenames or paths that match exactly
.github
README.md
LICENSE

# Wildcard patterns
# `*` matches any string (excluding path separators)
# `?` matches any single character
*.log
temp*
backup.???

# Gitignore-style patterns
# A pattern containing `/` matches a specific path from the repository root
# `**` matches any number of directories (including zero)
# A pattern ending with `/` matches directories only
docs/build/
config/local_*.json
HOME/**.log
**/temp/

# Negation patterns
# A pattern starting with `!` explicitly includes files that would otherwise be ignored
# Processed after non-negated patterns
# --------------------------
# Patterns are processed in two stages:
# 1. First, all non-negation patterns are evaluated
# 2. Then, negation patterns (`!`) are applied and can override previous exclusions
## Exclude all .log files except important.log
*.log
!important.log
## Exclude everything in docs except README.md
docs/
!docs/README.md
```

### Automatic Exclusions

The following files and directories are automatically excluded:
- Directories starting with `.git` (like `.github`)
- Non-dotfiles in the root directory

## Windows Security Notes

Windows Defender or other antivirus software may flag Go executables as suspicious. This is a common false positive for Go applications.

### Verifying Binary Integrity

To verify the integrity of the downloaded binary:

1. Download the `checksums.txt` file from the release page
2. Calculate the hash of the downloaded zip file:
   ```
   certutil -hashfile dotfileslinker_x.y.z_windows_amd64.zip SHA256
   ```
3. Compare the calculated hash with the value in `checksums.txt`

### Signed Releases

Starting from v0.2.1, release binaries are signed with Cosign. You can verify the signature if you have Cosign installed:

```bash
# Verify the checksums file signature
cosign verify-blob --signature checksums.txt.sig checksums.txt
```

### If Problems Persist

- Try the latest version as build configurations may have improved
- Report issues on the repository's issue page

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details.
