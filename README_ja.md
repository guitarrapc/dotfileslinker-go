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

# カレントディレクトリを移動せず、clone済みリポジトリを指定
$ dotfileslinker --root /path/to/dotfiles

# --force オプションで既存ファイルを上書き
$ dotfileslinker --force
```

## 動作原理

dotfileslinkerは、dotfilesリポジトリの構造に基づいてシンボリックリンクを作成します：

- ルートディレクトリのドットファイル → `$HOME` にリンク
- `HOME` ディレクトリ内のファイル → `$HOME` の対応するパスにリンク
- `ROOT` ディレクトリ内のファイル → ルートディレクトリ（`/`）の対応するパスにリンク（LinuxとmacOSのみ）

## インストール方法

### Scoop (Windows)

[Scoop](https://scoop.sh/)を使用してDotfilesLinkerをインストールできます：

```sh
$ scoop bucket add guitarrapc https://github.com/guitarrapc/scoop-bucket.git
$ scoop install dotfileslinker-go
```

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
go test ./...
golangci-lint run
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

2. dotfileslinkerコマンドを実行します。既存のファイルを上書きするには `--force` オプションが必要です。

```sh
$ dotfileslinker --force
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
| `--root PATH` | dotfilesリポジトリのディレクトリ。`DOTFILES_ROOT`より優先 |
| `--force` | 既存のファイルやディレクトリを上書き |
| `--verbose`, `-v` | 実行中の詳細情報を表示 |
| `--dry-run`, `-d` | 実際に変更を加えずに操作をシミュレーション |

### 環境変数

dotfileslinkerは以下の環境変数で設定をカスタマイズできます：

| 変数 | 説明 | デフォルト値 |
| --- | --- | --- |
| `DOTFILES_ROOT` | `--root`省略時に使うdotfilesリポジトリのルートディレクトリ | カレントディレクトリ |
| `DOTFILES_HOME` | ユーザーのホームディレクトリ | ユーザープロファイルディレクトリ（`$HOME`） |
| `DOTFILES_IGNORE_FILE` | 除外ファイルの名前 | `dotfiles_ignore` |

環境変数を使用する例：

```sh
# clone済みdotfilesリポジトリを直接指定
dotfileslinker --root /path/to/my/dotfiles

# または既定のdotfilesリポジトリを環境変数で設定
export DOTFILES_ROOT=/path/to/my/dotfiles

# カスタムホームディレクトリを設定
export DOTFILES_HOME=/custom/home/path

# カスタム設定で実行
dotfileslinker --force
```

### dotfiles_ignore ファイル

`dotfiles_ignore` ファイルを使用して、リンク作成から除外するファイルやディレクトリを指定できます。ルールはgitignore形式で記述し、パスはdotfilesリポジトリのルートを基準にします。

```
# dotfiles_ignore の例
.git
.github
README.md
LICENSE
```

#### Gitignore形式のルール

パターンは上から順に評価され、最後に一致したパターンが除外するかどうかを決定します。空行と `#` で始まる行は無視されます。

```
# `/` を含まない名前は任意の階層で一致
.github
README.md
LICENSE

# ワイルドカード
# `*`: 任意の文字列（パス区切り文字を除く）にマッチ
# `?`: 任意の1文字にマッチ
# `[a-z]`: 範囲内の任意の1文字にマッチ
*.log
temp*
backup.???
file[0-9].txt

# `/` を含むパターンはリポジトリルート基準
# 先頭の `/` はリポジトリルートへの明示的なアンカー
# `**`: 任意の数のディレクトリ（ゼロを含む）にマッチ
# 末尾が `/` のパターンはディレクトリのみにマッチ
docs/build/
/config/local_*.json
HOME/**/*.log
**/temp/

# 否定パターン
# `!`で始まるパターンで、通常なら無視されるファイルを明示的に含める
# important.log以外のすべての.logファイルを除外
*.log
!important.log

# docs内のREADME.md以外のすべてを除外
docs/
!docs/
docs/*
!docs/README.md

# 先頭の # または ! を文字として扱う場合はエスケープする
\#notes.txt
\!important.txt
```

Gitと同様に、親ディレクトリが除外されたままでは、その配下のファイルを再包含できません。`docs/README.md` の例のように、先に親ディレクトリを再包含してください。組み込みの自動除外は否定ルールで解除できません。

#### `.gitignore` との互換性

`dotfiles_ignore` は[Gitのignoreパターン形式](https://git-scm.com/docs/gitignore)の実用的なサブセットを実装していますが、Gitのignore機構全体を置き換えるものではありません。

対応している挙動：

| 機能 | 対応状況 |
| --- | --- |
| 空行と `#` で始まるコメント | 対応 |
| 先頭の `\#` と `\!` のエスケープ | 対応 |
| エスケープされていない末尾スペースの無視 | 対応 |
| エスケープされた末尾スペースの文字としての扱い | 対応 |
| 最後に一致したルールを優先 | 対応 |
| `!` による否定 | 親ディレクトリ除外時の制約を含めて対応 |
| `/` を含まないパターン | 任意の階層にあるファイル名またはディレクトリ名に一致 |
| 先頭の `/` と `/` を含むパターン | dotfilesリポジトリのルートを基準に一致 |
| 末尾の `/` | ディレクトリとその配下だけに一致 |
| `*` と `?` | 1つのパス要素内で対応 |
| `[abc]`、`[0-9]`、`[!abc]`、`[^abc]` などの単純な文字クラス | 対応 |
| `**/name`、`dir/**`、`a/**/b` | 対応 |
| `file\*.txt` などのバックスラッシュエスケープ | 1つのパス要素内で対応 |

Gitとの差異・未対応の挙動：

| Gitの挙動 | DotfilesLinkerの挙動 |
| --- | --- |
| リポジトリおよび各階層の `.gitignore`、`.git/info/exclude`、グローバル除外ファイル、コマンドラインルールを統合 | リポジトリルートにある `DOTFILES_IGNORE_FILE` 指定のファイルを1つだけ読み込み。デフォルト名は `dotfiles_ignore` |
| 各階層の `.gitignore` は、そのファイルが置かれたディレクトリを基準に評価 | ネストしたignoreファイルは探索せず、`/` を含む全パターンをdotfilesリポジトリのルート基準で評価 |
| 大文字小文字の扱いはファイルシステムや `core.ignoreCase` などのGit設定に従う | すべてのプラットフォームで常に大文字小文字を区別しない |
| Gitのwildmatch/fnmatch文字クラスを完全に利用可能 | `[[:digit:]]` などのPOSIX文字クラス、照合・等価クラス、文字クラス内でリテラルの `]` を使う一部の特殊形式は未対応 |
| 連続する `**` はGitが定義する特定の配置だけが特殊な意味を持つ | パス要素全体として使う上記の形式だけがGit互換。それ以外の連続スターは検証されず、`*` のように動作する場合がある |
| ignoreルールは未追跡ファイルの探索に適用され、Gitのインデックス状態と関係する | 追跡・未追跡の概念はなく、リンク対象として扱うリポジトリファイルの選択にだけ使用 |
| Git自体には解除不能な組み込み除外パターンはない | DotfilesLinkerの自動除外は別に適用され、`!` では再包含できない |

### 自動除外

以下のファイルやディレクトリは自動的に除外されます：
- `.git`、`.svn`、`.hg` という名前のバージョン管理メタデータディレクトリ
- ルートディレクトリの非ドットファイル（先頭が `.` でないファイル）

`.github` のような類似名のディレクトリは自動除外されません。リンク対象外にする場合は `dotfiles_ignore` に明示してください。

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
