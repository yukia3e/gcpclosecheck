package config

import (
	"testing"
	"golang.org/x/tools/go/analysis"
)

// TestDiagnosticLevel は診断レベルの制御機能テスト
func TestDiagnosticLevel(t *testing.T) {
	tests := []struct {
		name          string
		level         DiagnosticLevel
		diagnostic    analysis.Diagnostic
		shouldInclude bool
		description   string
	}{
		{
			name:  "Error level diagnostic always included",
			level: ErrorLevel,
			diagnostic: analysis.Diagnostic{
				Message: "Critical resource leak detected",
			},
			shouldInclude: true,
			description:   "エラーレベルは常に表示",
		},
		{
			name:  "Warning level filtered by setting", 
			level: WarningLevel,
			diagnostic: analysis.Diagnostic{
				Message: "Potential resource leak (potential false positive)",
			},
			shouldInclude: false, // 設定でフィルタ
			description:   "警告レベルは設定でフィルタ可能",
		},
		{
			name:  "Info level diagnostic included at info level",
			level: InfoLevel,
			diagnostic: analysis.Diagnostic{
				Message: "Resource usage suggestion",
			},
			shouldInclude: true, // InfoLevelなので含める
			description:   "情報レベル設定では情報診断も含める",
		},
		{
			name:  "Info level diagnostic filtered by warning level",
			level: WarningLevel, // 警告レベル設定
			diagnostic: analysis.Diagnostic{
				Message: "Resource usage suggestion", // 情報レベル診断
			},
			shouldInclude: false, // 警告レベル設定なので情報は除外
			description:   "警告レベル設定では情報診断を除外",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			
			// DiagnosticFilter が未実装のため失敗することを期待
			filter := NewDiagnosticFilter(tt.level)
			included := filter.ShouldIncludeDiagnostic(tt.diagnostic)
			
			if included != tt.shouldInclude {
				t.Errorf("Expected %v, got %v for level %v", 
					tt.shouldInclude, included, tt.level)
			}
		})
	}
}

// TestPotentialFalsePositiveMarking は偽陽性マーク機能のテスト
func TestPotentialFalsePositiveMarking(t *testing.T) {
	tests := []struct {
		name             string
		message          string
		expectedMarked   bool
		description      string
	}{
		{
			name:           "Clear false positive indicator",
			message:        "Resource leak detected (potential false positive)",
			expectedMarked: true,
			description:    "明確な偽陽性示唆があるメッセージ",
		},
		{
			name:           "Uncertain resource management",
			message:        "Resource usage pattern unclear",
			expectedMarked: true, 
			description:    "不明確なパターンでの偽陽性疑義",
		},
		{
			name:           "Definitive leak detection",
			message:        "Resource leak: missing defer Close()",
			expectedMarked: false,
			description:    "確実なリーク検出は偽陽性マークなし",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			
			// PotentialFalsePositiveDetector が未実装のため失敗を期待
			detector := NewPotentialFalsePositiveDetector()
			marked := detector.IsPotentialFalsePositive(tt.message)
			
			if marked != tt.expectedMarked {
				t.Errorf("Expected %v, got %v for message: %s", 
					tt.expectedMarked, marked, tt.message)
			}
		})
	}
}

// TestDiagnosticConfig は診断設定の読み込みテスト
func TestDiagnosticConfig(t *testing.T) {
	configYAML := `
diagnostics:
  level: "warning"
  include_suggestions: true
  include_escape_reasons: true
  confidence_threshold: 0.8
  potential_false_positive_detection: true
  custom_filters:
    - pattern: "potential false positive"
      action: "mark_uncertain"
    - pattern: "framework managed"
      action: "suppress"
`

	// Config読み込み機能が未実装のため失敗を期待
	config, err := LoadDiagnosticConfigFromYAML([]byte(configYAML))
	if err != nil {
		t.Fatalf("Failed to load diagnostic config: %v", err)
	}

	// 設定値の検証
	if config.Level != "warning" {
		t.Errorf("Expected level 'warning', got %s", config.Level)
	}

	if !config.IncludeSuggestions {
		t.Error("Expected IncludeSuggestions to be true")
	}

	if config.ConfidenceThreshold != 0.8 {
		t.Errorf("Expected confidence threshold 0.8, got %f", config.ConfidenceThreshold)
	}

	if len(config.CustomFilters) != 2 {
		t.Errorf("Expected 2 custom filters, got %d", len(config.CustomFilters))
	}

	t.Logf("✅ Diagnostic config structure validated")
}

// TestIntegratedDiagnosticFiltering は統合診断フィルタリングのテスト
func TestIntegratedDiagnosticFiltering(t *testing.T) {
	config := &DiagnosticConfig{
		Level:                           "info", // より低いレベルに変更  
		IncludeSuggestions:              true,
		IncludeEscapeReasons:           true,
		ConfidenceThreshold:            0.8,
		PotentialFalsePositiveDetection: true,
	}

	testCases := []struct {
		name         string
		diagnostic   analysis.Diagnostic
		confidence   float64
		shouldPass   bool
		description  string
	}{
		{
			name: "High confidence diagnostic passes",
			diagnostic: analysis.Diagnostic{
				Message: "Resource leak: missing defer Close()",
			},
			confidence:  0.9,
			shouldPass:  true,
			description: "高信頼度の診断は通す",
		},
		{
			name: "Low confidence diagnostic filtered",
			diagnostic: analysis.Diagnostic{
				Message: "Possible resource issue (uncertain)",
			},
			confidence:  0.6,
			shouldPass:  false,
			description: "低信頼度の診断はフィルタ",
		},
		{
			name: "Potential false positive marked",
			diagnostic: analysis.Diagnostic{
				Message: "Resource leak detected (potential false positive)",
			},
			confidence:  0.7,
			shouldPass:  false, // 偽陽性疑義によりフィルタ
			description: "偽陽性疑義のある診断はフィルタ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing: %s", tc.description)
			
			// IntegratedDiagnosticProcessor が未実装のため失敗を期待
			processor := NewIntegratedDiagnosticProcessor(config)
			result := processor.ProcessDiagnostic(tc.diagnostic, tc.confidence)
			
			if result.ShouldReport != tc.shouldPass {
				t.Errorf("Expected ShouldReport %v, got %v", 
					tc.shouldPass, result.ShouldReport)
			}
			
			t.Logf("Diagnostic processing result: ShouldReport=%v, Reason=%s", 
				result.ShouldReport, result.FilterReason)
		})
	}
}