[![Build and Test](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/build.yaml/badge.svg)](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/build.yaml)
[![Release](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/release.yaml/badge.svg)](https://github.com/guitarrapc/dotfileslinker-go/actions/workflows/release.yaml)

[English](README.md)

# DotfilesLinker (Go版)

Go言語で実装された高速な dotfiles シンボリックリンク作成ツール。これは C# NativeAOT版 [DotfilesLinker](https://github.com/guitarrapc/DotfilesLinker) の移植版です。Windows、Linux、macOSに対応し、dotfilesリポジトリの構造を尊重します。純粋なGoで実装されており、libcなどの外部ライブラリに依存しない静的リンクされたシングルバイナリです。

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
# Table of Contents

- [クイックスタート](#%E3%82%AF%E3%82%A4%E3%83%83%E3%82%AF%E3%82%B9%E3%82%BF%E3%83%BC%E3%83%88)
- [動作原理](#%E5%8B%95%E4%BD%9C%E5%8E%9F%E7%90%86)
- [インストール方法](#%E3%82%A4%E3%83%B3%E3%82%B9%E3%83%88%E3%83%BC%E3%83%AB%E6%96%B9%E6%B3%95)
- [使い方](#%E4%BD%BF%E3%81%84%E6%96%B9)
- [設定](#%E8%A8%AD%E5%AE%9A)
- [Windowsセキュリティについて](#windows%E3%82%BB%E3%82%AD%E3%83%A5%E3%83%AA%E3%83%86%E3%82%A3%E3%81%AB%E3%81%A4%E3%81%84%E3%81%A6)
- [ライセンス](#%E3%83%A9%E3%82%A4%E3%82%BB%E3%83%B3%E3%82%B9)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## クイックスタート

1. [GitHubリリースページ](https://github.com/guitarrapc/dotfileslinker-go/releases/latest)から最新のバイナリをダウンロードし、PATHの通ったディレクトリに配置します。
2. ターミナルで実行ファイル `dotfileslinker` を実行します。

```sh
# 安全モード、既存ファイルを上書きしません
$ dotfileslinker

# --force=y オプションで既存ファイルを上書き
$ dotfileslinker --force=y
```

## 動作原理

dotfileslinkerは、dotfilesリポジトリの構造に基づいてシンボリックリンクを作成します：

- ルートディレクトリのドットファイル → `$HOME` にリンク
- `HOME` ディレクトリ内のファイル → `$HOME` の対応するパスにリンク
- `ROOT` ディレクトリ内のファイル → ルートディレクトリ（`/`）の対応するパスにリンク（LinuxとmacOSのみ）

## インストール方法

### バイナリをダウンロード

[GitHubリリースページ](https://github.com/guitarrapc/dotfileslinker-go/releases)から最新のバイナリをダウンロードし、PATHの通ったディレクトリに配置してください。

対応プラットフォーム:
- Windows (x64, ARM64)
- Linux (x64, ARM64)
- macOS (x64, ARM64)

### ソースからビルド

```bash
git clone https://github.com/guitarrapc/dotfileslinker-go.git
cd dotfileslinker-go
go build ./cmd/dotfileslinker
```

## 使い方

1. 下記のようなdotfilesリポジトリの構造を準備します。

<details><summary>Linux の例</summary>

```sh
dotfiles
├─.bashrc_custom             # $HOME/.bashrc_customへリンク
├─.gitignore_global          # $HOME/.gitignore_globalへリンク
├─.gitconfig                 # $HOME/.gitconfigへリンク
├─aqua.yaml                  # ドットファイルでないため自動的に除外
├─dotfiles_ignore            # dotfilesリンク用除外リスト
├─.github
│  └─workflows               # 自動的に除外
├─HOME
│  ├─.config
│  │  └─aquaproj-aqua
│  │     └─aqua.yaml         # $HOME/.config/aquaproj-aqua/aqua.yamlへリンク
│  └─.ssh
│     └─config               # $HOME/.ssh/configへリンク
└─ROOT
    └─etc
        └─profile.d
           └─profile_foo.sh  # /etc/profile.d/profile_foo.shへリンク
```

</details>

<details><summary>Windows の例</summary>

```sh
dotfiles
├─dotfiles_ignore            # dotfilesリンク用除外リスト
├─.gitignore_global          # $HOME/.gitignore_globalへリンク
├─.gitconfig                 # $HOME/.gitconfigへリンク
├─.textlintrc.json           # $HOME/.textlintrc.jsonへリンク
├─.wslconfig                 # $HOME/.wslconfigへリンク
├─aqua.yaml                  # ドットファイルでないため自動的に除外
├─.github
│  └─workflows               # 自動的に除外
└─HOME
    ├─.config
    │  └─git
    │     └─config           # $HOME/.config/git/configへリンク
    │     └─ignore           # $HOME/.config/git/ignoreへリンク
    ├─.ssh
    │  ├─config              # $HOME/.ssh/configへリンク
    │  └─conf.d
    │     └─github           # $HOME/.ssh/conf.d/githubへリンク
    └─AppData
       ├─Local
       │  └─Packages
       │      └─Microsoft.WindowsTerminal_8wekyb3d8bbwe
       │          └─LocalState
       │              └─settings.json   # $HOME/AppData/Local/Packages/Microsoft.WindowsTerminal_8wekyb3d8bbwe/LocalState/settings.jsonへリンク
       └─Roaming
           └─Code
               └─User
                  └─settings.json   # $HOME/AppData/Roaming/Code/User/settings.jsonへリンク
```

</details>

2. dotfileslinkerコマンドを実行します。既存のファイルを上書きするには `--force=y` オプションが必要です。

```sh
$ dotfileslinker --force=y
[o] Skipping already linked: /home/user/.bashrc_custom -> /home/user/dotfiles/.bashrc_custom
[o] Skipping already linked: /home/user/.gitconfig -> /home/user/dotfiles/.gitconfig
[o] Creating symbolic link: /home/user/.gitignore_global -> /home/user/dotfiles/.gitignore_global
[o] Creating symbolic link: /home/user/.config/aquaproj-aqua/aqua.yaml -> /home/user/dotfiles/HOME/.config/aquaproj-aqua/aqua.yaml
[o] Creating symbolic link: /home/user/.ssh/config -> /home/user/dotfiles/HOME/.ssh/config
[o] All operations completed.
```

3. DotfilesLinkerによって作成されたシンボリックリンクを確認します。

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

4. 利用可能なすべてのオプションを表示するには、以下のコマンドを実行します：

```bash
dotfileslinker --help
```

## 設定

### コマンドオプション

すべてのオプションは任意です。デフォルトでは、リポジトリ内のすべてのドットファイルに対してシンボリックリンクを作成します。

| オプション | 説明 |
| --- | --- |
| `--help`, `-h` | ヘルプ情報を表示 |
| `--version` | バージョン情報を表示 |
| `--force=y` | 既存のファイルやディレクトリを上書き |
| `--verbose`, `-v` | 実行中の詳細情報を表示 |

### 環境変数

dotfileslinkerは以下の環境変数で設定をカスタマイズできます：

| 変数 | 説明 | デフォルト値 |
| --- | --- | --- |
| `DOTFILES_ROOT` | dotfilesリポジトリのルートディレクトリ | カレントディレクトリ |
| `DOTFILES_HOME` | ユーザーのホームディレクトリ | ユーザープロファイルディレクトリ（`$HOME`） |
| `DOTFILES_IGNORE_FILE` | 除外ファイルの名前 | `dotfiles_ignore` |

環境変数を使用する例：

```sh
# カスタムdotfilesリポジトリのパスを設定
export DOTFILES_ROOT=/path/to/my/dotfiles

# カスタムホームディレクトリを設定
export DOTFILES_HOME=/custom/home/path

# カスタム設定で実行
dotfileslinker --force=y
```

### dotfiles_ignore ファイル

`dotfiles_ignore` ファイルを使用して、リンク作成から除外するファイルやディレクトリを指定できます：

```
# dotfiles_ignore の例
.git
.github
README.md
LICENSE
```

### 自動除外

以下のファイルやディレクトリは自動的に除外されます：
- `.git` で始まるディレクトリ（`.github` など）
- ルートディレクトリの非ドットファイル（先頭が `.` でないファイル）

## Windowsセキュリティについて

Windows環境でdotfileslinkerを使用する際には、セキュリティ設定に注意してください。特に、シンボリックリンクを作成するためには管理者権限が必要です。以下の手順でセキュリティ設定を確認し、必要に応じて変更してください。

1. 管理者権限でコマンドプロンプトを開きます。
2. 以下のコマンドを実行して、シンボリックリンクの作成が許可されているか確認します。

```sh
fsutil behavior query SymlinkEvaluation
```

3. 出力結果に `Local to local symbolic links are enabled` が含まれていることを確認します。含まれていない場合は、以下のコマンドを実行して有効にします。

```sh
fsutil behavior set SymlinkEvaluation L2L:1
```

4. 必要に応じて、他のシンボリックリンク設定も有効にします。

```sh
fsutil behavior set SymlinkEvaluation L2R:1
fsutil behavior set SymlinkEvaluation R2R:1
fsutil behavior set SymlinkEvaluation R2L:1
```

Windows DefenderなどのアンチウイルスソフトウェアがGoのバイナリを不審なファイルとして検出する場合があります。これはGo言語で作成されたアプリケーションでは一般的な誤検出です。

### バイナリの整合性検証

ダウンロードしたバイナリの整合性を検証するには以下の手順に従ってください：

1. リリースページから `checksums.txt` ファイルをダウンロードします
2. ダウンロードしたzipファイルのハッシュ値を計算します：
   ```
   certutil -hashfile dotfileslinker_x.y.z_windows_amd64.zip SHA256
   ```
3. 計算されたハッシュ値と `checksums.txt` の値を比較します

### 署名済みリリース

v0.2.1以降、リリースバイナリはCosignで署名されています。Cosignがインストールされている場合、次のコマンドで署名を検証できます：

```bash
# checksumファイルの署名を検証
cosign verify-blob --signature checksums.txt.sig checksums.txt
```

### 問題が解決しない場合

- 最新バージョンを試してください。ビルド設定が改善されている可能性があります
- リポジトリのIssueページで問題を報告してください

## ライセンス

このプロジェクトは MIT ライセンスの下で公開されています。詳細は [LICENSE](LICENSE.md) ファイルを参照してください。
