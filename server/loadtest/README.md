# k6 Load Tests

HOOK RUNNER SERVERのGo APIについて、負荷試験、ボトルネック調査、性能改善、再測定を行うための資料です。

## 現在の完了状態

ランキング取得APIについて、以下まで完了しています。

- 本番環境とは分離した負荷試験専用Docker環境
- 10,000件のテストデータ生成
- k6による読み取り負荷試験
- 300 VUでの障害再現
- APIログと`EXPLAIN (ANALYZE, BUFFERS)`による原因調査
- ランキング用複合インデックス追加
- GoのDB接続プール設定
- 同じ300 VU条件での再測定
- VPS通常環境への反映
- UnityからVPSへのスコア送信とランキング表示確認

## 負荷試験専用環境

通常ランキングを汚さないように、通常用Composeとは別の環境を使用します。

```txt
k6
↓ HTTP localhost:8081
hook_runner_api_loadtest
↓ Docker内部ネットワーク
hook_runner_postgres_loadtest
↓
hook_runner_loadtest_db
```

構成ファイル：

```txt
server/docker-compose.loadtest.yml
server/loadtest/seed.sql
server/loadtest/smoke-readonly.js
server/loadtest/rankings-read.js
```

設定：

```txt
API：http://localhost:8081
DB：hook_runner_loadtest_db
テストデータ：10,000件
プレイヤー数：1,000人
1プレイヤー当たり：10件
stage_id：1
PostgreSQLのホスト側ポート公開：なし
```

## k6のインストール

Windows PowerShellで実行します。

```powershell
winget install k6 --source winget
k6 version
```

## 専用環境の起動

```powershell
cd "C:\Users\2240037\Desktop\3年\卒業制作\HookRunnerServer\server"
docker compose -f docker-compose.loadtest.yml up -d --build
docker ps
```

ヘルスチェック：

```powershell
Invoke-RestMethod "http://localhost:8081/health"
```

停止：

```powershell
docker compose -f docker-compose.loadtest.yml down
```

テストDBを初期化し直す場合のみ、ボリュームを削除します。

```powershell
docker compose -f docker-compose.loadtest.yml down -v
docker compose -f docker-compose.loadtest.yml up -d --build
```

> 通常用の`docker compose`に対して、誤って`down -v`を実行しないこと。

## スモークテスト

`/health`、`/api/rankings`、`/api/stats`が正常応答するかを、1 VU・15秒で確認します。

```powershell
cd "C:\Users\2240037\Desktop\3年\卒業制作\HookRunnerServer\server\loadtest"
k6 run -e BASE_URL=http://localhost:8081 smoke-readonly.js
```

## ランキング読み取り試験

```powershell
k6 run -e BASE_URL=http://localhost:8081 rankings-read.js
```

VUをコマンドから変更する例：

```powershell
k6 run -e BASE_URL=http://localhost:8081 `
  --stage 10s:100 `
  --stage 30s:100 `
  --stage 10s:0 `
  rankings-read.js
```

300 VUで実行する場合は、まず10 VU、50 VU、100 VUなどで段階的に確認してから増やします。

### RPSについての注意

`rankings-read.js`には各反復の最後に`sleep(1)`があります。
そのため、測定されたRPSは一定間隔でアクセスする利用者を想定した値であり、サーバーの理論上の最大RPSではありません。

## 合格基準

```txt
http_req_failed：1%未満
http_req_duration p95：500ms未満
checks成功率：99%超
APIコンテナが停止しない
PostgreSQL接続エラーが発生しない
```

# 性能改善結果

## 改善前に発生した問題

10,000件のデータを使用して300 VUのランキング取得試験を行ったところ、エラーと大きなDB負荷が発生しました。

| 項目 | 改善前 |
|---|---:|
| 成功率 | 95.24% |
| エラー率 | 4.75% |
| 平均応答時間 | 201.35ms |
| p95 | 741.83ms |
| 最大応答時間 | 1.47秒 |
| 平均RPS | 159.57 |
| 失敗リクエスト | 689件 |
| API最大CPU | 47.94% |
| PostgreSQL最大CPU | 1586.97% |
| API最大メモリ | 24.35MiB |
| PostgreSQL最大メモリ | 319.30MiB |

APIログ：

```txt
ranking query failed: pq: sorry, too many clients already (53300)
```

DockerのCPU使用率は、複数のCPUコアを使用すると100%を超えます。
`1586.97%`は、およそ15.9論理コア分のCPU時間を使用した表示です。

## 実行計画による原因調査

改善前：

```txt
Seq Scan on scores
走査行数：10,000
Sort Method：quicksort
Execution Time：19.916ms
```

ランキング取得のたびに10,000件を全件走査し、プレイヤー名、クリアタイム、作成日時で並べ替えていました。

## ランキング用複合インデックス

ランキングSQLの検索条件と並び順に合わせて、以下を追加しました。

```sql
CREATE INDEX IF NOT EXISTS idx_scores_ranking
ON scores (
    stage_id,
    player_name,
    clear_time,
    created_at
)
INCLUDE (death_count);
```

目的：

- `stage_id`で対象ステージを絞り込む
- `player_name`ごとのベストスコア取得に利用する
- `clear_time`と`created_at`の並び順を利用する
- `death_count`を`INCLUDE`し、テーブル本体の読み取りを減らす

