#!/bin/bash

# テスト用PDFファイル作成スクリプト
# 依存関係: pandoc, texlive-latex-base

set -e

TEST_DIR="/tmp/test-pdfs"
mkdir -p "$TEST_DIR"

echo "テスト用PDFファイルを作成中..."

# 1. 請求書サンプル
cat > "$TEST_DIR/invoice_202401.md" << 'EOF'
# 請求書 No.2024-001

**宛先**: 株式会社テストクライアント  
**発行日**: 2024年1月15日  
**支払期限**: 2024年2月14日

## 請求項目

| 項目 | 数量 | 単価 | 金額 |
|------|------|------|------|
| システム開発 | 100時間 | ¥8,000 | ¥800,000 |
| 保守サポート | 1式 | ¥50,000 | ¥50,000 |

**合計**: ¥850,000（税込）
EOF

# 2. 会議議事録
cat > "$TEST_DIR/meeting_minutes_20240315.md" << 'EOF'
# プロジェクトA 進捗会議

**日時**: 2024年3月15日 14:00-15:00  
**出席者**: 田中、佐藤、山田

## 議題

1. 開発進捗報告
2. 課題の共有
3. 次週のタスク

## 決定事項

- データベース設計の最終確認を来週までに完了
- テスト環境の構築を開始
- 次回会議: 3月22日 14:00-
EOF

# 3. 契約書サンプル
cat > "$TEST_DIR/contract_system_dev_2024.md" << 'EOF'
# システム開発委託契約書

**契約番号**: DEV-2024-001  
**契約日**: 2024年4月1日

## 契約内容

### 委託業務
- Webアプリケーション開発
- データベース設計・構築
- システムテスト

### 契約期間
2024年4月1日 ～ 2024年9月30日

### 契約金額
総額 ¥5,000,000（税込）

**発注者**: 株式会社サンプル  
**受注者**: 株式会社デベロップ
EOF

# 4. 技術仕様書
cat > "$TEST_DIR/technical_specification_v2.md" << 'EOF'
# システム技術仕様書 v2.0

## システム概要
- アーキテクチャ: マイクロサービス
- 言語: Go, TypeScript
- データベース: PostgreSQL
- インフラ: Docker, Kubernetes

## API設計

### 認証
- JWT token based authentication
- OAuth 2.0 support

### エンドポイント
- GET /api/v1/users
- POST /api/v1/auth/login
- GET /api/v1/files/search

## セキュリティ要件
- HTTPS通信必須
- API rate limiting
- SQL injection対策
EOF

# 5. 月次レポート
cat > "$TEST_DIR/monthly_report_202403.md" << 'EOF'
# 月次レポート 2024年3月

## 売上実績
- 目標: ¥10,000,000
- 実績: ¥12,500,000
- 達成率: 125%

## プロジェクト進捗
- プロジェクトA: 85%完了
- プロジェクトB: 60%完了
- プロジェクトC: 開始準備中

## 課題と対策
1. リソース不足 → 追加採用検討
2. 技術的負債 → リファクタリング計画策定
EOF

# pandocが利用可能かチェック
if command -v pandoc >/dev/null 2>&1; then
    echo "pandocでPDFを生成中..."
    
    # Markdownファイルをすべて PDFに変換
    for md_file in "$TEST_DIR"/*.md; do
        if [ -f "$md_file" ]; then
            base_name=$(basename "$md_file" .md)
            pandoc "$md_file" -o "$TEST_DIR/${base_name}.pdf"
            echo "✓ ${base_name}.pdf を作成"
        fi
    done
    
    # Markdownファイルを削除
    rm "$TEST_DIR"/*.md
    
else
    echo "警告: pandocがインストールされていません。"
    echo "Ubuntu/Debian: sudo apt-get install pandoc texlive-latex-base"
    echo "macOS: brew install pandoc basictex"
    echo ""
    echo "代替として、ダミーPDFファイルを作成します..."
    
    # ダミーPDFファイル作成（バイナリではなくテキスト）
    for name in "invoice_202401" "meeting_minutes_20240315" "contract_system_dev_2024" "technical_specification_v2" "monthly_report_202403"; do
        echo "%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]>>endobj
xref
0 4
0000000000 65535 f 
0000000010 00000 n 
0000000060 00000 n 
0000000120 00000 n 
trailer<</Size 4/Root 1 0 R>>
startxref
190
%%EOF" > "$TEST_DIR/${name}.pdf"
        echo "✓ ${name}.pdf を作成（ダミー）"
    done
fi

echo ""
echo "テストPDFファイルの作成完了！"
echo "場所: $TEST_DIR"
echo ""
echo "ファイル一覧:"
ls -la "$TEST_DIR"/*.pdf

echo ""
echo "次のステップ:"
echo "1. データベースにサンプルconnectionsを挿入:"
echo "   psql -h localhost -U sokoni -d sokoni -f db/sample_data.sql"
echo ""
echo "2. ローカルテスト用connectionでスキャン実行:"
echo "   ./sokoni scan 6"