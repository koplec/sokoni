package collector

import (
	"os"
	"path/filepath"
	"strings"

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
