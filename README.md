# Offline File Transfer

Windows 11 PC と iPhone / Android の間で、クラウドやインターネットを転送経路として使わずにファイルを送受信するローカル転送ツールです。

PC が転送専用 Wi-Fi Direct ネットワークを作成し、スマートフォン側は Safari / Chrome などのブラウザだけで利用できます。

## 一般利用者の方へ

**GitHub リポジトリを clone して使うことは想定していません。**

GitHub Releases から `OfflineFileTransfer-<version>.zip` をダウンロードし、展開後に最上位の **`Setup.cmd`** を実行してください。

配布 ZIP は次のような構造です。

```text
OfflineFileTransfer/
├─ Setup.cmd              <- 最初に実行するファイル
└─ app/                   <- 内部ファイル。通常は開く必要なし
   ├─ Launch.cmd
   ├─ Launch-Test.cmd
   ├─ Bootstrap.ps1
   ├─ Start-Transfer.ps1
   ├─ LocalPhoneTransfer.exe
   ├─ LocalPhoneTransfer_ARM64.exe
   ├─ SHA256SUMS.txt
   ├─ VERSION.txt
   ├─ transfer-config.json
   └─ locales/
```

`Setup.cmd` は管理者権限を要求し、アプリを次へインストールします。

```text
C:\ProgramData\OfflineFileTransfer\
```

その後、デスクトップに次のショートカットを作成します。

```text
オフラインファイル転送（PC↔スマホ）
```

普段はこのショートカットだけを使用してください。

内部 EXE は一般利用者が直接起動することを想定しておらず、Release パッケージ上で直接実行した場合は Setup を案内して終了します。

## 主な仕様

- iPhone / Android ⇄ Windows 11 の双方向ファイル転送
- スマホ側アプリ不要
- PC 自身が Wi-Fi Direct の転送専用ネットワークを作成
- Wi-Fi パスワードは起動ごとに暗号学的乱数で生成
- 転送 URL / 管理 URL もセッションごとにランダム生成
- 管理画面は localhost のみ
- HTTP サーバーは localhost と転送専用 Wi-Fi 側 IPv4 だけで待ち受け
- Windows Firewall は標準 `192.168.137.0/24` のみに限定
- 最後の実操作から 180 分で自動終了
- 起動から 480 分（8時間）で必ず終了
- 状態ポーリングは実操作として扱わない
- 自動終了の10分前だけ注意表示
- 1ファイルあたり最大 2048 MB（設定可能）
- セッション全体の転送量上限なし
- 同時アップロード数制限
- 保存前の空き容量チェック
- `.upload-*` 一時ファイルを起動時に清掃
- CSP / Permissions-Policy / Referrer-Policy / X-Frame-Options 等を設定
- 日本語 / 英語のローカライズ基盤
- VM テストモードは `127.0.0.1` のみに限定

## 終了方法

PC 側の画面下に表示される **「ファイル転送を終了」** を使用します。

ウィンドウを閉じるだけではなく、終了ボタンから正常終了することを前提にしています。ブラウザを閉じようとした場合は `beforeunload` による確認も行います。

正常終了しなかった場合でも、アイドルタイムアウトと絶対上限が最終的な安全網になります。

## 多言語対応

標準は日本語です。

`config/default.json` / 配布物の `transfer-config.json` にある

```json
"locale": "ja"
```

を

```json
"locale": "en"
```

へ変更すると、PC ブラウザ画面、スマホ画面、起動時メッセージが英語になります。

翻訳ファイルは `locales/` にあります。

```text
locales/
├─ ja.json
├─ en.json
├─ ja.console.json
└─ en.console.json
```

## セキュリティ境界

通常モードは HTTP ですが、通信経路を起動ごとのランダムパスワード付き転送専用 Wi-Fi に限定します。

さらに以下を行います。

- HTTP サーバーを通常 LAN の NIC では待ち受けない
- Firewall も転送専用サブネットだけ許可
- スマホ用 URL は転送専用サブネット以外から拒否
- 正しいセッショントークンで最初に接続したスマホ IP に固定
- 管理 API / QR API はループバックのみ
- URL トークンは暗号学的乱数
- ファイル名サニタイズ / パストラバーサル対策
- 一時ファイル保存後、完了時のみ正式ファイル名へ rename
- 同時アップロード数制限
- ディスク空き容量チェック
- SHA-256 による配布 EXE の整合性確認

SHA-256 は破損や不整合の検出用です。GitHub アカウントや Release 自体の侵害まで独立して保証するものではありません。将来的な公開版では Authenticode コード署名を追加する予定です。

## VM テストモード

インストール後の内部ディレクトリに `Launch-Test.cmd` があります。

VM テストモードでは：

- Wi-Fi Direct を開始しない
- Firewall を変更しない
- HTTP サーバーは `127.0.0.1` のみ
- ホスト PC / 実スマートフォン / 同一 LAN から接続不可
- VM 内ブラウザだけで PC 側とスマホ側 UI を検証可能

## 開発者向け構成

GitHub リポジトリは配布物そのものではなく、開発ソースとして整理しています。

```text
.
├─ src/                         # Go アプリケーション
├─ scripts/windows/             # Windows ランタイム / ランチャー原本
├─ packaging/windows/           # Release 用 Setup
├─ config/                      # 既定設定
├─ locales/                     # UI / console 翻訳
├─ .github/workflows/           # CI / Release パッケージ生成
└─ README.md
```

### CI

`Windows CI` は Go テストと Windows x64 / ARM64 ビルドを行い、バイナリは7日間の Actions artifact として保存します。バイナリを `main` に自動コミットしません。

### Release

`v*` タグを push すると `Release Windows package` workflow が以下を実行します。

1. `go test ./...`
2. Windows x64 / ARM64 EXE を再ビルド
3. ランタイムスクリプト・設定・翻訳を収集
4. `SHA256SUMS.txt` を生成
5. `Setup.cmd + app/` の配布ツリーを構築
6. `OfflineFileTransfer-<version>.zip` と ZIP の SHA-256 を生成
7. GitHub Release に公開

手作業でバイナリを組み合わせて Release を作る運用は想定していません。

## 対応 OS

- Windows 11 x64
- Windows 11 ARM64
