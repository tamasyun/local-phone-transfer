# ルーターなしモード

Windows 11 PC自身が **Wi-Fi Direct Legacy AP** を作り、iPhone / Android から通常のWi-Fiとして接続するモードです。

追加のWi-Fiルーター、スマホのテザリング、インターネット接続は不要です。

## 今回確認したPC

`Get-NetAdapter -IncludeHidden` の結果に **Microsoft Wi-Fi Direct Virtual Adapter** が存在しているため、Wi-Fi Directを利用できる可能性が高い構成です。

一方、`netsh wlan show drivers` の

```text
ホストされたネットワークのサポート: いいえ
```

は、旧式の Hosted Network (`netsh wlan set hostednetwork`) が使えないことを示す項目です。今回利用する Wi-Fi Direct Legacy AP とは別機能です。

## 使い方

初回だけ `初期セットアップ.cmd` を実行し、Windowsファイアウォール規則を追加してください。

その後は `スマホファイル転送.exe` を直接起動せず、次をダブルクリックします。

```text
スマホファイル転送（ルーター不要）.cmd
```

処理の流れは次の通りです。

1. Windows PowerShell 5.1 から Wi-Fi Direct Legacy AP を開始
2. `transfer-config.json` の `wifiSsid` / `wifiPassword` をSSID・パスワードとして利用
3. Wi-Fi Direct仮想アダプターの準備を待つ
4. 既存の `スマホファイル転送.exe` を起動
5. PC画面に表示されたWi-Fi QRをiPhone / Androidで読み取る
6. 転送ページのQRを読み取り、ブラウザからファイルを送受信
7. 転送アプリ終了後、Wi-Fi Direct APも停止

## インターネットについて

転送専用Wi-Fiにはインターネット接続を提供しません。

```text
Windows 11 PC
     )))  PC-FileTransfer
       ├─ iPhone
       └─ Android
```

PCが別のWi-Fiでインターネット接続されていても、ファイル転送自体はこのローカル接続内で行います。

## 動作しない場合

### Windowsの「モバイル ホットスポット」をOFFにする

WindowsのMobile HotspotとWi-Fi Directは同時利用できないことがあります。

### Wi-FiがONか確認する

USB無線LANアダプターを抜いた状態や、Wi-Fiが無効化されている状態では開始できません。

### Wi-Fi Direct Virtual Adapterを確認する

PowerShellで以下を実行します。

```powershell
Get-NetAdapter -IncludeHidden
```

`Microsoft Wi-Fi Direct Virtual Adapter` が存在するか確認してください。

### Hosted Networkが「いいえ」でも問題ない

```powershell
netsh wlan show drivers
```

の `ホストされたネットワークのサポート: いいえ` は今回の方式の可否を直接示す項目ではありません。

## 現時点の注意

このモードはWindows 11実機での最終検証前です。今回確認したPC上で実際にSSIDが表示され、iPhone / Androidが接続できるかを確認してからmainへマージしてください。