改善後：

```txt
Index Only Scan using idx_scores_ranking
Heap Fetches：0
Execution Time：1.925ms
```

SQL単体では、19.916msから1.925msへ約90.3%短縮し、約10.3倍高速化しました。

## DB接続プール

Goの`database/sql`に以下を設定しました。

```go
db.SetMaxOpenConns(20)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(30 * time.Minute)
db.SetConnMaxIdleTime(5 * time.Minute)
```

目的：

- PostgreSQLへの同時接続を最大20に制限する
- 待機接続を再利用する
- 古い接続や長時間未使用の接続を更新する
- アクセス増加時の接続数を制御する

## 改善段階ごとの300 VU結果

| 項目 | 改善前 | インデックス追加後 | 接続プール追加後 |
|---|---:|---:|---:|
| 成功率 | 95.24% | 100.00% | 100.00% |
| エラー率 | 4.75% | 0.00% | 0.00% |
| 平均応答時間 | 201.35ms | 7.59ms | 7.62ms |
| p95 | 741.83ms | 28.69ms | 25.20ms |
| 最大応答時間 | 1.47秒 | 62.81ms | 64.34ms |
| 平均RPS | 159.57 | 189.74 | 190.41 |
| 完了リクエスト | - | 17,249件 | 17,239件 |
| API最大CPU | 47.94% | 74.31% | 41.57% |
| PostgreSQL最大CPU | 1586.97% | 216.71% | 169.38% |
| API最大メモリ | 24.35MiB | 21.10MiB | 19.88MiB |
| PostgreSQL最大メモリ | 319.30MiB | 47.18MiB | 52.06MiB |

### 結果の解釈

最も大きな性能改善は、ランキングSQLに合わせた複合インデックスによるものです。
インデックス追加後の時点でエラー率は0%になり、平均応答時間も約7.6msまで短縮しました。

DB接続プールは、改善全体を単独で発生させたものではありません。
接続数の上限を明示し、高負荷時にPostgreSQLへ接続が無制限に増えないようにする安全性と安定性の改善です。

## 改善前後の最終比較

| 項目 | 改善前 | 最終 | 変化 |
|---|---:|---:|---:|
| 成功率 | 95.24% | 100.00% | +4.76ポイント |
| エラー率 | 4.75% | 0.00% | エラー解消 |
| 平均応答時間 | 201.35ms | 7.62ms | 約96.2%短縮 |
| p95 | 741.83ms | 25.20ms | 約96.6%短縮 |
| PostgreSQL最大CPU | 1586.97% | 169.38% | 約89.3%減少 |
| PostgreSQL最大メモリ | 319.30MiB | 52.06MiB | 約83.7%減少 |

## VPSで行った読み取り試験の記録

以下は、負荷試験専用ローカル環境を作る前にVPS通常環境で行った読み取り専用テストの記録です。
専用環境での改善比較とは測定環境が異なるため、同じ表の数値を直接比較しません。

| VU | 平均応答時間 | p95 | 最大 | RPS | エラー率 | API CPU | PostgreSQL CPU |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 100 | 36.51ms | 75.91ms | 887.63ms | 66.38 | 0% | 11.03% | 19.44% |
| 200 | 29.36ms | 74.20ms | 714.92ms | 121.77 | 0% | 44.59% | 115.70% |
| 300 | 33.85ms | 94.17ms | 567.65ms | 185.28 | 0% | 59.82% | 123.04% |
| 500 | 66.00ms | 209.24ms | 6.99秒 | 274.02 | 0% | 124.75% | 269.03% |

# 結果の保存

結果フォルダ：

```powershell
New-Item -ItemType Directory -Force results
```

JSONサマリー保存例：

```powershell
k6 run -e BASE_URL=http://localhost:8081 `
  --summary-export results\rankings-local.json `
  rankings-read.js
```

`server/loadtest/results/`はGit管理対象外です。

負荷試験中のリソース確認：

```powershell
docker stats hook_runner_api_loadtest hook_runner_postgres_loadtest
```

記録する値：

- VU数
- リクエスト数
- RPS
- 平均応答時間
- p95応答時間
- 最大応答時間
- エラー率
- APIコンテナのCPU・メモリ
- PostgreSQLコンテナのCPU・メモリ

# 安全上の注意

- 最初にスモークテストを実行する
- 10 VU、50 VU、100 VUのように段階的に増やす
- 本番ランキングへPOSTを含む負荷試験を行わない
- 自分が管理する環境以外へ負荷試験を行わない
- 高負荷を長時間継続しない
- CPUとメモリを`docker stats`で監視する
- 通常用Composeで`down -v`を実行しない

# 今後の追加検証候補

必要性と制作時間を確認して、次の項目を追加します。

- 10万件以上のテストデータ
- ランキング取得とスコア送信を混ぜた負荷試験
- HTTPサーバーの`ReadTimeout`、`WriteTimeout`、`IdleTimeout`
- リクエスト処理時間の計測Middleware
- 異常な連続投稿への簡易制限

Redis、Kubernetes、gRPCなどの大規模な構成は、必要性が確認できない限り追加しません。
