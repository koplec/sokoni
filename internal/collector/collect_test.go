package collector

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/koplec/sokoni/internal/model"
)

// テスト用に一時ディレクトリを作成して、ダミーのpdfを配置
func setupTestDir(t *testing.T) string {
	// t.TempDirはtestingパッケージが提供する、一時ディレクトリ作成関数
	// テストが終わると、Goが勝手に削除する
	dir := t.TempDir()

	// 0644 自分は編集できる、他の人は読むだけ
	pdf1 := filepath.Join(dir, "sample1.pdf")
	os.WriteFile(pdf1, []byte("dummy"), 0644)

	boo1 := filepath.Join(dir, "note.boo")
	os.WriteFile(boo1, []byte("not a pdf"), 0644)

	return dir
}

func TestScanPDFs(t *testing.T) {
	dir := setupTestDir(t)

	files, err := Scan(dir)
	if err != nil {
		t.Fatalf("unexpectred error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 pdf file, got %d", len(files))
	}

	if files[0].Name != "sample1.pdf" {
		t.Errorf("unexpecrted file name: %s", files[0].Name)
	}

	if files[0].Size != 5 {
		t.Errorf("unexpected size 5, got %d", files[0].Size)
	}

	if files[0].ModTime.After(time.Now()) {
		t.Errorf("mod time is in the future: %v", files[0].ModTime)
	}
}

func TestScanWithPDFs(t *testing.T) {
	dir := setupTestDir(t)

	var called []string

	handle := func(file model.FileInfo) error {
		called = append(called, file.Name)
		return nil
	}

	err := scanWith(dir, handle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(called) != 1 {
		t.Errorf("expected 1 file to be handled, got %d", len(called))
	}

	if called[0] != "sample1.pdf" {
		t.Errorf("unexpected file handled: %s", called[0])
	}
}
