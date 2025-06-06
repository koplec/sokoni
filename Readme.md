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

## DBリセット（必要時）

```bash
docker compose down
docker volume ls 
docker volume rm sokoni_sokoni_pgadata
docker compose up -d
migrate -path db/migrations -database "$DATABASE_URL" up
```