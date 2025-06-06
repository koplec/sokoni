BEGIN;

-- ユーザーテーブル追加
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- usersテーブルコメント
COMMENT ON TABLE users IS 'Sokoniシステムのユーザー管理テーブル';
COMMENT ON COLUMN users.id IS 'ユーザーID（主キー）';
COMMENT ON COLUMN users.username IS 'ユーザー名（一意）';
COMMENT ON COLUMN users.email IS 'メールアドレス（一意）';
COMMENT ON COLUMN users.password_hash IS 'パスワードハッシュ値';
COMMENT ON COLUMN users.created_at IS '作成日時';
COMMENT ON COLUMN users.updated_at IS '更新日時';

-- connectionsテーブルにユーザー管理とスキャン設定追加
ALTER TABLE connections 
ADD COLUMN user_id INT REFERENCES users(id) ON DELETE CASCADE,
ADD COLUMN last_scan TIMESTAMP WITH TIME ZONE,
ADD COLUMN scan_interval INT DEFAULT 604800, -- 7 days = 7 * 24 * 3600 seconds
ADD COLUMN auto_scan BOOLEAN DEFAULT true;

-- connectionsテーブルコメント追加
COMMENT ON TABLE connections IS 'ネットワーク共有（NAS等）への接続設定テーブル';
COMMENT ON COLUMN connections.id IS '接続ID（主キー）';
COMMENT ON COLUMN connections.name IS '管理用のわかりやすい接続名';
COMMENT ON COLUMN connections.base_path IS 'マウントされるLinuxパス（例: /mnt/share）';
COMMENT ON COLUMN connections.remote_path IS 'リモートパス（例: //192.168.3.63/share）';
COMMENT ON COLUMN connections.username IS 'SMB/CIFS接続用ユーザー名';
COMMENT ON COLUMN connections.password IS 'SMB/CIFS接続用パスワード';
COMMENT ON COLUMN connections.options IS 'マウントオプション文字列';
COMMENT ON COLUMN connections.user_id IS '所有者ユーザーID';
COMMENT ON COLUMN connections.last_scan IS '最後にスキャンを実行した日時';
COMMENT ON COLUMN connections.scan_interval IS 'スキャン間隔（秒）';
COMMENT ON COLUMN connections.auto_scan IS '自動スキャンの有効/無効';
COMMENT ON COLUMN connections.created_at IS '作成日時';
COMMENT ON COLUMN connections.updated_at IS '更新日時';

-- filesテーブルコメント追加
COMMENT ON TABLE files IS 'スキャンされたPDFファイルのメタデータテーブル';
COMMENT ON COLUMN files.id IS 'ファイルID（主キー）';
COMMENT ON COLUMN files.connection_id IS '接続ID（外部キー）';
COMMENT ON COLUMN files.path IS 'ファイルの絶対パス（一意）';
COMMENT ON COLUMN files.name IS 'ファイル名';
COMMENT ON COLUMN files.size IS 'ファイルサイズ（バイト）';
COMMENT ON COLUMN files.mod_time IS 'ファイル最終更新日時';
COMMENT ON COLUMN files.created_at IS 'レコード作成日時';
COMMENT ON COLUMN files.updated_at IS 'レコード更新日時';

-- インデックス追加
CREATE INDEX idx_connections_user_id ON connections(user_id);
CREATE INDEX idx_connections_auto_scan ON connections(auto_scan, last_scan);

COMMIT;