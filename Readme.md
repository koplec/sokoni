# Sokoni

PDF ファイルスキャン・検索システム

## セットアップ

### 1. データベース起動

```bash
docker compose up -d
```

### 2. データベース初期化

```bash
export DATABASE_URL="postgres://sokoni:sokoni@localhost:5432/sokoni?sslmode=disable"
migrate -path db/migrations -database "$DATABASE_URL" up
```

### 3. 環境設定

```bash
cp test.env.sample test.env
```

## ビルド・実行

### ビルド

```bash
go build ./cmd/sokoni
```

### 実行

```bash
# REST API サーバー起動 (ポート8080)
./sokoni

# ファイルスキャン実行
./sokoni scan

# 特定のconnectionをスキャン
./sokoni scan <connection_id>
```

## テストデータのセットアップ

### 1. サンプルConnectionの挿入

```bash
# テンプレートから独自のサンプルデータを作成
cp db/sample_data.sql.sample db/sample_data.sql

# 環境に合わせてconnection情報を編集（SMBサーバーアドレス、認証情報など）
# vim db/sample_data.sql

# データベースに挿入
psql -h localhost -U sokoni -d sokoni -f db/sample_data.sql
```

### 2. テスト用PDFファイルの作成

```bash
# pandocがインストールされている場合（推奨）
sudo apt-get install pandoc texlive-latex-base  # Ubuntu/Debian
# brew install pandoc basictex  # macOS

# テストPDFファイル作成
./scripts/create_test_pdfs.sh
```

### 3. ローカルテスト用スキャン実行

```bash
# ローカルテスト用connection（ID: 6）でスキャン
./sokoni scan 6
```

## テスト実行

### 全テスト実行

```bash
go test ./...
```

### 特定テストの実行

```bash
# 特定のテスト関数を実行
go test -run TestScanConnectionSMB ./internal/service

# 詳細出力
go test -v ./internal/service
```

### NAS接続テスト

`ScanConnection` の挙動を確認する統合テストを実行するには、まず `test.env.sample`
を `test.env` としてコピーし、必要に応じて NAS(SMB) 接続用の環境変数を設定します。
`ScanConnection` は **connection ID** と **user ID** を受け取るため、テストでは
事前にユーザーを作成してその ID を渡します。

```bash
cp test.env.sample test.env
# NAS 環境に合わせて設定 (任意)
export SOKONI_TEST_SMB_BASE_PATH=//nas/share   # SMBサーバーURI
export SOKONI_TEST_SMB_REMOTE_PATH=docs        # 接続後に参照するフォルダ
export SOKONI_TEST_SMB_USER=myuser       # 任意
export SOKONI_TEST_SMB_PASS=mypass       # 任意
export SOKONI_TEST_SMB_OPTIONS=vers=3.0  # 任意
SOKONI_TEST_SMB_EXPECTED_PDF_COUNT=1

# SMBテストのみ実行
export $(cat test.env | xargs) && go test -run TestScanConnectionSMB -v ./internal/service
```

`SOKONI_TEST_SMB_BASE_PATH` が未設定の場合、SMB を利用したテストはスキップされます。

**注意**: `export $(cat test.env | xargs)` により `test.env` ファイル内の環境変数を現在のシェルセッションに読み込むことができます。

## API使用方法

### Connection一覧取得

```bash
curl "http://localhost:8080/connections"
```

### ファイル名検索

```bash
curl "http://localhost:8080/search?q=invoice"
curl "http://localhost:8080/search?q=contract"
```

### ヘルスチェック

```bash
curl "http://localhost:8080/health"
```

## ログ戦略の考え方

Sokoniは**シンプルなログ出力戦略**を採用しています：

### 基本方針
- **アプリケーション内部でのログ設定は最小限**に留める
- **外部ツールによるログ管理**を前提とした設計
- Unix哲学の「一つのことをうまくやる」に従い、ログの複雑な制御は外部に委ねる

### 出力先
- **通常の情報・進捗**: 標準出力（stdout）
- **エラー情報**: 標準エラー出力（stderr）

### 外部でのログ管理例

#### ファイルへの出力
```bash
# 通常ログとエラーログを分離
./sokoni api 1>app.log 2>error.log

# すべてのログを一つのファイルに
./sokoni api &>all.log
```

#### システムサービス化
```bash
# systemd でサービス化
sudo systemctl start sokoni
journalctl -u sokoni -f  # ログをリアルタイム表示
```

#### Docker運用
```bash
# Dockerコンテナのログ確認
docker logs sokoni-container

# ログドライバーを使用してログ管理システムに転送
docker run --log-driver=fluentd sokoni
```

#### ログローテーション
```bash
# logrotate を使用した自動ローテーション
/etc/logrotate.d/sokoni
```

この方針により、運用環境に応じて柔軟なログ管理が可能になります。

## DBリセット（必要時）

```bash
docker compose down
docker volume ls 
docker volume rm sokoni_sokoni_pgadata
docker compose up -d
migrate -path db/migrations -database "$DATABASE_URL" up
```
