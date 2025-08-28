package config

import (
	"golang.org/x/tools/go/analysis"
	"gopkg.in/yaml.v2"
	"strings"
)

// DiagnosticLevel は診断の重要度レベル
type DiagnosticLevel int

const (
	InfoLevel DiagnosticLevel = iota
	WarningLevel
	ErrorLevel
)

// String はレベルを文字列で返す
func (d DiagnosticLevel) String() string {
	switch d {
	case InfoLevel:
		return "info"
	case WarningLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	default:
		return "unknown"
	}
}

// DiagnosticConfig は診断制御の設定
type DiagnosticConfig struct {
	Level                           string         `yaml:"level"`
	IncludeSuggestions              bool           `yaml:"include_suggestions"`
	IncludeEscapeReasons            bool           `yaml:"include_escape_reasons"`
	ConfidenceThreshold             float64        `yaml:"confidence_threshold"`
	PotentialFalsePositiveDetection bool           `yaml:"potential_false_positive_detection"`
	CustomFilters                   []CustomFilter `yaml:"custom_filters"`
}

// CustomFilter はカスタムフィルタ設定
type CustomFilter struct {
	Pattern string `yaml:"pattern"`
	Action  string `yaml:"action"`
}

// DiagnosticFilter は診断フィルタリング機能
type DiagnosticFilter struct {
	level DiagnosticLevel
}

// NewDiagnosticFilter は新しいフィルターを作成
func NewDiagnosticFilter(level DiagnosticLevel) *DiagnosticFilter {
	return &DiagnosticFilter{level: level}
}

// ShouldIncludeDiagnostic は診断を含めるべきかを判定
func (f *DiagnosticFilter) ShouldIncludeDiagnostic(diag analysis.Diagnostic) bool {
	// メッセージ内容により重要度を判定
	message := strings.ToLower(diag.Message)

	// エラーレベルの判定
	if strings.Contains(message, "critical") || strings.Contains(message, "leak detected") {
		diagLevel := ErrorLevel
		return f.level <= diagLevel
	}

	// 警告レベルの判定（"potential false positive"を含む場合は除外）
	if strings.Contains(message, "potential") || strings.Contains(message, "possible") {
		if strings.Contains(message, "potential false positive") {
			return false // 偽陽性疑義のある診断は除外
		}
		diagLevel := WarningLevel
		return f.level <= diagLevel
	}

	// 情報レベル（デフォルト）
	diagLevel := InfoLevel
	return f.level <= diagLevel
}

// PotentialFalsePositiveDetector は偽陽性の疑いを検出
type PotentialFalsePositiveDetector struct {
	patterns []string
}

// NewPotentialFalsePositiveDetector は新しい検出器を作成
func NewPotentialFalsePositiveDetector() *PotentialFalsePositiveDetector {
	patterns := []string{
		"potential false positive",
		"uncertain",
		"unclear",
		"possible",
		"may be",
		"might be",
	}

	return &PotentialFalsePositiveDetector{patterns: patterns}
}

// IsPotentialFalsePositive は偽陽性の疑いがあるかを判定
func (d *PotentialFalsePositiveDetector) IsPotentialFalsePositive(message string) bool {
	lowerMessage := strings.ToLower(message)

	for _, pattern := range d.patterns {
		if strings.Contains(lowerMessage, pattern) {
			return true
		}
	}

	return false
}

// LoadDiagnosticConfigFromYAML はYAMLから診断設定を読み込む
func LoadDiagnosticConfigFromYAML(data []byte) (*DiagnosticConfig, error) {
	// 全体の設定構造体
	var fullConfig struct {
		Diagnostics DiagnosticConfig `yaml:"diagnostics"`
	}

	err := yaml.Unmarshal(data, &fullConfig)
	if err != nil {
		return nil, err
	}

	config := &fullConfig.Diagnostics

	// デフォルト値の設定（空の場合）
	if config.Level == "" {
		config.Level = "warning"
	}
	if config.ConfidenceThreshold == 0 {
		config.ConfidenceThreshold = 0.7
	}

	return config, nil
}

// DiagnosticProcessingResult は診断処理結果
type DiagnosticProcessingResult struct {
	ShouldReport    bool
	FilterReason    string
	ModifiedMessage string
	Confidence      float64
}

// IntegratedDiagnosticProcessor は統合診断処理
type IntegratedDiagnosticProcessor struct {
	config   *DiagnosticConfig
	filter   *DiagnosticFilter
	detector *PotentialFalsePositiveDetector
}

// NewIntegratedDiagnosticProcessor は新しいプロセッサを作成
func NewIntegratedDiagnosticProcessor(config *DiagnosticConfig) *IntegratedDiagnosticProcessor {
	// レベルを解析
	var level DiagnosticLevel
	switch strings.ToLower(config.Level) {
	case "error":
		level = ErrorLevel
	case "warning":
		level = WarningLevel
	case "info":
		level = InfoLevel
	default:
		level = WarningLevel
	}

	return &IntegratedDiagnosticProcessor{
		config:   config,
		filter:   NewDiagnosticFilter(level),
		detector: NewPotentialFalsePositiveDetector(),
	}
}

// ProcessDiagnostic は診断を処理
func (p *IntegratedDiagnosticProcessor) ProcessDiagnostic(diag analysis.Diagnostic, confidence float64) DiagnosticProcessingResult {
	result := DiagnosticProcessingResult{
		ShouldReport:    true,
		FilterReason:    "",
		ModifiedMessage: diag.Message,
		Confidence:      confidence,
	}

	// 信頼度チェック
	if confidence < p.config.ConfidenceThreshold {
		result.ShouldReport = false
		result.FilterReason = "Low confidence below threshold"
		return result
	}

	// レベルフィルタリング
	if !p.filter.ShouldIncludeDiagnostic(diag) {
		result.ShouldReport = false
		result.FilterReason = "Level filtered"
		return result
	}

	// 偽陽性検出
	if p.config.PotentialFalsePositiveDetection && p.detector.IsPotentialFalsePositive(diag.Message) {
		result.ShouldReport = false
		result.FilterReason = "Potential false positive detected"
		return result
	}

	return result
}
