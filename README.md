# Local Phone Transfer

Windows 11 共用PCと iPhone / Android の間で、インターネットやクラウドを使わずにファイルを送受信するためのローカル転送ツールです。

PC自身が転送専用Wi-Fiを作成し、スマートフォンはSafari / ChromeなどのWebブラウザだけで利用できます。

## 主な仕様

- iPhone / Android ⇄ Windows 11 の双方向ファイル転送
- スマホ側アプリ不要
- 転送専用Wi-FiをPC自身が作成
- Wi-Fiパスワードは起動ごとに16文字でランダム生成
- 転送URLと管理URLも起動ごとにランダム生成
- 管理画面は `127.0.0.1` からのみ利用可能
- 通常モードのHTTPサーバーは `127.0.0.1` と転送専用Wi-FiのIPv4アドレスだけで待ち受け
- 通常LANやEthernetのアドレスでは待ち受けない
- Windows Firewallも標準では `192.168.137.0/24` のみに限定
- 最後の実操作から180分間操作がないと自動終了
- 定期的な画面更新は実操作として扱わない
- 残り時間は常時表示せず、自動終了の10分前からだけ注意を表示
- 起動からの絶対時間上限は現在設定していない
- 起動時に未完了の `.upload-*` 一時ファイルを清掃
- 同時アップロード数を制限
- 保存前に空き容量を確認
- 1ファイルあたりの最大サイズを制限
- HTTPセキュリティヘッダー（CSP / Permissions-Policy / X-Frame-Options等）を設定
- VMテストモードは `127.0.0.1` のみに限定
- note / Notionのような白基調・余白中心のシンプルUI

## 初期セットアップ

```powershell
git clone https://github.com/tamasyun/local-phone-transfer.git
cd local-phone-transfer
```

最初に1回だけ `初期セットアップ.cmd` を実行してください。

セットアップでは次を行います。

- 転送アプリのEXEだけを対象にWindows Firewall規則を作成
- TCPポートを `transfer-config.json` の `port` に限定
- 接続元を `firewallRemoteSubnet`（標準 `192.168.137.0/24`）に限定
- デスクトップへ `Local Phone Transfer` ショートカットを作成

Firewall規則の仕様を更新した場合は、`git pull` 後に `初期セットアップ.cmd` をもう一度実行してください。

## 普段の使い方

デスクトップの `Local Phone Transfer` を起動します。

1. 起動時にアプリ本体のSHA-256を `SHA256SUMS.txt` と照合
2. GitHubへ接続できる場合だけ `main` の更新を確認
3. Gitの接続先が公式リポジトリと一致することを確認
4. Fast-forwardできる場合だけ更新
5. 更新後に再度EXEのSHA-256を検証
6. 検証に失敗した場合は更新前のコミットへ戻す
7. 今回限りのWi-Fiパスワードを生成
8. 転送専用Wi-Fiを開始
9. PC操作画面を開く

インターネットがない場合は更新確認だけをスキップし、ローカルファイル転送は通常どおり利用できます。

> `SHA256SUMS.txt` は破損や意図しないバイナリ差し替えの検出には有効ですが、GitHubアカウント自体が侵害された場合まで保証する独立した署名ではありません。配布用途でより強い発行者検証が必要な場合はAuthenticodeコード署名証明書を追加してください。

## アイドルタイムアウト

標準では `idleTimeoutMinutes: 180` です。

タイマーを更新する「実操作」の例：

- スマホで転送ページを開く
- スマホ → PC のアップロード
- PC → スマホ のダウンロード
- PC側で共有ファイルを追加
- 共有/受信フォルダーを開く
- 共有ファイルや受信ファイルを削除

タイマーを更新しないもの：

- PC画面の状態確認ポーリング
- スマホ側のファイル一覧自動更新
- 画面を開いたまま放置しているだけの状態

ファイルのアップロード中は、データが継続して流れている限り一定間隔で実操作として更新されます。

## 異常終了・シャットダウン

PCのシャットダウンや強制終了で転送が中断した場合、転送用Wi-FiとHTTPサーバーは停止します。

アップロード中のファイルは `.upload-*` の一時ファイルとして保存され、正常完了した時だけ正式なファイル名へ変更されます。次回起動時に残っている `.upload-*` は自動削除されます。

完了済みの受信ファイルはそのまま残ります。

## セキュリティ上の境界

通常モードではHTTPを使用しますが、通信経路を起動ごとのランダムパスワード付き転送専用Wi-Fiに限定し、HTTPサーバー自体もその転送用IPとlocalhostにだけバインドします。

さらにWindows Firewallでも転送専用サブネットだけを許可します。

管理APIはリクエスト元がループバックアドレスでなければ404を返します。

ログは `logs/app.log` に保存されます。Wi-Fiパスワード、URLトークン、ファイル名はログへ記録しません。ログが大きくなった場合は `app.log.1` へローテーションします。

## VMテストモード

VMwareなどWi-Fi Directを利用できない環境では `LocalPhoneTransfer-Test.cmd` を使用できます。

テストモードでは：

- Wi-Fi Directを起動しない
- Windows Firewallを変更しない
- HTTPサーバーは `127.0.0.1` のみ
- ホストPC、スマートフォン、同一LANの別端末から接続不可
- PC画面とスマホ画面をVM内ブラウザで確認可能

## 設定

`transfer-config.json`

| 項目 | 標準値 | 意味 |
|---|---:|---|
| `port` | `8765` | HTTPサーバーポート |
| `maxUploadMB` | `2048` | 1ファイルあたりの最大サイズ |
| `minFreeSpaceMB` | `1024` | 転送後も残す最低空き容量 |
| `maxConcurrentUploads` | `2` | 同時アップロード数 |
| `idleTimeoutMinutes` | `180` | 最後の実操作から自動終了までの時間 |
| `preferredIpPrefix` | `192.168.137.` | 転送専用Wi-FiのIPv4プレフィックス |
| `firewallRemoteSubnet` | `192.168.137.0/24` | Firewallで許可する転送専用サブネット |
| `wifiSsid` | `PC-FileTransfer` | 転送専用Wi-Fi名 |
| `wifiPasswordLength` | `16` | 起動時に生成するWi-Fiパスワード長 |
| `clearSendOnExit` | `true` | 終了時にPC→スマホ共有ファイルを消すか |

## ビルド

Goソースは `src/` にあります。`src/` の変更時にGitHub ActionsがWindows x64 / ARM64のEXEをビルドし、`SHA256SUMS.txt` を更新します。

## 対応OS

- Windows 11 x64
- Windows 11 ARM64
