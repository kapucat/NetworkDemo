# ランキングAPIの負荷試験と性能改善

ネットワークDemoのGo製ランキング取得APIを対象に、k6による負荷試験、ボトルネック調査、性能改善を実施しました。
本資料は、その試験構成と測定結果をまとめたものです。

## 試験の概要

10,000件のテストデータを使用した300 VUの試験で、DB接続エラーと応答時間の増大を再現しました。
実行計画の分析に基づく複合インデックスの追加と、DB接続プールの設定後、同じ300 VU条件で再測定しています。
ここで示すVUは、ランキング取得リクエストを実行する仮想利用者数です。

| 項目 | 改善前 | 改善後 |
|---|---:|---:|
| 成功率 | 95.24% | 100.00% |
| 平均応答時間 | 201.35ms | 7.62ms |
| p95応答時間 | 741.83ms | 25.20ms |

## 試験環境

負荷試験には、通常のランキングとは独立したDocker Compose構成と専用DBを使用しています。

```text
k6
↓ HTTP localhost:8081
hook_runner_api_loadtest
↓ Docker内部ネットワーク
hook_runner_postgres_loadtest
↓
hook_runner_loadtest_db
```

| 項目 | 設定 |
|---|---|
| API | http://localhost:8081 |
| DB | hook_runner_loadtest_db |
| テストデータ | 10,000件 |
| テスト用プレイヤー | 1,000人 |
| 1プレイヤー当たりの記録 | 10件 |
| stage_id | 1 |
| PostgreSQLのホスト側ポート | 非公開 |

### 関連ファイル

| ファイル | 内容 |
|---|---|
| `server/docker-compose.loadtest.yml` | 負荷試験専用のAPI・DB構成 |
| `server/loadtest/seed.sql` | テストデータの生成 |
| `server/loadtest/smoke-readonly.js` | ヘルスチェック・ランキング・統計APIの疎通試験 |
| `server/loadtest/rankings-read.js` | ランキング取得APIの負荷試験 |

### 評価指標

- リクエスト失敗率：1%未満
- p95応答時間：500ms未満
- checks成功率：99%超
- APIコンテナの稼働継続
- PostgreSQL接続エラーの有無
- API・DBコンテナのCPU使用率とメモリ使用量

### 測定値の読み方

`rankings-read.js`は、各反復の最後に`sleep(1)`を実行します。
測定されたRPSは、この待機時間を含む試験条件での処理件数です。

## 試験の実行例

実行環境はDocker Compose、k6、Windows PowerShellです。
以下のパスはリポジトリのルートを基準にしています。

### 専用環境の起動

```powershell
cd server
docker compose -f docker-compose.loadtest.yml up -d --build
Invoke-RestMethod "http://localhost:8081/health"
```

### 疎通試験

`smoke-readonly.js`は、1 VU・15秒で3つの読み取りAPIを確認します。

```powershell
cd loadtest
k6 run -e BASE_URL=http://localhost:8081 smoke-readonly.js
```

### ランキング取得試験

`rankings-read.js`の標準設定は最大10 VUです。下記の実行例と、本文の300 VU比較結果は負荷条件が異なります。

```powershell
k6 run -e BASE_URL=http://localhost:8081 rankings-read.js
```

実行時に負荷条件を指定する例：

```powershell
k6 run -e BASE_URL=http://localhost:8081 `
  --stage 10s:100 `
  --stage 30s:100 `
  --stage 10s:0 `
  rankings-read.js
```

## 性能改善結果

### 改善前に発生した問題

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

### 実行計画による原因調査

改善前：

```txt
Seq Scan on scores
走査行数：10,000
Sort Method：quicksort
Execution Time：19.916ms
```

ランキング取得のたびに10,000件を全件走査し、プレイヤー名、クリアタイム、作成日時で並べ替えていました。

### ランキング用複合インデックス

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

### DB接続プール

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

### 改善段階ごとの300 VU結果

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

#### 結果の解釈

最も大きな性能改善は、ランキングSQLに合わせた複合インデックスによるものです。
インデックス追加後の時点でエラー率は0%になり、平均応答時間も約7.6msまで短縮しました。

DB接続プールは、改善全体を単独で発生させたものではありません。
接続数の上限を明示し、高負荷時にPostgreSQLへ接続が無制限に増えないようにする安全性と安定性の改善です。

### 改善前後の最終比較

| 項目 | 改善前 | 最終 | 変化 |
|---|---:|---:|---:|
| 成功率 | 95.24% | 100.00% | +4.76ポイント |
| エラー率 | 4.75% | 0.00% | エラー解消 |
| 平均応答時間 | 201.35ms | 7.62ms | 約96.2%短縮 |
| p95 | 741.83ms | 25.20ms | 約96.6%短縮 |
| PostgreSQL最大CPU | 1586.97% | 169.38% | 約89.3%減少 |
| PostgreSQL最大メモリ | 319.30MiB | 52.06MiB | 約83.7%減少 |

### VPSで行った読み取り試験の記録

負荷試験専用のローカル環境を構築する前に、VPSで実施した読み取り試験の記録です。
ローカル環境の改善前後比較とは測定環境が異なります。

| VU | 平均応答時間 | p95 | 最大 | RPS | エラー率 | API CPU | PostgreSQL CPU |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 100 | 36.51ms | 75.91ms | 887.63ms | 66.38 | 0% | 11.03% | 19.44% |
| 200 | 29.36ms | 74.20ms | 714.92ms | 121.77 | 0% | 44.59% | 115.70% |
| 300 | 33.85ms | 94.17ms | 567.65ms | 185.28 | 0% | 59.82% | 123.04% |
| 500 | 66.00ms | 209.24ms | 6.99秒 | 274.02 | 0% | 124.75% | 269.03% |

## 結果の記録

JSON形式のサマリー出力例：

```powershell
New-Item -ItemType Directory -Force results
k6 run -e BASE_URL=http://localhost:8081 `
  --summary-export results\rankings-local.json `
  rankings-read.js
```

コンテナのCPU・メモリ使用量は、以下のコマンドで取得します。

```powershell
docker stats hook_runner_api_loadtest hook_runner_postgres_loadtest
```

生成される測定結果の保存先は`server/loadtest/results/`です。
リポジトリには試験スクリプトと本文の測定記録を公開しています。
