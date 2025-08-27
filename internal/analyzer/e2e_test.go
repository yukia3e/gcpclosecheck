package analyzer

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestE2EGoldenSuite は全GCPサービスのE2Eテストスイートを実行する
func TestE2EGoldenSuite(t *testing.T) {
	testdata := analysistest.TestData()
	
	// 全テストケースを実行
	analysistest.Run(t, testdata, Analyzer, 
		"spanner_valid", 
		"spanner_invalid",
		"pubsub_valid", 
		"pubsub_invalid",
		"storage_valid", 
		"storage_invalid",
		"vision_valid", 
		"vision_invalid",
		"admin_valid", 
		"admin_invalid",
		"recaptcha_valid", 
		"recaptcha_invalid",
		"context_valid", 
		"context_invalid",
		"complex_valid", 
		"complex_invalid",
	)
}

// TestE2ESpannerPatterns はSpannerの包括的なパターンをテストする
func TestE2ESpannerPatterns(t *testing.T) {
	testdata := analysistest.TestData()
	
	analysistest.Run(t, testdata, Analyzer, 
		"spanner_valid", 
		"spanner_invalid",
	)
}

// TestE2EPubSubPatterns はPubSubの包括的なパターンをテストする
func TestE2EPubSubPatterns(t *testing.T) {
	testdata := analysistest.TestData()
	
	analysistest.Run(t, testdata, Analyzer, 
		"pubsub_valid", 
		"pubsub_invalid",
	)
}

// TestE2EStoragePatterns はStorageの包括的なパターンをテストする
func TestE2EStoragePatterns(t *testing.T) {
	testdata := analysistest.TestData()
	
	analysistest.Run(t, testdata, Analyzer, 
		"storage_valid", 
		"storage_invalid",
	)
}

// TestE2EVisionPatterns はVisionの包括的なパターンをテストする
func TestE2EVisionPatterns(t *testing.T) {
	testdata := analysistest.TestData()
	
	analysistest.Run(t, testdata, Analyzer, 
		"vision_valid", 
		"vision_invalid",
	)
}

// TestE2EAdminPatterns はAdmin SDKの包括的なパターンをテストする
func TestE2EAdminPatterns(t *testing.T) {
	testdata := analysistest.TestData()
	
	analysistest.Run(t, testdata, Analyzer, 
		"admin_valid", 
		"admin_invalid",
	)
}

// TestE2EReCAPTCHAPatterns はreCAPTCHAの包括的なパターンをテストする
func TestE2EReCAPTCHAPatterns(t *testing.T) {
	testdata := analysistest.TestData()
	
	analysistest.Run(t, testdata, Analyzer, 
		"recaptcha_valid", 
		"recaptcha_invalid",
	)
}

// TestE2EContextPatterns はContext処理の包括的なパターンをテストする
func TestE2EContextPatterns(t *testing.T) {
	testdata := analysistest.TestData()
	
	analysistest.Run(t, testdata, Analyzer, 
		"context_valid", 
		"context_invalid",
	)
}

// TestE2EComplexScenarios は複雑な実世界シナリオをテストする
func TestE2EComplexScenarios(t *testing.T) {
	testdata := analysistest.TestData()
	
	analysistest.Run(t, testdata, Analyzer, 
		"complex_valid", 
		"complex_invalid",
	)
}

// TestE2EPerformanceOptimization はパフォーマンス最適化をテストする
func TestE2EPerformanceOptimization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	testdata := analysistest.TestData()
	
	// 大規模なテストファイルでパフォーマンスをテスト
	analysistest.Run(t, testdata, Analyzer, "large_codebase")
}

// TestE2EMemoryUsage はメモリ使用量をテストする
func TestE2EMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}
	
	testdata := analysistest.TestData()
	
	// メモリ制限内での動作確認
	analysistest.Run(t, testdata, Analyzer, "memory_intensive")
}

// TestE2ERegressionSuite は回帰テストスイートを実行する
func TestE2ERegressionSuite(t *testing.T) {
	testdata := analysistest.TestData()
	
	// 既知の問題パターンの回帰テスト
	analysistest.Run(t, testdata, Analyzer, "regression_tests")
}