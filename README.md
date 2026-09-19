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

# Link a cloned repository without changing the current directory
$ dotfileslinker --root /path/to/dotfiles

# use --force to overwrite destination files
$ dotfileslinker --force
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

2. Run the dotfileslinker command. The `--force` option is required to overwrite existing files.

```sh
$ dotfileslinker --force
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
| `--root PATH` | Directory containing dotfiles; takes precedence over `DOTFILES_ROOT` |
| `--force` | Overwrite existing files or directories |
| `--verbose`, `-v` | Display detailed information during execution |
| `--dry-run`, `-d` | Simulate operations without making any changes |

### Environment Variables

dotfiles can be configured using the following environment variables:

| Variable | Description | Default |
| --- | --- | --- |
| `DOTFILES_ROOT` | Root directory used when `--root` is omitted | Current directory |
| `DOTFILES_HOME` | User's home directory | User profile directory (`$HOME`) |
| `DOTFILES_IGNORE_FILE` | Name of the ignore file | `dotfiles_ignore` |

Example usage with environment variables:

```sh
# Specify a cloned dotfiles repository directly
dotfileslinker --root /path/to/my/dotfiles

# Alternatively, set a default dotfiles repository path
export DOTFILES_ROOT=/path/to/my/dotfiles

# Set custom home directory
export DOTFILES_HOME=/custom/home/path

# Run with custom settings
dotfileslinker --force
```

### Directory Links (`dotfiles_link_dirs`)

By default, directories are traversed and their files are linked individually. To link selected directories as a whole, create `dotfiles_link_dirs` in the repository root:

```gitignore
# Link each Codex skill directory, keeping SKILL.md as a regular file
HOME/.agents/skills/*

# Link a complete application configuration directory
HOME/.config/nvim
```

Patterns use the same gitignore-style syntax as `dotfiles_ignore`, including comments, wildcards, trailing `/`, and `!` exclusions, but a positive match **selects a directory for linking**. Paths are repository-relative and include `HOME/` or `ROOT/`. Only directories beneath those containers are eligible; files, repository-root dotfiles, and the `HOME`/`ROOT` containers themselves keep their existing behavior. `ROOT` is processed only on Linux/macOS.

- Each selected directory becomes one symlink at the corresponding destination. Its contents are not traversed; empty directories work too, and new source files appear through the link immediately.
- Default exclusions and `dotfiles_ignore` take precedence when evaluating the directory and its ancestors. Once a directory is selected, its **entire contents** are exposed through the link: ignore rules and directory-link exclusions for descendants cannot filter those contents. For selective contents, leave the parent unselected and select individual child directories instead.
- A missing or empty `dotfiles_link_dirs` preserves the default file-by-file behavior. The configuration file is not linked to the home directory.
- An existing link to the same directory is skipped. Converting an existing real directory (including one containing file symlinks) requires `--force`. As with other forced replacements, the old destination and its contents are removed after the new link is created; preserve any local-only files first.

Preview the migration, then apply it:

```shell
dotfileslinker --root /path/to/dotfiles --dry-run --force
dotfileslinker --root /path/to/dotfiles --force
```

`--dry-run` reports directory symlinks without changing the filesystem. Both the C# and Go implementations use this configuration format.

### dotfiles_ignore File

You can specify files or directories to be excluded from linking in the `dotfiles_ignore` file. Rules use gitignore-style syntax and paths are relative to the dotfiles repository root.

```
# Example dotfiles_ignore
.git
.github
README.md
LICENSE
```

#### Gitignore-style Rules

Patterns are evaluated from top to bottom, and the last matching pattern decides whether a path is ignored. Blank lines and lines beginning with `#` are ignored.

```
# A name without `/` matches at any depth
.github
README.md
LICENSE

# Wildcards
# `*` matches any string (excluding path separators)
# `?` matches any single character
# `[a-z]` matches one character in a range
*.log
temp*
backup.???
file[0-9].txt

# A pattern containing `/` is relative to the repository root
# A leading `/` explicitly anchors a pattern to the repository root
# `**` matches any number of directories (including zero)
# A pattern ending with `/` matches directories only
docs/build/
/config/local_*.json
HOME/**/*.log
**/temp/

