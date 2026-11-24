# ランキングのHTTPS接続

ランキングAPIとMMO APIは、別々のPostgreSQLデータベースで動作します。
nginxのHTTPS入口から、`/ranking/`への通信をランキングAPIへ振り分けます。

| 用途 | 接続先 |
|---|---|
| ランキングAPIの基準URL | `https://49.212.160.97/ranking` |
| ランキングの接続確認 | `https://49.212.160.97/ranking/health` |
| MMOの接続確認 | `https://49.212.160.97/health` |
| MMO WebSocket | `wss://49.212.160.97/ws` |

## 構成

`server/docker-compose.yml`はランキングAPIとDBに`restart: unless-stopped`を設定しています。
APIは通常のDB接続用ネットワークに加え、`networkdemo_ranking_gateway`に接続します。
このネットワークで使用するAPIの名前は`ranking-api`です。
DBはこの共有ネットワークには接続しません。

MMO側のComposeには`mmo-gateway.override.yaml`を重ねるか、同等のネットワーク定義を統合します。
nginxの既存HTTPS serverブロック内に`ranking-locations.conf`を組み込みます。
DockerのDNSを使って接続先を再解決するため、ランキングAPI再作成後も新しいIPアドレスを取得できます。
既存のMMOログイン、WebSocket、証明書更新用の経路はそのまま使用します。

## 起動順序

1. ランキング側で`docker compose up -d --build`を実行し、共有ネットワークを作成します。
2. MMO側でネットワーク定義を統合したComposeを使用し、gatewayを起動します。
3. `nginx -t`で設定を検証してから反映します。
4. HTTPSの接続確認とランキングの取得を確認します。

`unless-stopped`は、異常終了時やDocker起動時のコンテナ復帰を設定するものです。
サーバーOSの起動時にはDockerサービスが起動する必要があります。
手動で停止したコンテナの再開や、VPS自体の電源投入を行う機能ではありません。

## クライアント

`ScoreApiClient`と`RankingApiClient`の基準URLはHTTPSです。
Unity Inspectorで基準URLを設定している場合は、シーン側の値も更新します。
既に配布したビルドは更新されないため、従来のHTTP 8080接続も移行期間中は継続します。
暗号化の対象は新しいHTTPS接続であり、従来のHTTP接続は暗号化されません。
