# Local Phone Transfer

Windows 11 共用PCと iPhone / Android の間で、**インターネットを使わずに**ファイルを送受信するためのローカル転送ツールです。

PC自身が転送専用Wi-Fiを作成するため、外付けWi-Fiルーター、スマホのテザリング、クラウドサービスは不要です。スマホ側もSafari / ChromeなどのWebブラウザだけで利用できます。

## できること

- iPhone → PC
- Android → PC
- PC → iPhone
- PC → Android
- PDF / Word / Excel / PowerPoint / ZIP / 画像 / 動画など任意ファイルを転送
- インターネット接続不要
- 外部クラウド不要
- Apple Account / Googleアカウント / Microsoftアカウント不要
- スマホ側アプリ不要
- QRコードから転送専用Wi-Fiへ接続
- QRコードから転送ページを開く
- 起動ごとにランダムな転送URLを生成
- 管理画面はPC自身からのみ利用可能
- 一定時間後に自動終了

## 仕組み

```text
Windows 11 共用PC
  │
  ├─ 転送専用Wi-Fiを作成
  │      SSID: PC-FileTransfer
  │
  └──── ))) ──── iPhone / Android
                    │
                    └─ Safari / Chromeでファイル送受信
```

転送専用Wi-FiはPCとスマホを直接つなぐためだけのローカルネットワークです。インターネット接続は必要ありません。

## 必要条件

- Windows 11
- Wi-Fi機能を搭載したPC、またはUSB無線LANアダプター
- Windows側のWi-Fi Direct機能が利用できること

Windowsの「モバイル ホットスポット」がONになっている場合は、転送開始前にOFFにしてください。Wi-Fi Directと無線リソースが競合する場合があります。

確認例：

```powershell
Get-NetAdapter -IncludeHidden
```

`Microsoft Wi-Fi Direct Virtual Adapter` が表示される環境では、この方式を利用できる可能性があります。最終的な可否は無線LANアダプターとドライバーに依存します。

---

# 初期セットアップ

## 1. リポジトリを配置

例：

```powershell
git clone https://github.com/tamasyun/local-phone-transfer.git
cd local-phone-transfer
```

公開リポジトリなので、clone / pullだけならGitHubへのログインは不要です。

## 2. `初期セットアップ.cmd` を実行

最初に1回だけ、`初期セットアップ.cmd` をダブルクリックしてください。

管理者権限の確認が表示されます。

セットアップでは次を行います。

- TCP 8765 のWindowsファイアウォール受信規則を追加
- 対象をこの転送アプリのEXEに限定
- 接続元を `LocalSubnet` に限定
- デスクトップに「スマホファイル転送」ショートカットを作成

ファイアウォール全体を無効化することはありません。

## 3. Wi-Fi名・パスワード

`transfer-config.json` で設定できます。

```json
"wifiSsid": "PC-FileTransfer",
"wifiPassword": "CHANGE-ME"
```

`CHANGE-ME` のまま利用せず、8〜63文字のパスワードに変更してください。

設定変更は `設定を編集.cmd` から行えます。

---

# 普段の使い方

利用者はデスクトップの

```text
スマホファイル転送
```

をダブルクリックするだけです。

内部では次の処理を自動で行います。

1. Windowsが転送専用Wi-Fiを開始
2. ファイル転送サーバーを開始
3. ブラウザで操作画面を表示
4. 終了時に転送専用Wi-Fiも停止

## STEP 1：スマホを接続

PC画面の最初のQRコードをiPhone / Androidのカメラで読み取ります。

転送専用Wi-Fiへ接続してください。

スマホに

> インターネットに接続されていません

などと表示されても正常です。そのWi-Fiへの接続を維持してください。

## STEP 2：転送ページを開く

PC画面の2つ目のQRコードを読み取ります。

Safari / Chromeで転送ページが開きます。

## スマホ → PC

スマホの転送画面で「ファイルを選んで送信」を押します。

受信したファイルは、アプリと同じ場所の

```text
受信ファイル
```

フォルダーへ保存されます。

## PC → スマホ

PC画面で共有するファイルを選択します。

スマホ側に表示されたファイルの「受け取る」を押すとダウンロードできます。

## 終了

PC画面の「転送を終了する」を押してください。

終了すると、転送URLと転送専用Wi-Fiが無効になります。

標準設定では60分後にも自動終了します。

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
| `wifiPassword` | 転送専用Wi-Fiのパスワード |
| `clearSendOnExit` | 終了時にPC→スマホ共有ファイルを削除するか |
| `autoStopMinutes` | 自動終了までの分数。0で無効 |

Wi-Fi Directでは通常 `192.168.137.x` が使われるため、標準設定ではこの範囲を優先します。

---

# トラブルシューティング

## 転送専用Wi-Fiを開始できない

次を確認してください。

1. WindowsのWi-FiがONになっている
2. 「モバイル ホットスポット」がOFFになっている
3. 無線LANアダプターが有効になっている
4. デバイスマネージャーで無線LANドライバーにエラーがない
5. `Get-NetAdapter -IncludeHidden` でWi-Fi Direct仮想アダプターを確認する

## スマホは接続できたが転送ページを開けない

`初期セットアップ.cmd` をもう一度実行し、ファイアウォール規則を作り直してください。

## 「インターネットなし」と表示される

正常です。このツールはインターネットを経由せず、PCとスマホの間だけで通信します。

---

# セキュリティ

- 転送ページURLは起動ごとにランダム生成
- 管理ページはPC自身からのみアクセス可能
- ファイル名のパストラバーサル対策
- アップロードサイズ制限
- アップロード途中のファイルは一時ファイルとして保存
- Windowsファイアウォールは `LocalSubnet` に限定
- インターネット上のサーバーへファイルを送信しない

転送ページ自体はローカルHTTPです。転送専用Wi-Fiのパスワードは推測されにくい値に設定してください。

# 対応OS

- Windows 11 x64: `スマホファイル転送.exe`
- Windows 11 ARM64: `スマホファイル転送_ARM64.exe`

通常利用者はこれらのEXEを直接起動せず、デスクトップの「スマホファイル転送」ショートカットを使用します。