# Negation patterns
# A pattern starting with `!` explicitly includes files that would otherwise be ignored
# Exclude all .log files except important.log
*.log
!important.log

# Exclude everything in docs except README.md
docs/
!docs/
docs/*
!docs/README.md

# Escape a leading # or ! to match it literally
\#notes.txt
\!important.txt
```

As with Git, a file cannot be re-included while one of its parent directories remains excluded. Re-include the parent directory first, as shown in the `docs/README.md` example. Built-in automatic exclusions cannot be overridden by negation rules.

#### Compatibility with `.gitignore`

`dotfiles_ignore` implements a practical subset of the [Git ignore pattern format](https://git-scm.com/docs/gitignore), but it is not a drop-in replacement for Git's complete ignore mechanism.

Supported behavior:

| Feature | Support |
| --- | --- |
| Empty lines and comments beginning with `#` | Supported |
| Escaped leading `\#` and `\!` | Supported |
| Unescaped trailing spaces are ignored | Supported |
| Escaped trailing spaces are significant | Supported |
| Last matching rule wins | Supported |
| Negation with `!` | Supported, including Git's excluded-parent restriction |
| Patterns without `/` | Match a file or directory name at any depth |
| Leading `/` and patterns containing `/` | Match relative to the dotfiles repository root |
| Trailing `/` | Matches directories and their descendants only |
| `*` and `?` | Supported within one path segment |
| Simple character classes such as `[abc]`, `[0-9]`, `[!abc]`, and `[^abc]` | Supported |
| `**/name`, `dir/**`, and `a/**/b` | Supported |
| Backslash escapes such as `file\*.txt` | Supported within a path segment |

Differences and unsupported behavior:

| Git behavior | DotfilesLinker behavior |
| --- | --- |
| Git combines repository `.gitignore` files, nested `.gitignore` files, `.git/info/exclude`, a global excludes file, and command-line rules | Only the configured `DOTFILES_IGNORE_FILE` is read once from the repository root; the default filename is `dotfiles_ignore` |
| A nested `.gitignore` uses its containing directory as the pattern base | Nested ignore files are not discovered; all slash-containing patterns use the dotfiles repository root as their base |
| Case sensitivity follows Git/filesystem configuration such as `core.ignoreCase` | Matching is always case-insensitive on every platform |
| Git uses its complete wildmatch/fnmatch character-class behavior | POSIX classes such as `[[:digit:]]`, collating/equivalence classes, and uncommon literal `]` class forms are not supported |
| Git defines only specific placements of consecutive `**` as special | Only `**` used as a complete path segment in the documented forms is Git-compatible; other consecutive-star forms are not validated and may behave like `*` |
| Git ignore rules affect untracked-file discovery and interact with Git's index | DotfilesLinker has no tracked/untracked concept; rules only decide which repository files are considered for linking |
| Git has no mandatory built-in ignore patterns | DotfilesLinker's automatic exclusions are applied separately and cannot be re-included with `!` |

### Automatic Exclusions

The following files and directories are automatically excluded:
- Version control system folders/files (`.git`, `.svn`, `.hg`)
- Non-dotfiles in the root directory
- OS-specific files like `.DS_Store` (macOS) and `Thumbs.db` (Windows)
- Temporary files like `*.bak`, `*.tmp`, and vim swap files

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

### Attested Releases and SBOMs

Each release archive has GitHub build-provenance and SBOM attestations. The release also contains an artifact-specific SPDX JSON SBOM alongside every archive.

Verify that an archive was produced by this repository's release workflow:

```bash
gh attestation verify dotfileslinker_x.y.z_windows_amd64.zip -R guitarrapc/dotfileslinker-go
```

Verify and inspect its SPDX SBOM attestation:

```bash
gh attestation verify dotfileslinker_x.y.z_windows_amd64.zip -R guitarrapc/dotfileslinker-go --predicate-type https://spdx.dev/Document/v2.3 --format json --jq '.[].verificationResult.statement.predicate'
```

### If Problems Persist

- Try the latest version as build configurations may have improved
- Report issues on the repository's issue page

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details.
