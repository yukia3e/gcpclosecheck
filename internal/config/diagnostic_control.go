package config

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"gopkg.in/yaml.v2"
)

// DiagnosticLevel represents the importance level of diagnostics
type DiagnosticLevel int

const (
	InfoLevel DiagnosticLevel = iota
	WarningLevel
	ErrorLevel
)

// String returns the level as a string
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

// DiagnosticConfig represents diagnostic control settings
type DiagnosticConfig struct {
	Level                           string         `yaml:"level"`
	IncludeSuggestions              bool           `yaml:"include_suggestions"`
	IncludeEscapeReasons            bool           `yaml:"include_escape_reasons"`
	ConfidenceThreshold             float64        `yaml:"confidence_threshold"`
	PotentialFalsePositiveDetection bool           `yaml:"potential_false_positive_detection"`
	CustomFilters                   []CustomFilter `yaml:"custom_filters"`
}

// CustomFilter represents custom filter settings
type CustomFilter struct {
	Pattern string `yaml:"pattern"`
	Action  string `yaml:"action"`
}

// DiagnosticFilter provides diagnostic filtering functionality
type DiagnosticFilter struct {
	level DiagnosticLevel
}

// NewDiagnosticFilter creates a new filter
func NewDiagnosticFilter(level DiagnosticLevel) *DiagnosticFilter {
	return &DiagnosticFilter{level: level}
}

// ShouldIncludeDiagnostic determines if a diagnostic should be included
func (f *DiagnosticFilter) ShouldIncludeDiagnostic(diag analysis.Diagnostic) bool {
	// Determine importance level based on message content
	message := strings.ToLower(diag.Message)

	// Error level determination
	if strings.Contains(message, "critical") || strings.Contains(message, "leak detected") {
		diagLevel := ErrorLevel
		return f.level <= diagLevel
	}

	// Warning level determination (exclude if contains "potential false positive")
	if strings.Contains(message, "potential") || strings.Contains(message, "possible") {
		if strings.Contains(message, "potential false positive") {
			return false // Exclude diagnostics with false positive suspicion
		}
		diagLevel := WarningLevel
		return f.level <= diagLevel
	}

	// Info level (default)
	diagLevel := InfoLevel
	return f.level <= diagLevel
}

// PotentialFalsePositiveDetector detects potential false positives
type PotentialFalsePositiveDetector struct {
	patterns []string
}

// NewPotentialFalsePositiveDetector creates a new detector
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

// IsPotentialFalsePositive determines if there is suspicion of false positive
func (d *PotentialFalsePositiveDetector) IsPotentialFalsePositive(message string) bool {
	lowerMessage := strings.ToLower(message)

	for _, pattern := range d.patterns {
		if strings.Contains(lowerMessage, pattern) {
			return true
		}
	}

	return false
}

// LoadDiagnosticConfigFromYAML loads diagnostic configuration from YAML
func LoadDiagnosticConfigFromYAML(data []byte) (*DiagnosticConfig, error) {
	// Overall configuration structure
	var fullConfig struct {
		Diagnostics DiagnosticConfig `yaml:"diagnostics"`
	}

	err := yaml.Unmarshal(data, &fullConfig)
	if err != nil {
		return nil, err
	}

	config := &fullConfig.Diagnostics

	// Set default values (if empty)
	if config.Level == "" {
		config.Level = "warning"
	}
	if config.ConfidenceThreshold == 0 {
		config.ConfidenceThreshold = 0.7
	}

	return config, nil
}

// DiagnosticProcessingResult represents diagnostic processing result
type DiagnosticProcessingResult struct {
	ShouldReport    bool
	FilterReason    string
	ModifiedMessage string
	Confidence      float64
}

// IntegratedDiagnosticProcessor provides integrated diagnostic processing
type IntegratedDiagnosticProcessor struct {
	config   *DiagnosticConfig
	filter   *DiagnosticFilter
	detector *PotentialFalsePositiveDetector
}

// NewIntegratedDiagnosticProcessor creates a new processor
func NewIntegratedDiagnosticProcessor(config *DiagnosticConfig) *IntegratedDiagnosticProcessor {
	// Parse level
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

// ProcessDiagnostic processes a diagnostic
func (p *IntegratedDiagnosticProcessor) ProcessDiagnostic(diag analysis.Diagnostic, confidence float64) DiagnosticProcessingResult {
	result := DiagnosticProcessingResult{
		ShouldReport:    true,
		FilterReason:    "",
		ModifiedMessage: diag.Message,
		Confidence:      confidence,
	}

	// Confidence check
	if confidence < p.config.ConfidenceThreshold {
		result.ShouldReport = false
		result.FilterReason = "Low confidence below threshold"
		return result
	}

	// Level filtering
	if !p.filter.ShouldIncludeDiagnostic(diag) {
		result.ShouldReport = false
		result.FilterReason = "Level filtered"
		return result
	}

	// False positive detection
	if p.config.PotentialFalsePositiveDetection && p.detector.IsPotentialFalsePositive(diag.Message) {
		result.ShouldReport = false
		result.FilterReason = "Potential false positive detected"
		return result
	}

	return result
}
