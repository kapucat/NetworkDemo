# k6 Load Tests

HOOK RUNNER SERVERのGo APIを測定するためのk6スクリプトです。

最初は本番ランキングを汚さないように、GETだけを使う読み取り専用テストから開始します。
`POST /api/scores` を含むテストは、テストデータの分離方法を決めてから追加します。

## 1. k6のインストール

Windows PowerShellで実行します。

```powershell
winget install k6 --source winget
```

インストール確認：

```powershell
k6 version
```

## 2. サーバーを起動

ローカルで測定する場合：

```powershell
cd "C:\Users\2240037\OneDrive - yamaguchigakuen\HookRunnerServer\server"
docker compose up -d --build
docker ps
```

ヘルスチェック：

```powershell
Invoke-RestMethod "http://localhost:8080/health"
```

## 3. 読み取り専用スモークテスト

`/health`、`/api/rankings`、`/api/stats` が正常応答するかを1 VU・15秒で確認します。

```powershell
cd "C:\Users\2240037\OneDrive - yamaguchigakuen\HookRunnerServer\server\loadtest"
k6 run -e BASE_URL=http://localhost:8080 smoke-readonly.js
```

VPSを確認する場合：

```powershell
k6 run -e BASE_URL=http://49.212.160.97:8080 smoke-readonly.js
```

## 4. 最初のランキング負荷試験

デフォルトでは、5 VUまで増加し、その後10 VUで測定してから0 VUへ戻します。

```powershell
k6 run -e BASE_URL=http://localhost:8080 rankings-read.js
```

VPSを対象にする場合：

```powershell
k6 run -e BASE_URL=http://49.212.160.97:8080 rankings-read.js
```

段階をコマンド側から変更する例：

```powershell
k6 run -e BASE_URL=http://49.212.160.97:8080 --stage 10s:10 --stage 30s:10 --stage 10s:0 rankings-read.js
```

## 5. 結果を保存

結果フォルダを作成：

```powershell
New-Item -ItemType Directory -Force results
```

サマリーをJSON保存する例：

```powershell
k6 run -e BASE_URL=http://localhost:8080 --summary-export results\smoke-local.json smoke-readonly.js
```

VPS結果の例：

```powershell
k6 run -e BASE_URL=http://49.212.160.97:8080 --summary-export results\rankings-vps-10vu.json rankings-read.js
```

## 6. VPS側のCPU・メモリ確認

負荷試験中に別のPowerShellからVPSへSSH接続します。

```powershell
ssh ubuntu@49.212.160.97
```

VPS上で実行：

```bash
cd ~/HookRunnerServer/server
sudo docker stats hook_runner_api hook_runner_postgres
```

記録する値：

- VU数
- リクエスト数
- RPS
- 平均応答時間
- p95応答時間
- エラー率
- hook_runner_apiのCPU・メモリ
- hook_runner_postgresのCPU・メモリ

## 現在の合格基準

- `http_req_failed` が1%未満
- `http_req_duration` のp95が500ms未満
- checks成功率が99%より高い
- APIコンテナが停止しない
- PostgreSQL接続エラーが発生しない

## 安全上の注意

- 最初はスモークテストを実行する
- いきなり100 VU以上で開始しない
- 10 VU、50 VU、100 VUのように段階的に増やす
- 自分が管理するVPS以外へ負荷試験を行わない
- 高負荷を長時間継続しない
- POSTテストはランキングデータの分離方法を決めてから行う
