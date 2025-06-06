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
```

## API使用方法

### ファイル名検索

```bash
curl "http://localhost:8080/search?q=sample"
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