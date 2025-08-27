package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServiceRuleEngine_LoadRules(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() (string, func())
		wantErr     bool
		errorMsg    string
	}{
		{
			name: "正常な設定ファイル読み込み",
			setupConfig: func() (string, func()) {
				// テスト用一時設定ファイル作成
				testYAML := `
services:
  - service_name: "test_spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions:
      - "NewClient"
    cleanup_methods:
      - method: "Close"
        required: true
        description: "テスト用クローズ"
`
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "test_config.yaml")
				err := os.WriteFile(configFile, []byte(testYAML), 0644)
				if err != nil {
					t.Fatalf("テスト設定ファイル作成失敗: %v", err)
				}
				return configFile, func() {} // クリーンアップ不要（t.TempDirが自動削除）
			},
			wantErr: false,
		},
		{
			name: "設定ファイルが存在しない場合のフォールバック",
			setupConfig: func() (string, func()) {
				return "/nonexistent/config.yaml", func() {}
			},
			wantErr: false, // フォールバックでデフォルト設定を使用
		},
		{
			name: "不正なYAMLファイルの場合（フォールバック）",
			setupConfig: func() (string, func()) {
				invalidYAML := `
services:
  - invalid_yaml: [
`
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "invalid_config.yaml")
				err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
				if err != nil {
					t.Fatalf("不正テスト設定ファイル作成失敗: %v", err)
				}
				return configFile, func() {}
			},
			wantErr: false, // フォールバックでデフォルト設定を使用
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, cleanup := tt.setupConfig()
			defer cleanup()

			engine := NewServiceRuleEngine()
			err := engine.LoadRules(configPath)

			if tt.wantErr {
				if err == nil {
					t.Error("エラーが期待されましたが、エラーが発生しませんでした")
				}
			} else {
				if err != nil {
					t.Errorf("予期しないエラー: %v", err)
				}
			}
		})
	}
}

func TestServiceRuleEngine_GetCleanupMethod(t *testing.T) {
	// テスト用ルール設定
	engine := NewServiceRuleEngine()
	
	// デフォルト設定を読み込み
	err := engine.LoadRules("")
	if err != nil {
		t.Fatalf("デフォルト設定読み込み失敗: %v", err)
	}

	tests := []struct {
		name        string
		serviceType string
		wantMethod  string
		wantFound   bool
	}{
		{
			name:        "Spannerサービス - Close メソッド",
			serviceType: "spanner.Client",
			wantMethod:  "Close",
			wantFound:   true,
		},
		{
			name:        "Storageサービス - Close メソッド",
			serviceType: "storage.Client",
			wantMethod:  "Close",
			wantFound:   true,
		},
		{
			name:        "存在しないサービス",
			serviceType: "unknown.Service",
			wantMethod:  "",
			wantFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, found := engine.GetCleanupMethod(tt.serviceType)
			
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
			
			if method != tt.wantMethod {
				t.Errorf("method = %v, want %v", method, tt.wantMethod)
			}
		})
	}
}

