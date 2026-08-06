# Local Phone Transfer

Windows 11 共用PCと iPhone / Android の間で、**インターネットを使わずに**ファイルを送受信するためのローカル転送ツールです。

PC自身が転送専用Wi-Fiを作成するため、外付けWi-Fiルーター、スマホのテザリング、クラウドサービスは不要です。スマホ側もSafari / ChromeなどのWebブラウザだけで利用できます。

## できること

- iPhone / Android ⇄ Windows 11 の双方向ファイル転送
- PDF / Word / Excel / PowerPoint / ZIP / 画像 / 動画など任意ファイルを転送
- インターネット接続不要
- スマホ側アプリ不要
- QRコードで転送専用Wi-Fiへ接続
- QRコードで転送ページを開く
- **Wi-Fiパスワードを起動ごとに自動生成**
- 転送URLも起動ごとにランダム生成
- 転送終了時にWi-Fiを停止し、今回のパスワードを破棄
- 起動時にGitHubの `main` を自動確認し、更新があれば自動更新
- インターネットがない場合は更新確認だけをスキップして通常起動
- 公共端末を意識した、手順が分かりやすいPC/スマホUI
- 管理画面はPC自身からのみ利用可能
- 標準では60分後に自動終了

## 仕組み

```text
Windows 11 共用PC
  │
  ├─ 起動時に更新確認（インターネットがある場合のみ）
  │
  ├─ 今回限りのWi-Fiパスワードを生成
  │
  ├─ 転送専用Wi-Fiを作成
  │      SSID: PC-FileTransfer
  │
  └──── ))) ──── iPhone / Android
                    │
                    └─ Safari / Chromeでファイル送受信
```

ファイル転送自体はインターネットを使用しません。

## 必要条件

- Windows 11
- Wi-Fi機能を搭載したPC、またはUSB無線LANアダプター
- Windows側のWi-Fi Direct機能が利用できること
- 自動更新を利用する場合のみ Git for Windows

Windowsの「モバイル ホットスポット」がONの場合は、転送開始前にOFFにしてください。

---

# 初期セットアップ

## 1. リポジトリを配置

```powershell
git clone https://github.com/tamasyun/local-phone-transfer.git
cd local-phone-transfer
```

公開リポジトリなので、clone / pullだけならGitHubへのログインは不要です。

## 2. `初期セットアップ.cmd` を実行

最初に1回だけ実行してください。

セットアップでは次を行います。

- TCP 8765 のWindowsファイアウォール受信規則を追加
- 対象を転送アプリのEXEに限定
- 接続元を `LocalSubnet` に限定
- デスクトップに `Local Phone Transfer` ショートカットを作成

ファイアウォール全体を無効化することはありません。

---

# 普段の使い方

デスクトップの

```text
Local Phone Transfer
```

をダブルクリックします。

内部では次の順に処理します。

1. GitHubの更新を確認
2. 更新があれば `main` を最新化
3. 今回限りのWi-Fiパスワードを生成
4. Windowsが転送専用Wi-Fiを開始
5. ファイル転送サーバーを開始
6. 公共端末風の操作画面を表示
7. 終了時に転送専用Wi-Fiを停止

インターネットに接続されていない場合、1〜2だけスキップし、現在のバージョンでそのまま起動します。

## STEP 1：スマホを接続

PC画面の最初のQRコードをiPhone / Androidのカメラで読み取り、転送専用Wi-Fiへ接続します。

スマホに

> インターネットに接続されていません

などと表示されても正常です。

Wi-Fiパスワードは起動ごとに新しく生成され、PC画面にも表示されます。QRコードを使えば通常は手入力不要です。

## STEP 2：転送ページを開く

PC画面の2つ目のQRコードを読み取ります。

Safari / Chromeで転送画面が開きます。

## スマホ → PC

スマホ画面の「ファイルを選ぶ」から送信します。

受信したファイルは

```text
受信ファイル
```

へ保存されます。

## PC → スマホ

PC画面の「PCからスマホへ送るファイルを選ぶ」から共有します。

スマホ側にファイルが表示されたら「受け取る」を押します。

## 終了

PC画面の「転送を終了する」を押します。

終了すると、

- 転送ページURLが無効化
- 転送専用Wi-Fiが停止
- 今回のWi-Fiパスワードが破棄

されます。

---

# 自動更新

`LocalPhoneTransfer.cmd` は `Bootstrap.ps1` を経由して起動します。

起動時に、

- Gitが利用可能
- 現在のブランチが `main`
- GitHubへ短時間で接続可能
- 追跡対象ファイルにローカル変更がない

場合に `origin/main` を確認します。

更新があればFast-forwardで更新し、その最新版で転送を開始します。

ネットワークがない、GitHubへ接続できない、ローカル変更がある、といった場合は**更新だけをスキップ**し、ファイル転送機能は止めません。

---

# 設定項目

`transfer-config.json`

| 項目 | 意味 |
|---|---|
| `deviceName` | スマホ画面に表示するPC名 |
| `port` | 転送サーバーのTCPポート |
| `receiveDir` | スマホ→PCの保存先 |
| `sendDir` | PC→スマホの共有用保存先 |
| `maxUploadMB` | 1ファイルあたりの最大サイズ(MB) |
| `preferredInterfaceContains` | 優先するネットワークアダプター名の一部 |
| `preferredIpPrefix` | 優先するIPv4アドレスの先頭部分 |
| `wifiSsid` | 転送専用Wi-Fi名 |
| `wifiPasswordLength` | 起動時に生成するWi-Fiパスワード長 |
| `clearSendOnExit` | 終了時にPC→スマホ共有ファイルを削除するか |
| `autoStopMinutes` | 自動終了までの分数。0で無効 |

Wi-Fiパスワードそのものは設定ファイルへ保存しません。

---

# ターミナル表示

PowerShell 5.1の文字コード問題を避けるため、実行用 `.ps1` はASCII中心で記述し、日本語表示文は `messages.json` からUTF-8として読み込みます。

そのため通常の起動時メッセージは日本語ですが、日本語ソース文字列によってPowerShell構文が壊れにくい構成です。

---

# ビルド

Goソースは `src/` にあります。

`src/` が更新されるとGitHub Actionsが自動で、

- Windows x64
- Windows ARM64

のEXEをビルドし、リポジトリ直下の実行ファイルと `SHA256SUMS.txt` を更新します。

---

# セキュリティ

- Wi-Fiパスワードは起動ごとにランダム生成
- 読み間違えやすい文字を除外した16文字を標準使用
- 転送ページURLも起動ごとにランダム生成
- 管理ページはPC自身からのみアクセス可能
- ファイル名のパストラバーサル対策
- アップロードサイズ制限
- アップロード途中のファイルは一時ファイルとして保存
- Windowsファイアウォールは `LocalSubnet` に限定
- インターネット上のサーバーへ転送ファイルを送信しない

# 対応OS

- Windows 11 x64: `スマホファイル転送.exe`
- Windows 11 ARM64: `スマホファイル転送_ARM64.exe`

通常利用者はEXEを直接起動せず、デスクトップの `Local Phone Transfer` を使用します。
