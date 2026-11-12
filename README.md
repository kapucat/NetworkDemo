# ネットワークDemo

UnityとGoで制作した、オンライン通信の技術デモです。
ランキングへの記録送信・取得と、複数プレイヤーの移動同期・チャットを実装しています。

## 公開内容

このリポジトリは、自作プログラムを閲覧するためのソースコード公開版です。
キャラクター、ステージ、画像、フォント、Unityのシーン・Prefab、外部パッケージ本体、実行ファイルは含みません。
そのため、このリポジトリだけでは元のゲームを再現・ビルドできません。
ゲームの実行には、展示用に別途配布するWindowsビルドを使用してください。

## 主なコード

| 場所 | 内容 |
|---|---|
| `Assets/ランキングシステムのアクションゲーム/Scripts/` | スコア送信、ランキング表示、アクション操作 |
| `Assets/大規模多人数同時参加型オンラインゲーム/MMO-Scripts/` | アカウント登録・ログイン、WebSocket通信、移動同期、チャット |
| `Assets/Common共通/Scripts/` | 共通の画面遷移 |
| `server/` | Go製API・WebSocketサーバー、PostgreSQL接続 |
| `server/loadtest/` | k6による負荷試験コードと性能改善の記録 |

## 使用技術

- Unity 2022.3.48f1 / C#
- Go / HTTP API / WebSocket
- PostgreSQL / Docker Compose
- k6

サーバーのコードは `server/main.go`、`server/accounts.go`、`server/websocket.go` にあります。
負荷試験の条件と結果は [負荷試験の説明](server/loadtest/README.md) を参照してください。
Compose内の開発用設定値を、そのまま公開サーバーの認証情報として使用しないでください。