func TestServiceRuleEngine_IsCleanupRequired(t *testing.T) {
	engine := NewServiceRuleEngine()
	
	// デフォルト設定を読み込み
	err := engine.LoadRules("")
	if err != nil {
		t.Fatalf("デフォルト設定読み込み失敗: %v", err)
	}

	tests := []struct {
		name        string
		serviceType string
		want        bool
	}{
		{
			name:        "Spanner Client - 必須",
			serviceType: "spanner.Client",
			want:        true,
		},
		{
			name:        "Storage Client - 必須",
			serviceType: "storage.Client",
			want:        true,
		},
		{
			name:        "存在しないサービス - 不要",
			serviceType: "unknown.Service",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.IsCleanupRequired(tt.serviceType)
			if result != tt.want {
				t.Errorf("IsCleanupRequired() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestServiceRuleEngine_CachePerformance(t *testing.T) {
	engine := NewServiceRuleEngine()
	
	// デフォルト設定を読み込み
	err := engine.LoadRules("")
	if err != nil {
		t.Fatalf("デフォルト設定読み込み失敗: %v", err)
	}

	serviceType := "spanner.Client"
	
	// 初回アクセス
	method1, found1 := engine.GetCleanupMethod(serviceType)
	if !found1 {
		t.Fatal("サービスが見つかりません")
	}
	
	// 2回目アクセス（キャッシュから取得）
	method2, found2 := engine.GetCleanupMethod(serviceType)
	if !found2 {
		t.Fatal("キャッシュからの取得に失敗")
	}
	
	if method1 != method2 {
		t.Errorf("キャッシュの一貫性が保たれていません: %v != %v", method1, method2)
	}
}

func TestServiceRuleEngine_GetServiceRule(t *testing.T) {
	engine := NewServiceRuleEngine()
	
	// デフォルト設定を読み込み
	err := engine.LoadRules("")
	if err != nil {
		t.Fatalf("デフォルト設定読み込み失敗: %v", err)
	}

	tests := []struct {
		name        string
		serviceName string
		wantFound   bool
	}{
		{
			name:        "存在するサービス - spanner",
			serviceName: "spanner",
			wantFound:   true,
		},
		{
			name:        "存在するサービス - storage",
			serviceName: "storage",
			wantFound:   true,
		},
		{
			name:        "存在しないサービス",
			serviceName: "nonexistent",
			wantFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := engine.GetServiceRule(tt.serviceName)
			found := rule != nil
			
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
			
			if found && rule.ServiceName != tt.serviceName {
				t.Errorf("ServiceName = %v, want %v", rule.ServiceName, tt.serviceName)
			}
		})
	}
}

func TestServiceRuleEngine_ShouldExemptPackage(t *testing.T) {
	engine := NewServiceRuleEngine()
	
	// デフォルト設定を読み込み（package_exceptionsを含む）
	err := engine.LoadDefaultRules()
	if err != nil {
		t.Fatalf("デフォルト設定読み込み失敗: %v", err)
	}

	tests := []struct {
		name        string
		packagePath string
		wantExempt  bool
		wantReason  string
	}{
		{
			name:        "cmd パッケージ - 例外対象",
			packagePath: "github.com/example/project/cmd/server",
			wantExempt:  true,
			wantReason:  "短命プログラム例外",
		},
		{
			name:        "function パッケージ - 例外対象",
			packagePath: "github.com/example/project/internal/function/handler",
			wantExempt:  true,
			wantReason:  "Cloud Functions例外",
		},
		{
			name:        "test ファイル - 例外対象外（デフォルト無効）",
			packagePath: "github.com/example/project/pkg/test_util.go",
			wantExempt:  false,
			wantReason:  "",
		},
		{
			name:        "通常のパッケージ - 例外対象外",
			packagePath: "github.com/example/project/pkg/service",
			wantExempt:  false,
			wantReason:  "",
		},
		{
			name:        "複雑なcmdパターン",
			packagePath: "github.com/example/nested/cmd/migrate",
			wantExempt:  true,
			wantReason:  "短命プログラム例外",
		},
		{
			name:        "複雑なfunctionパターン",
			packagePath: "github.com/example/nested/internal/function/webhook/handler",
			wantExempt:  true,
			wantReason:  "Cloud Functions例外",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exempt, reason := engine.ShouldExemptPackage(tt.packagePath)
			
			if exempt != tt.wantExempt {
				t.Errorf("ShouldExemptPackage() exempt = %v, want %v", exempt, tt.wantExempt)
			}
			
			if reason != tt.wantReason {
				t.Errorf("ShouldExemptPackage() reason = %v, want %v", reason, tt.wantReason)
			}
		})
	}
}

func TestServiceRuleEngine_LoadPackageExceptions(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() (string, func())
		wantErr     bool
	}{
		{
			name: "有効なpackage_exceptions設定",
			setupConfig: func() (string, func()) {
				testYAML := `
services:
  - service_name: "test_service"
    package_path: "cloud.google.com/go/test"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "テスト用クローズ"

package_exceptions:
  - name: "test_cmd"
    pattern: "*/cmd/*"
    condition:
      type: "short_lived"
      description: "テスト用短命プログラム例外"
      enabled: true
  - name: "test_function"
    pattern: "**/function/**"
    condition:
      type: "cloud_function"
      description: "テスト用Cloud Functions例外"
      enabled: true
`
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "test_config.yaml")
				err := os.WriteFile(configFile, []byte(testYAML), 0644)
				if err != nil {
					t.Fatalf("テスト設定ファイル作成失敗: %v", err)
				}
				return configFile, func() {}
			},
			wantErr: false,
		},
		{
			name: "package_exceptionsなしの設定",
			setupConfig: func() (string, func()) {
				testYAML := `
services:
  - service_name: "test_service"
    package_path: "cloud.google.com/go/test"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "テスト用クローズ"
`
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "test_config.yaml")
				err := os.WriteFile(configFile, []byte(testYAML), 0644)
				if err != nil {
					t.Fatalf("テスト設定ファイル作成失敗: %v", err)
				}
				return configFile, func() {}
			},
			wantErr: false, // package_exceptionsが空でもエラーにならない
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, cleanup := tt.setupConfig()
			defer cleanup()

			engine := NewServiceRuleEngine()
			err := engine.LoadPackageExceptions(configPath)

			if tt.wantErr {
				if err == nil {
					t.Error("エラーが期待されましたが、エラーが発生しませんでした")
				}
			} else {
				if err != nil {
					t.Errorf("予期しないエラー: %v", err)
				}
			}
		})
	}
}