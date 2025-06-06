BEGIN;

-- インデックス削除
DROP INDEX IF EXISTS idx_connections_auto_scan;
DROP INDEX IF EXISTS idx_connections_user_id;

-- connectionsテーブルのカラム削除
ALTER TABLE connections 
DROP COLUMN IF EXISTS auto_scan,
DROP COLUMN IF EXISTS scan_interval,
DROP COLUMN IF EXISTS last_scan,
DROP COLUMN IF EXISTS user_id;

-- usersテーブル削除
DROP TABLE IF EXISTS users;

COMMIT;