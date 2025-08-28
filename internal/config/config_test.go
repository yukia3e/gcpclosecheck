package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// テスト用の一時設定ファイルを作成
	testYAML := `
services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions:
      - "NewClient"
      - "ReadOnlyTransaction"
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"
      - method: "Stop"
        required: true
        description: "RowIterator の停止"
  - service_name: "storage"
    package_path: "cloud.google.com/go/storage"
    creation_functions:
      - "NewClient"
      - "NewReader"
    cleanup_methods:
      - method: "Close"
        required: true
        description: "ストリーム接続のクローズ"
`

	// 一時ファイル作成
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")
	if err := os.WriteFile(configFile, []byte(testYAML), 0644); err != nil {
		t.Fatalf("テスト設定ファイルの作成に失敗: %v", err)
	}

	// 設定読み込みテスト
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("設定読み込みに失敗: %v", err)
	}

	if config == nil {
		t.Fatal("設定が nil です")
	}

	if len(config.Services) != 2 {
		t.Errorf("サービス数が期待値と異なります: got %d, want 2", len(config.Services))
	}

	// Spanner サービスの検証
	spannerService := findServiceByName(config.Services, "spanner")
	if spannerService == nil {
		t.Fatal("spanner サービスが見つかりません")
	}
	if spannerService.PackagePath != "cloud.google.com/go/spanner" {
		t.Errorf("spanner パッケージパスが異なります: %s", spannerService.PackagePath)
	}
	if len(spannerService.CreationFuncs) != 2 {
		t.Errorf("spanner 生成関数数が異なります: %d", len(spannerService.CreationFuncs))
	}
	if len(spannerService.CleanupMethods) != 2 {
		t.Errorf("spanner 解放メソッド数が異なります: %d", len(spannerService.CleanupMethods))
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	// デフォルト設定読み込みテスト
	config, err := LoadDefaultConfig()
	if err != nil {
		t.Fatalf("デフォルト設定読み込みに失敗: %v", err)
	}

	if config == nil {
		t.Fatal("デフォルト設定が nil です")
	}

	// 最低限のサービスが含まれているかチェック
	expectedServices := []string{"spanner", "storage", "pubsub", "vision"}
	for _, serviceName := range expectedServices {
		if findServiceByName(config.Services, serviceName) == nil {
			t.Errorf("期待されるサービス %s が見つかりません", serviceName)
		}
	}

	// パッケージ例外が設定されているかチェック
	if len(config.PackageExceptions) < 3 {
		t.Errorf("期待されるパッケージ例外数が不足: got %d, want >= 3", len(config.PackageExceptions))
	}

	// デフォルト例外の存在確認
	expectedExceptions := []string{"cmd_short_lived", "cloud_functions", "test_files"}
	for _, exceptionName := range expectedExceptions {
		if findExceptionByName(config.PackageExceptions, exceptionName) == nil {
			t.Errorf("期待される例外 %s が見つかりません", exceptionName)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	// 不正な設定のテスト
	invalidYAML := `
services:
  - service_name: ""
    package_path: "invalid"
    creation_functions: []
    cleanup_methods: []
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid_config.yaml")
	if err := os.WriteFile(configFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("不正テスト設定ファイルの作成に失敗: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("設定読み込みに失敗: %v", err)
	}

	// バリデーションテスト
	if err := config.Validate(); err == nil {
		t.Error("不正な設定でバリデーションエラーになるべきです")
	}
}

func TestConfigGetService(t *testing.T) {
	// GetService メソッドのテスト
	testYAML := `
services:
  - service_name: "test_service"
    package_path: "test/package"
    creation_functions: ["TestFunc"]
    cleanup_methods:
      - method: "TestClose"
        required: true
        description: "テストクローズ"
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_service_config.yaml")
	if err := os.WriteFile(configFile, []byte(testYAML), 0644); err != nil {
		t.Fatalf("テスト設定ファイルの作成に失敗: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("設定読み込みに失敗: %v", err)
	}

	// 存在するサービスの取得
	service := config.GetService("test_service")
	if service == nil {
		t.Error("存在するサービスが取得できません")
	}

	// 存在しないサービスの取得
	nonExistentService := config.GetService("non_existent")
	if nonExistentService != nil {
		t.Error("存在しないサービスが取得されました")
	}
}

// TestPackageExceptionRule は PackageExceptionRule 構造体の動作をテストする
func TestPackageExceptionRule(t *testing.T) {
	testYAML := `
services:
  - service_name: "pubsub"
    package_path: "cloud.google.com/go/pubsub"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"

package_exceptions:
  - name: "cmd_short_lived"
    pattern: "*/cmd/*"
    condition:
      type: "short_lived"
      description: "短命プログラム例外"
      enabled: true
  - name: "cloud_functions"
    pattern: "**/function/**"
    condition:
      type: "cloud_function"
      description: "Cloud Functions例外"
      enabled: true
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: false
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "exception_config.yaml")
	if err := os.WriteFile(configFile, []byte(testYAML), 0644); err != nil {
		t.Fatalf("テスト設定ファイルの作成に失敗: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("設定読み込みに失敗: %v", err)
	}

	// PackageExceptions が正しく読み込まれているかチェック
	if len(config.PackageExceptions) != 3 {
		t.Errorf("パッケージ例外数が期待値と異なります: got %d, want 3", len(config.PackageExceptions))
	}

	// cmd_short_lived 例外の検証
	cmdException := findExceptionByName(config.PackageExceptions, "cmd_short_lived")
	if cmdException == nil {
		t.Fatal("cmd_short_lived 例外が見つかりません")
	}
	if cmdException.Pattern != "*/cmd/*" {
		t.Errorf("cmd_short_lived パターンが異なります: got %s, want */cmd/*", cmdException.Pattern)
	}
	if cmdException.Condition.Type != "short_lived" {
		t.Errorf("cmd_short_lived タイプが異なります: got %s, want short_lived", cmdException.Condition.Type)
	}
	if !cmdException.Condition.Enabled {
		t.Error("cmd_short_lived が無効になっています")
	}

	// cloud_functions 例外の検証
	functionException := findExceptionByName(config.PackageExceptions, "cloud_functions")
	if functionException == nil {
		t.Fatal("cloud_functions 例外が見つかりません")
	}
	if functionException.Pattern != "**/function/**" {
		t.Errorf("cloud_functions パターンが異なります: got %s, want **/function/**", functionException.Pattern)
	}
	if functionException.Condition.Type != "cloud_function" {
		t.Errorf("cloud_functions タイプが異なります: got %s, want cloud_function", functionException.Condition.Type)
	}

	// test_files 例外の検証（デフォルト無効）
	testException := findExceptionByName(config.PackageExceptions, "test_files")
	if testException == nil {
		t.Fatal("test_files 例外が見つかりません")
	}
	if testException.Pattern != "**/*_test.go" {
		t.Errorf("test_files パターンが異なります: got %s, want **/*_test.go", testException.Pattern)
	}
	if testException.Condition.Type != "test" {
		t.Errorf("test_files タイプが異なります: got %s, want test", testException.Condition.Type)
	}
	if testException.Condition.Enabled {
		t.Error("test_files がデフォルトで有効になっています")
	}
}

// TestPackageExceptionRuleValidation は PackageExceptionRule のバリデーションをテストする
func TestPackageExceptionRuleValidation(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_exception",
			yaml: `
services:
  - service_name: "test"
    package_path: "test"
    creation_functions: ["Test"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "テスト"

package_exceptions:
  - name: "valid"
    pattern: "*/test/*"
    condition:
      type: "short_lived"
      description: "テスト例外"
      enabled: true
`,
			expectError: false,
		},
		{
			name: "empty_exception_name",
			yaml: `
services:
  - service_name: "test"
    package_path: "test"
    creation_functions: ["Test"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "テスト"

package_exceptions:
  - name: ""
    pattern: "*/test/*"
    condition:
      type: "short_lived"
      description: "テスト例外"
      enabled: true
`,
			expectError: true,
			errorMsg:    "例外名が空です",
		},
		{
			name: "empty_pattern",
			yaml: `
services:
  - service_name: "test"
    package_path: "test"
    creation_functions: ["Test"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "テスト"

package_exceptions:
  - name: "test"
    pattern: ""
    condition:
      type: "short_lived"
      description: "テスト例外"
      enabled: true
`,
			expectError: true,
			errorMsg:    "パターンが空です",
		},
		{
			name: "invalid_condition_type",
			yaml: `
services:
  - service_name: "test"
    package_path: "test"
    creation_functions: ["Test"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "テスト"

package_exceptions:
  - name: "test"
    pattern: "*/test/*"
    condition:
      type: "invalid_type"
      description: "テスト例外"
      enabled: true
`,
			expectError: true,
			errorMsg:    "不正な条件タイプです",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "test_config.yaml")
			if err := os.WriteFile(configFile, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("テスト設定ファイルの作成に失敗: %v", err)
			}

			config, err := LoadConfig(configFile)
			if err != nil {
				t.Fatalf("設定読み込みに失敗: %v", err)
			}

			err = config.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("バリデーションエラーが期待されましたが、エラーが発生しませんでした")
				} else if !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("期待されるエラーメッセージが含まれていません: got %v, want contains %s", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("バリデーションエラーが発生しました: %v", err)
				}
			}
		})
	}
}

// TestShouldExemptPackage は ShouldExemptPackage メソッドをテストする
func TestShouldExemptPackage(t *testing.T) {
	testYAML := `
services:
  - service_name: "pubsub"
    package_path: "cloud.google.com/go/pubsub"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"

package_exceptions:
  - name: "cmd_short_lived"
    pattern: "*/cmd/*"
    condition:
      type: "short_lived"
      description: "短命プログラム例外"
      enabled: true
  - name: "cloud_functions"
    pattern: "**/function/**"
    condition:
      type: "cloud_function"
      description: "Cloud Functions例外"
      enabled: true
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: false
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "exempt_config.yaml")
	if err := os.WriteFile(configFile, []byte(testYAML), 0644); err != nil {
		t.Fatalf("テスト設定ファイルの作成に失敗: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("設定読み込みに失敗: %v", err)
	}

	tests := []struct {
		name         string
		packagePath  string
		shouldExempt bool
		exemptReason string
	}{
		{
			name:         "cmd_package_should_exempt",
			packagePath:  "github.com/example/project/cmd/server",
			shouldExempt: true,
			exemptReason: "短命プログラム例外",
		},
		{
			name:         "function_package_should_exempt",
			packagePath:  "github.com/example/project/internal/function/handler",
			shouldExempt: true,
			exemptReason: "Cloud Functions例外",
		},
		{
			name:         "test_file_should_not_exempt_when_disabled",
			packagePath:  "github.com/example/project/internal/handler_test.go",
			shouldExempt: false,
			exemptReason: "",
		},
		{
			name:         "regular_package_should_not_exempt",
			packagePath:  "github.com/example/project/internal/handler",
			shouldExempt: false,
			exemptReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldExempt, reason := config.ShouldExemptPackage(tt.packagePath)
			if shouldExempt != tt.shouldExempt {
				t.Errorf("ShouldExemptPackage(%s): got %v, want %v", tt.packagePath, shouldExempt, tt.shouldExempt)
			}
			if reason != tt.exemptReason {
				t.Errorf("ShouldExemptPackage(%s): reason got %s, want %s", tt.packagePath, reason, tt.exemptReason)
			}
		})
	}
}

// ヘルパー関数: サービス名でサービスを検索
func findServiceByName(services []ServiceRule, name string) *ServiceRule {
	for i := range services {
		if services[i].ServiceName == name {
			return &services[i]
		}
	}
	return nil
}

// ヘルパー関数: 例外名でパッケージ例外を検索
func findExceptionByName(exceptions []PackageExceptionRule, name string) *PackageExceptionRule {
	for i := range exceptions {
		if exceptions[i].Name == name {
			return &exceptions[i]
		}
	}
	return nil
}

// ヘルパー関数: 文字列が含まれているかチェック
func containsString(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(strings.Contains(s, substr))))
}
