[![Build and Test](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/build.yml/badge.svg)](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/build.yml)
[![Release](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/release.yml/badge.svg)](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/release.yml)

[English](README.md)

# DotfilesLinker (Go版)

DotfilesLinker は、dotfiles リポジトリからホームディレクトリにシンボリックリンクを作成するためのシンプルなツールです。この Go 版は、オリジナルの C# NativeAOT版 [DotfilesLinker](https://github.com/guitarrapc/DotfilesLinker) の移植版です。

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
# Table of Contents

- [機能](#%E6%A9%9F%E8%83%BD)
- [インストール](#%E3%82%A4%E3%83%B3%E3%82%B9%E3%83%88%E3%83%BC%E3%83%AB)
- [使い方](#%E4%BD%BF%E3%81%84%E6%96%B9)
- [ディレクトリ構造](#%E3%83%87%E3%82%A3%E3%83%AC%E3%82%AF%E3%83%88%E3%83%AA%E6%A7%8B%E9%80%A0)
- [dotfiles_ignore ファイル](#dotfiles_ignore-%E3%83%95%E3%82%A1%E3%82%A4%E3%83%AB)
- [ライセンス](#%E3%83%A9%E3%82%A4%E3%82%BB%E3%83%B3%E3%82%B9)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## 機能

- リポジトリルートの `.` で始まるファイルをホームディレクトリに自動的にリンク
- `HOME/` ディレクトリ内のファイルを `$HOME` の同じ相対パスにリンク
- `ROOT/` ディレクトリ内のファイルをルートディレクトリ（`/`）の同じ相対パスにリンク（Linux/macOS のみ）
- 既存のファイルやディレクトリを上書きするオプション
- 詳細なログ出力オプション
- `dotfiles_ignore` ファイルを使用した特定のファイルの除外

## インストール

### バイナリをダウンロード

[GitHub Releases](https://github.com/guitarrapc/dotfileslinker-go/releases) から、お使いのプラットフォーム用のバイナリをダウンロードしてください。

### ソースからビルド

```bash
git clone https://github.com/guitarrapc/dotfileslinker-go.git
cd dotfileslinker-go
go build ./cmd/dotfileslinker
```

## 使い方

### 基本的な使い方

リポジトリのルートディレクトリで実行するだけです：

```bash
dotfileslinker
```

### コマンドラインオプション

```
Dotfiles Linker - A utility to link dotfiles from a repository to your home directory

Usage: dotfileslinker [options]

Options:
  --help, -h         ヘルプメッセージを表示
  --force=y          既存のファイルやディレクトリを上書き
  --verbose, -v      実行中に詳細情報を表示
  --version          バージョン情報を表示
```

### 環境変数

- `DOTFILES_ROOT` - dotfiles を含むディレクトリ（デフォルト: カレントディレクトリ）
- `DOTFILES_HOME` - ターゲットのホームディレクトリ（デフォルト: ユーザーのホームディレクトリ）
- `DOTFILES_IGNORE_FILE` - 除外パターンを含むファイルの名前（デフォルト: `dotfiles_ignore`）

## ディレクトリ構造

DotfilesLinker は以下のようなディレクトリ構造を想定しています：

```
dotfiles/                 # dotfiles リポジトリのルート
├── .gitconfig            # ホームディレクトリにリンク
├── .bashrc               # ホームディレクトリにリンク
├── dotfiles_ignore       # リンクから除外するファイルのリスト
├── HOME/                 # $HOME ディレクトリ構造
│   ├── .config/          # $HOME/.config にリンク
│   │   └── nvim/
│   │       └── init.vim  # $HOME/.config/nvim/init.vim にリンク
│   └── bin/
│       └── script.sh     # $HOME/bin/script.sh にリンク
└── ROOT/                 # ルートディレクトリ構造 (Linux/macOS のみ)
    └── etc/
        └── hosts         # /etc/hosts にリンク（管理者権限が必要）
```

## dotfiles_ignore ファイル

`dotfiles_ignore` ファイルには、リンクしたくないファイル名を1行に1つずつリストアップします：

```
LICENSE
README.md
README_ja.md
dotfiles_ignore
.git
.github
```

## ライセンス

このプロジェクトは MIT ライセンスの下で公開されています。詳細は [LICENSE](LICENSE) ファイルを参照してください。
