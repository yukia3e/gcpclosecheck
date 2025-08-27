package config

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// 有効な例外タイプの定義
const (
	ExceptionTypeShortLived    = "short_lived"    // 短命プログラム（cmdパッケージ等）
	ExceptionTypeCloudFunction = "cloud_function" // Cloud Functions実行環境
	ExceptionTypeTest          = "test"           // テストコード
)

// validExceptionTypes は有効な例外タイプのリスト
var validExceptionTypes = []string{
	ExceptionTypeShortLived,
	ExceptionTypeCloudFunction,
	ExceptionTypeTest,
}

// ServiceRule は GCP サービス固有の解放ルール定義を表す
type ServiceRule struct {
	ServiceName    string          `yaml:"service_name"`     // サービス名
	PackagePath    string          `yaml:"package_path"`     // パッケージパス
	CreationFuncs  []string        `yaml:"creation_functions"` // 生成関数一覧
	CleanupMethods []CleanupMethod `yaml:"cleanup_methods"`  // 解放メソッド一覧
}

// CleanupMethod は解放メソッドの詳細情報を表す
type CleanupMethod struct {
	Method      string `yaml:"method"`      // メソッド名
	Required    bool   `yaml:"required"`    // 必須かどうか
	Description string `yaml:"description"` // 説明
}

// ExceptionCondition はパッケージ例外の条件を表す
// 例外タイプとその有効性、説明を管理する
type ExceptionCondition struct {
	Type        string `yaml:"type"`        // 例外タイプ (short_lived, cloud_function, test)
	Description string `yaml:"description"` // 例外の説明
	Enabled     bool   `yaml:"enabled"`     // この例外が有効かどうか
}

// PackageExceptionRule はパッケージベースの例外ルールを表す
// 特定のパッケージパスパターンに対してリソース終了チェックを例外的に緩和するためのルール
type PackageExceptionRule struct {
	Name      string             `yaml:"name"`      // 例外ルール名（識別用）
	Pattern   string             `yaml:"pattern"`   // パッケージパスパターン（glob形式: **/function/**, */cmd/*等）
	Condition ExceptionCondition `yaml:"condition"` // 例外適用条件
}

// Config はツール全体の設定を表す
type Config struct {
	Services          []ServiceRule            `yaml:"services"`
	PackageExceptions []PackageExceptionRule   `yaml:"package_exceptions,omitempty"`
}

// LoadConfig は指定されたパスから設定ファイルを読み込む
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, errors.New("設定ファイルパスが空です")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("YAML解析に失敗: %w", err)
	}

	return &config, nil
}

//go:embed rules.yaml
var defaultRules embed.FS

