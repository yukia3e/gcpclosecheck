package analyzer

import (
	"sync"

	"github.com/yukia3e/gcpclosecheck/internal/config"
)

// ServiceRuleEngine はGCPサービスルールの管理エンジン
type ServiceRuleEngine struct {
	config *config.Config
	cache  map[string]string // serviceType -> cleanupMethod のキャッシュ
	mu     sync.RWMutex      // 並行アクセス制御
}

// NewServiceRuleEngine は新しいServiceRuleEngineを作成する
func NewServiceRuleEngine() *ServiceRuleEngine {
	return &ServiceRuleEngine{
		cache: make(map[string]string),
	}
}

// LoadDefaultRules はデフォルト設定を読み込む
func (sre *ServiceRuleEngine) LoadDefaultRules() error {
	return sre.LoadRules("")
}

// LoadRules は設定ファイルからルールを読み込む
// configPathが空またはファイルが存在しない場合はデフォルト設定を使用
func (sre *ServiceRuleEngine) LoadRules(configPath string) error {
	var err error
	
	if configPath == "" {
		// デフォルト設定を読み込み
		sre.config, err = config.LoadDefaultConfig()
	} else {
		// カスタム設定を読み込み、失敗時はデフォルトにフォールバック
		sre.config, err = config.LoadConfig(configPath)
		if err != nil {
			// フォールバック: デフォルト設定を読み込み
			sre.config, err = config.LoadDefaultConfig()
		}
	}
	
	if err != nil {
		return err
	}
	
	// 設定を検証
	return sre.config.Validate()
}

// GetCleanupMethod は指定されたサービスタイプの解放メソッドを取得する
func (sre *ServiceRuleEngine) GetCleanupMethod(serviceType string) (string, bool) {
	// キャッシュから確認
	sre.mu.RLock()
	if method, exists := sre.cache[serviceType]; exists {
		sre.mu.RUnlock()
		return method, true
	}
	sre.mu.RUnlock()
	
	// 設定から検索
	method, found := sre.findCleanupMethodFromConfig(serviceType)
	if found {
		// キャッシュに保存
		sre.mu.Lock()
		sre.cache[serviceType] = method
		sre.mu.Unlock()
	}
	
	return method, found
}

// IsCleanupRequired は指定されたサービスタイプで解放が必須かを判定する
func (sre *ServiceRuleEngine) IsCleanupRequired(serviceType string) bool {
	_, found := sre.GetCleanupMethod(serviceType)
	return found
}

// GetServiceRule はサービス名からServiceRuleを取得する
func (sre *ServiceRuleEngine) GetServiceRule(serviceName string) *ServiceRule {
	if sre.config == nil {
		return nil
	}
	
	configService := sre.config.GetService(serviceName)
	if configService == nil {
		return nil
	}
	
	// config.ServiceRule から analyzer.ServiceRule に変換
	rule := &ServiceRule{
		ServiceName:     configService.ServiceName,
		PackagePath:     configService.PackagePath,
		CreationFuncs:   configService.CreationFuncs,
		CleanupMethods:  make([]CleanupMethod, len(configService.CleanupMethods)),
	}
	
	for i, cm := range configService.CleanupMethods {
		rule.CleanupMethods[i] = CleanupMethod{
			Method:      cm.Method,
			Required:    cm.Required,
			Description: cm.Description,
		}
	}
	
	return rule
}

// findCleanupMethodFromConfig は設定からサービスタイプに対応する解放メソッドを検索する
func (sre *ServiceRuleEngine) findCleanupMethodFromConfig(serviceType string) (string, bool) {
	if sre.config == nil {
		return "", false
	}
	
	// サービスタイプから実際のサービス名を推定
	// 例: "spanner.Client" -> "spanner"
	serviceName := extractServiceName(serviceType)
	
	service := sre.config.GetService(serviceName)
	if service == nil {
		return "", false
	}
	
	// 最初の必須メソッドを返す
	for _, method := range service.CleanupMethods {
		if method.Required {
			return method.Method, true
		}
	}
	
	return "", false
}

// extractServiceName はサービスタイプからサービス名を抽出する
func extractServiceName(serviceType string) string {
	// 簡単な実装：パッケージ名部分を抽出
	// 例: "spanner.Client" -> "spanner"
	if len(serviceType) == 0 {
		return ""
	}
	
	// ドットで分割して最初の部分を取得
	for i, r := range serviceType {
		if r == '.' {
			return serviceType[:i]
		}
	}
	
	return serviceType
}

// ShouldExemptPackage は指定されたパッケージパスが例外対象かを判定する
func (sre *ServiceRuleEngine) ShouldExemptPackage(packagePath string) (bool, string) {
	if sre.config == nil {
		return false, ""
	}
	
	return sre.config.ShouldExemptPackage(packagePath)
}

// LoadPackageExceptions はパッケージ例外設定を読み込む
// 設定がない場合、またはパッケージ例外が定義されていない場合でもエラーにならない
func (sre *ServiceRuleEngine) LoadPackageExceptions(configPath string) error {
	// LoadRulesメソッドを再利用して設定を読み込み
	return sre.LoadRules(configPath)
}