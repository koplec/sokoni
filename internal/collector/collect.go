package collector

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/hirochachacha/go-smb2"
	"github.com/koplec/sokoni/internal/db"
	"github.com/koplec/sokoni/internal/model"
)

func Scan(root string) ([]model.FileInfo, error) {
	var files []model.FileInfo

	err := filepath.WalkDir(root, func(path string, f os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// ここでnilを返しても再帰は続く
		if f.IsDir() {
			return nil
		}

		// .pdfのみ対象
		if strings.HasSuffix(strings.ToLower(f.Name()), ".pdf") {
			info, err := f.Info()
			if err != nil {
				return err
			}
			files = append(files, model.FileInfo{
				Path:    path,
				Name:    f.Name(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func scanWith(root string, handle func(model.FileInfo) error) error {
	return filepath.WalkDir(root, func(path string, f os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// ここでnilを返しても再帰は続く
		if f.IsDir() {
			return nil
		}

		// .pdfのみ対象
		if strings.HasSuffix(strings.ToLower(f.Name()), ".pdf") {
			info, err := f.Info()
			if err != nil {
				return err
			}

			file := model.FileInfo{
				Path:    path,
				Name:    f.Name(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			}

			if err := handle(file); err != nil {
				return err
			}
		}

		return nil
	})
}

func ScanConnectionWith(connection *db.Connection, handle func(model.FileInfo) error) error {
	// SMBパスかどうかはBasePathで判定
	if strings.HasPrefix(connection.BasePath, "//") {
		return scanSMBWith(connection, handle)
	}

	// ローカルパスの場合は base_path と remote_path を結合
	root := connection.BasePath
	if connection.RemotePath != "" {
		root = filepath.Join(connection.BasePath, connection.RemotePath)
	}
	return scanWith(root, handle)
}

func scanSMBWith(connection *db.Connection, handle func(model.FileInfo) error) error {
	// SMB接続情報を解析: BasePath="//server/share", RemotePath="dir"
	parts := strings.Split(strings.TrimPrefix(connection.BasePath, "//"), "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid SMB base path: %s", connection.BasePath)
	}

	server := parts[0]
	share := parts[1]
	remotePath := strings.TrimPrefix(connection.RemotePath, "/")
	if remotePath == "" {
		remotePath = "."
	}

	// SMB接続を確立
	conn, err := net.Dial("tcp", server+":445")
	if err != nil {
		return fmt.Errorf("failed to connect to SMB server: %w", err)
	}
	defer conn.Close()

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     getStringValue(connection.Username),
			Password: getStringValue(connection.Password),
		},
	}

	s, err := d.Dial(conn)
	if err != nil {
		return fmt.Errorf("failed to authenticate SMB: %w", err)
	}
	defer s.Logoff()

	// 共有にマウント
	fs, err := s.Mount(share)
	if err != nil {
		return fmt.Errorf("failed to mount share: %w", err)
	}
	defer fs.Umount()

	// ディレクトリを再帰的にスキャン
	return walkSMBDir(fs, remotePath, "", handle)
}

func walkSMBDir(fs *smb2.Share, dirPath, basePath string, handle func(model.FileInfo) error) error {
	entries, err := fs.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(basePath, entry.Name())
		if entry.IsDir() {
			// ディレクトリの場合は再帰
			subDirPath := filepath.Join(dirPath, entry.Name())
			if err := walkSMBDir(fs, subDirPath, fullPath, handle); err != nil {
				return err
			}
		} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".pdf") {
			// PDFファイルの場合は処理
			info := entry.Sys().(*smb2.FileStat)
			file := model.FileInfo{
				Path:    fullPath,
				Name:    entry.Name(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			}

			if err := handle(file); err != nil {
				return err
			}
		}
	}

	return nil
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
