package main

import (
	"go/build"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectStructure(t *testing.T) {
	// プロジェクトルートディレクトリの確認
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("作業ディレクトリを取得できません: %v", err)
	}

	// 必要なディレクトリ構造のテスト
	requiredDirs := []string{
		"cmd/gcpclosecheck",
		"internal/analyzer", 
		"internal/rules",
		"internal/config",
		"testdata/valid",
		"testdata/invalid",
	}

	for _, dir := range requiredDirs {
		dirPath := filepath.Join(wd, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("必要なディレクトリが存在しません: %s", dir)
		}
	}

	// go.mod ファイルの存在確認
	goModPath := filepath.Join(wd, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Error("go.mod ファイルが存在しません")
	}

	// main.go ファイルの存在確認
	mainGoPath := filepath.Join(wd, "cmd/gcpclosecheck/main.go")
	if _, err := os.Stat(mainGoPath); os.IsNotExist(err) {
		t.Error("cmd/gcpclosecheck/main.go ファイルが存在しません")
	}
}

func TestGoModuleConfiguration(t *testing.T) {
	// go.mod の内容検証
	wd, _ := os.Getwd()
	goModPath := filepath.Join(wd, "go.mod")
	
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("go.mod ファイルを読み込めません: %v", err)
	}

	expectedModule := "github.com/yukia3e/gcpclosecheck"
	if !containsString(string(content), expectedModule) {
		t.Errorf("go.mod にモジュールパス %s が含まれていません", expectedModule)
	}
}

func TestMainGoCompiles(t *testing.T) {
	// main.go がコンパイル可能かテスト
	wd, _ := os.Getwd()
	pkgDir := filepath.Join(wd, "cmd/gcpclosecheck")
	
	pkg, err := build.ImportDir(pkgDir, 0)
	if err != nil {
		t.Fatalf("cmd/gcpclosecheck パッケージをインポートできません: %v", err)
	}

	if pkg.Name != "main" {
		t.Errorf("パッケージ名が main ではありません: %s", pkg.Name)
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}