// LoadDefaultConfig はデフォルトの設定を読み込む
func LoadDefaultConfig() (*Config, error) {
	data, err := defaultRules.ReadFile("rules.yaml")
	if err != nil {
		return nil, fmt.Errorf("デフォルト設定ファイルの読み込みに失敗: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("デフォルト設定のYAML解析に失敗: %w", err)
	}

	return &config, nil
}

// Validate は設定の妥当性を検証する
func (c *Config) Validate() error {
	if len(c.Services) == 0 {
		return errors.New("サービス定義が空です")
	}

	for i, service := range c.Services {
		if service.ServiceName == "" {
			return fmt.Errorf("サービス[%d]: サービス名が空です", i)
		}
		if service.PackagePath == "" {
			return fmt.Errorf("サービス[%d](%s): パッケージパスが空です", i, service.ServiceName)
		}
		if len(service.CreationFuncs) == 0 {
			return fmt.Errorf("サービス[%d](%s): 生成関数が定義されていません", i, service.ServiceName)
		}
		if len(service.CleanupMethods) == 0 {
			return fmt.Errorf("サービス[%d](%s): 解放メソッドが定義されていません", i, service.ServiceName)
		}

		// 解放メソッドの検証
		for j, method := range service.CleanupMethods {
			if method.Method == "" {
				return fmt.Errorf("サービス[%d](%s): 解放メソッド[%d]のメソッド名が空です", i, service.ServiceName, j)
			}
		}
	}

	// パッケージ例外の検証
	for i, exception := range c.PackageExceptions {
		if exception.Name == "" {
			return fmt.Errorf("パッケージ例外[%d]: 例外名が空です", i)
		}
		if exception.Pattern == "" {
			return fmt.Errorf("パッケージ例外[%d](%s): パターンが空です", i, exception.Name)
		}
		
		// 例外条件タイプの検証
		if !isValidExceptionType(exception.Condition.Type) {
			return fmt.Errorf("パッケージ例外[%d](%s): 不正な条件タイプです: %s (有効なタイプ: %v)", 
				i, exception.Name, exception.Condition.Type, validExceptionTypes)
		}
	}

	return nil
}

// GetService は指定された名前のサービスを取得する
func (c *Config) GetService(serviceName string) *ServiceRule {
	for i := range c.Services {
		if c.Services[i].ServiceName == serviceName {
			return &c.Services[i]
		}
	}
	return nil
}

// GetServiceByPackagePath はパッケージパスでサービスを取得する
func (c *Config) GetServiceByPackagePath(packagePath string) *ServiceRule {
	for i := range c.Services {
		if c.Services[i].PackagePath == packagePath {
			return &c.Services[i]
		}
	}
	return nil
}

// HasService は指定された名前のサービスが存在するかチェックする
func (c *Config) HasService(serviceName string) bool {
	return c.GetService(serviceName) != nil
}

// ShouldExemptPackage は指定されたパッケージパスが例外対象かチェックする
func (c *Config) ShouldExemptPackage(packagePath string) (bool, string) {
	for _, exception := range c.PackageExceptions {
		if !exception.Condition.Enabled {
			continue
		}
		
		// パターンマッチング（簡単なglob実装）
		if matchPattern(exception.Pattern, packagePath) {
			return true, exception.Condition.Description
		}
	}
	
	return false, ""
}

// ShouldExemptFilePath は指定されたファイルパスが例外対象かチェックする
func (c *Config) ShouldExemptFilePath(filePath string) (bool, string) {
	for _, exception := range c.PackageExceptions {
		if !exception.Condition.Enabled {
			continue
		}
		
		// ファイルパスパターンマッチング
		if matchPattern(exception.Pattern, filePath) {
			return true, exception.Condition.Description
		}
	}
	
	return false, ""
}

// matchPattern は簡単なglobパターンマッチングを行う
func matchPattern(pattern, str string) bool {
	// **/* パターン（任意のディレクトリ階層）
	// 例: "**/function/**" は "github.com/example/project/internal/function/handler" にマッチ
	if strings.Contains(pattern, "**/") {
		// パターンを分割
		beforeAfter := strings.Split(pattern, "**/")
		if len(beforeAfter) == 2 {
			before := beforeAfter[0]
			after := beforeAfter[1]
			
			// afterの**も処理
			if strings.HasSuffix(after, "/**") {
				after = strings.TrimSuffix(after, "/**")
			} else if strings.HasPrefix(after, "*/") {
				after = strings.TrimPrefix(after, "*/")
			}
			
			// beforeで始まり、afterが含まれる
			hasPrefix := before == "" || strings.HasPrefix(str, before)
			var hasAfter bool
			
			// afterがワイルドカードを含む場合の特別処理
			if strings.HasPrefix(after, "*") {
				suffix := strings.TrimPrefix(after, "*")
				hasAfter = strings.HasSuffix(str, suffix)
			} else {
				hasAfter = after == "" || strings.Contains(str, after)
			}
			
			return hasPrefix && hasAfter
		}
	}
	
	// */ パターン（単一ディレクトリ階層） 
	// 例: "*/cmd/*" は "github.com/example/project/cmd/server" にマッチ
	if strings.Contains(pattern, "*/") && !strings.Contains(pattern, "**/") {
		parts := strings.Split(pattern, "*/")
		if len(parts) == 2 {
			before := parts[0]
			after := parts[1]
			
			// afterの後ろの*も処理
			if strings.HasSuffix(after, "/*") {
				after = strings.TrimSuffix(after, "/*")
			}
			
			if before == "" {
				// "*/after" の形式 - afterが含まれる
				return strings.Contains(str, after)
			} else {
				// "before/*/after" の形式
				// beforeで始まり、afterが含まれる
				return strings.HasPrefix(str, before) && strings.Contains(str, after)
			}
		}
	}
	
	// **/*_test.go のようなパターン
	if strings.HasPrefix(pattern, "**/") {
		suffix := strings.TrimPrefix(pattern, "**/")
		// 先頭の*を処理
		if strings.HasPrefix(suffix, "*") {
			suffix = strings.TrimPrefix(suffix, "*")
			return strings.HasSuffix(str, suffix)
		}
		return strings.HasSuffix(str, suffix)
	}
	
	// 単純な文字列マッチ
	return str == pattern
}

// isValidExceptionType は指定された例外タイプが有効かチェックする
func isValidExceptionType(exceptionType string) bool {
	for _, validType := range validExceptionTypes {
		if exceptionType == validType {
			return true
		}
	}
	return false
}