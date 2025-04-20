package model

import "time"

type FileInfo struct {
	Path    string
	Name    string
	Size    int64     //os.FileInfo.SIze()でint64が返る
	ModTime time.Time // 最終更新日時
}
