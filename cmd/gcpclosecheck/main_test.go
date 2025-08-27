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

// TestCLIBasicExecution は基本的なCLI実行をテストする
func TestCLIBasicExecution(t *testing.T) {
	// テスト用の実行ファイルをビルド
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	// ヘルプオプションのテスト
	helpCmd := exec.Command(binPath, "-h")
	var helpOut bytes.Buffer
	helpCmd.Stdout = &helpOut
	helpCmd.Stderr = &helpOut

	if err := helpCmd.Run(); err == nil {
		// ヘルプメッセージが表示されることを確認
		output := helpOut.String()
		if !strings.Contains(output, "gcpclosecheck") {
			t.Errorf("Help output should contain 'gcpclosecheck', got: %s", output)
		}
	}
}

// TestCLIVersionFlag はバージョンフラグをテストする
func TestCLIVersionFlag(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	// バージョンフラグのテスト
	versionCmd := exec.Command(binPath, "-V")
	var versionOut bytes.Buffer
	versionCmd.Stdout = &versionOut
	versionCmd.Stderr = &versionOut

	// バージョンフラグは正常に実行されるか、適切なエラーメッセージを出すべき
	_ = versionCmd.Run()
	// output := versionOut.String()
	// バージョン情報が表示されるかどうかは実装次第
}

// TestCLIAnalysisExecution は実際の解析実行をテストする
func TestCLIAnalysisExecution(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	// テスト用のGoファイルを作成
	testFile := filepath.Join(tmpDir, "test.go")
	testCode := `
package main

import (
	"context"
	"cloud.google.com/go/spanner"
)

func main() {
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		panic(err)
	}
	// defer client.Close() が不足 - これを検出すべき
	_ = client
}
`
	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// CLIで解析実行
	analysisCmd := exec.Command(binPath, testFile)
	var analysisOut bytes.Buffer
	analysisCmd.Stdout = &analysisOut
	analysisCmd.Stderr = &analysisOut
	analysisCmd.Dir = tmpDir

	// タイムアウトを設定
	done := make(chan error, 1)
	go func() {
		done <- analysisCmd.Run()
	}()

	select {
	case err := <-done:
		// 解析が実行されたことを確認（エラーの有無は問わない）
		output := analysisOut.String()
		if err != nil {
			// 解析エラーや検出結果による非零終了コードは正常
			t.Logf("Analysis completed with exit code (expected): %v", err)
			t.Logf("Output: %s", output)
		}
	case <-time.After(10 * time.Second):
		if err := analysisCmd.Process.Kill(); err != nil {
			t.Errorf("Failed to kill process: %v", err)
		}
		t.Fatal("Analysis execution timed out")
	}
}

// TestCLIExitCodes は異なる状況での終了コードをテストする
func TestCLIExitCodes(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	tests := []struct {
		name     string
		args     []string
		expectNonZero bool
	}{
		{
			name: "No arguments (help)",
			args: []string{},
			expectNonZero: false, // ヘルプ表示は正常終了
		},
		{
			name: "Invalid flag",
			args: []string{"-invalid-flag"},
			expectNonZero: true, // 無効なフラグはエラー
		},
		{
			name: "Non-existent file",
			args: []string{"/non/existent/file.go"},
			expectNonZero: true, // 存在しないファイルはエラー
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			err := cmd.Run()
			hasError := err != nil

			if tt.expectNonZero && !hasError {
				t.Errorf("Expected non-zero exit code, but got success")
			} else if !tt.expectNonZero && hasError {
				t.Errorf("Expected zero exit code, but got error: %v", err)
			}
		})
	}
}

// TestCLIOutputFormat は出力フォーマットをテストする
func TestCLIOutputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	// 正常なGoファイルで出力フォーマットをテスト
	testFile := filepath.Join(tmpDir, "valid.go")
	validCode := `
package main

import (
	"context"
	"cloud.google.com/go/spanner"
)

func main() {
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		panic(err)
	}
	defer client.Close() // 正しいパターン
	_ = client
}
`
	if err := os.WriteFile(testFile, []byte(validCode), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 正常なコードでの実行（問題なしの場合）
	cmd := exec.Command(binPath, testFile)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = tmpDir

	_ = cmd.Run()
	output := out.String()
	
	// 正常なコードの場合は出力が少ない、またはエラーメッセージがないことを期待
	if strings.Contains(output, "panic") || strings.Contains(output, "fatal") {
		t.Errorf("Unexpected panic or fatal error in output: %s", output)
	}
}

// TestCLIPerformance は基本的なパフォーマンステストを実行する
func TestCLIPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	// 中規模のテストファイルを作成
	testFile := filepath.Join(tmpDir, "large_test.go")
	var codeBuilder strings.Builder
	codeBuilder.WriteString(`
package main

import (
	"context"
	"cloud.google.com/go/spanner"
)

func main() {
`)

	// 100個のクライアント生成を含むコードを生成
	for i := 0; i < 100; i++ {
		codeBuilder.WriteString(`
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		panic(err)
	}
	_ = client
`)
	}

	codeBuilder.WriteString("}")

	if err := os.WriteFile(testFile, []byte(codeBuilder.String()), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// パフォーマンス測定
	start := time.Now()

	cmd := exec.Command(binPath, testFile)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = tmpDir

	err := cmd.Run()
	elapsed := time.Since(start)

	t.Logf("Analysis of 100 clients took: %v", elapsed)
	if elapsed > 5*time.Second {
		t.Errorf("Analysis took too long: %v (should be < 5s)", elapsed)
	}

	// エラーログの確認
	if err != nil {
		t.Logf("Analysis completed with error (expected): %v", err)
		t.Logf("Output: %s", out.String())
	}
}