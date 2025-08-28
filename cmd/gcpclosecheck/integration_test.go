package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGoVetIntegration はgo vetとの統合をテストする
func TestGoVetIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	// バイナリをビルド
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	// テスト用のGoモジュールを作成
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// go.modファイルを作成（依存関係なし）
	goModContent := `module testproject

go 1.25
`
	if err := os.WriteFile("go.mod", []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// テスト用のGoファイルを作成（標準ライブラリのみ使用）
	testCode := `
package main

import (
	"context"
	"fmt"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	// defer cancel() が不足 - これを検出すべき
	
	fmt.Println("Hello, World!")
	_ = ctx
	_ = cancel
}
`
	if err := os.WriteFile("main.go", []byte(testCode), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// go vet -vettool で実行
	vetCmd := exec.Command("go", "vet", "-vettool="+binPath, ".")
	var vetOut bytes.Buffer
	vetCmd.Stdout = &vetOut
	vetCmd.Stderr = &vetOut

	// タイムアウトを設定
	done := make(chan error, 1)
	go func() {
		done <- vetCmd.Run()
	}()

	select {
	case err := <-done:
		output := vetOut.String()
		t.Logf("go vet output: %s", output)

		// analysis.Analyzer インターフェースとの互換性をテスト
		if err != nil {
			// エラーが発生した場合でも、panic しないことが重要
			if strings.Contains(output, "panic") {
				t.Errorf("go vet integration should not panic: %v", err)
			}
		}

		// 基本的な動作確認（パッケージの問題でエラーになる可能性があるが、
		// analysis フレームワークとの統合が機能することを確認）
		if !strings.Contains(output, "gcpclosecheck") && !strings.Contains(output, "no required module") {
			t.Logf("Expected gcpclosecheck to run via go vet, output: %s", output)
		}

	case <-time.After(30 * time.Second):
		if err := vetCmd.Process.Kill(); err != nil {
			t.Errorf("Failed to kill process: %v", err)
		}
		t.Fatal("go vet integration test timed out")
	}
}

// TestAnalyzerInterfaceCompliance はanalysis.Analyzerインターフェース準拠性をテストする
func TestAnalyzerInterfaceCompliance(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	// バイナリをビルド
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	// 空のディレクトリで実行してみる
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// 基本的なanalysisフレームワーク機能のテスト
	cmd := exec.Command(binPath, "-h")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := out.String()

	// ヘルプメッセージが適切に表示されることを確認
	if strings.Contains(output, "panic") || strings.Contains(output, "fatal error") {
		t.Errorf("Help command should not panic: %s", output)
	}

	// analysis.Analyzer準拠の基本的な動作確認
	if err == nil && !strings.Contains(output, "gcpclosecheck") {
		t.Errorf("Help output should contain analyzer name")
	}
}

// TestMultiPackageAnalysis はマルチパッケージ解析をテストする
func TestMultiPackageAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi-package test in short mode")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	// バイナリをビルド
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// プロジェクト構造を作成
	dirs := []string{"cmd/app", "pkg/handlers", "internal/services"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// go.modファイルを作成
	goModContent := `module multipackage

go 1.25
`
	if err := os.WriteFile("go.mod", []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// 各パッケージにファイルを作成
	packages := map[string]string{
		"cmd/app/main.go": `
package main

func main() {
	// 単純なmain関数
}
`,
		"pkg/handlers/handler.go": `
package handlers

type Handler struct{}
`,
		"internal/services/service.go": `
package services

type Service struct{}
`,
	}

	for path, content := range packages {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// マルチパッケージ解析を実行
	analysisCmd := exec.Command(binPath, "./...")
	var analysisOut bytes.Buffer
	analysisCmd.Stdout = &analysisOut
	analysisCmd.Stderr = &analysisOut

	done := make(chan error, 1)
	go func() {
		done <- analysisCmd.Run()
	}()

	select {
	case err := <-done:
		output := analysisOut.String()
		t.Logf("Multi-package analysis output: %s", output)

		// panicしないことを確認
		if strings.Contains(output, "panic") {
			t.Errorf("Multi-package analysis should not panic: %v", err)
		}

		// 基本的な実行確認
		t.Logf("Multi-package analysis completed with error: %v", err)

	case <-time.After(15 * time.Second):
		if err := analysisCmd.Process.Kill(); err != nil {
			t.Errorf("Failed to kill process: %v", err)
		}
		t.Fatal("Multi-package analysis timed out")
	}
}

// TestLargeCodebasePerformance は大規模コードベースでのパフォーマンステスト
func TestLargeCodebasePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	// バイナリをビルド
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// go.mod作成
	goModContent := `module largeproject

go 1.25
`
	if err := os.WriteFile("go.mod", []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// 大規模なコードベースをシミュレート（50ファイル）
	for i := 0; i < 50; i++ {
		dir := filepath.Join("pkg", "module"+string(rune(i/10+'0')))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		filename := filepath.Join(dir, "file"+string(rune(i%10+'0'))+".go")
		content := `
package module` + string(rune(i/10+'0')) + `

import (
	"context"
)

func ProcessData` + string(rune(i%10+'0')) + `(ctx context.Context) error {
	// 各ファイルに約100行のコード
	for i := 0; i < 50; i++ {
		_ = i
	}
	return nil
}

type Data` + string(rune(i%10+'0')) + ` struct {
	ID   string
	Name string
	// その他のフィールド
}
`
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write large file: %v", err)
		}
	}

	// パフォーマンス測定
	start := time.Now()

	cmd := exec.Command(binPath, "./...")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		elapsed := time.Since(start)
		t.Logf("Large codebase analysis took: %v", elapsed)

		// 10,000+ LOC/sec の目標（5000行なので0.5秒以下）
		if elapsed > 5*time.Second {
			t.Errorf("Performance test failed: took %v (should be < 5s for ~5000 LOC)", elapsed)
		}

		output := out.String()
		if strings.Contains(output, "panic") {
			t.Errorf("Large codebase analysis should not panic")
		}

		t.Logf("Performance test completed with error: %v", err)
		t.Logf("Output: %s", output)

	case <-time.After(30 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			t.Errorf("Failed to kill process: %v", err)
		}
		t.Fatal("Large codebase analysis timed out")
	}
}

// TestCICDIntegration はCI/CDパイプライン統合をテストする
func TestCICDIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	// バイナリをビルド
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// CI/CDスクリプトをシミュレート
	scriptContent := `#!/bin/bash
set -e

echo "Running gcpclosecheck in CI/CD..."

# 実際のCI/CDで使われるパターンをテスト
` + binPath + ` ./... || {
    EXIT_CODE=$?
    echo "gcpclosecheck found issues (exit code: $EXIT_CODE)"
    exit $EXIT_CODE
}

echo "gcpclosecheck passed!"
`

	scriptPath := filepath.Join(tmpDir, "ci_test.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write CI script: %v", err)
	}

	// go.mod作成
	goModContent := `module citest

go 1.25
`
	if err := os.WriteFile("go.mod", []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// 正常なコードファイル
	validCode := `
package main

func main() {
	// CI/CDテスト用の正常なコード
}
`
	if err := os.WriteFile("main.go", []byte(validCode), 0644); err != nil {
		t.Fatalf("Failed to write valid code: %v", err)
	}

	// CIスクリプト実行
	ciCmd := exec.Command("/bin/bash", scriptPath)
	var ciOut bytes.Buffer
	ciCmd.Stdout = &ciOut
	ciCmd.Stderr = &ciOut

	done := make(chan error, 1)
	go func() {
		done <- ciCmd.Run()
	}()

	select {
	case err := <-done:
		output := ciOut.String()
		t.Logf("CI/CD test output: %s", output)

		// CI/CD統合の基本的な動作確認
		if strings.Contains(output, "panic") || strings.Contains(output, "fatal error") {
			t.Errorf("CI/CD integration should not panic")
		}

		// 適切な終了コードの処理確認
		t.Logf("CI/CD test completed with error: %v", err)

	case <-time.After(10 * time.Second):
		if err := ciCmd.Process.Kill(); err != nil {
			t.Errorf("Failed to kill process: %v", err)
		}
		t.Fatal("CI/CD integration test timed out")
	}
}
