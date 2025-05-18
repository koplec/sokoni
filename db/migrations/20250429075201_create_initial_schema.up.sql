BEGIN;

CREATE TABLE connections (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,          -- 管理用のわかりやすい名前
    base_path TEXT NOT NULL,     -- マウントされるLinuxパス (/mnt/share など)
    remote_path TEXT NOT NULL,   -- 例）//192.168.3.63/share
    username TEXT,               -- SMBなら使う（Linuxマウント用）
    password TEXT,               -- SMBなら使う
    options TEXT,                -- fstabに追加したいマウントオプションをここに保存してもいい
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE files (
    id SERIAL PRIMARY KEY,
    connection_id INT NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    size BIGINT,
    mod_time TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

COMMIT;