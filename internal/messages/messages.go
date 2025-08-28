// Package messages provides English message constants for gcpclosecheck linter.
// This package centralizes all user-facing messages to ensure consistency
// and maintainability across the entire application.
//
// Message categories:
//   - Diagnostic Messages: Used for reporting analysis results
//   - Configuration Errors: Used for configuration validation
//   - Validation Errors: Used for data structure validation
//   - Help Messages: Used in CLI interface
//   - Suggested Fix Messages: Used for automated fix suggestions
package messages

const (
	// Diagnostic Messages - used in analyzer package for issue reporting
	MissingResourceCleanup = "GCP resource client '%s' missing cleanup method (%s)"
	MissingContextCancel   = "Context.WithCancel missing cancel function call '%s'"

	// Configuration Errors - used in config package for setup validation
	ConfigFileEmpty        = "Configuration file path is empty"
	ConfigLoadFailed       = "Failed to load configuration file: %w"
	ConfigYAMLParseFailed  = "Failed to parse YAML configuration: %w"
	DefaultConfigLoadFailed = "Failed to load default configuration file: %w"
	DefaultConfigYAMLParseFailed = "Failed to parse default YAML configuration: %w"

	// Validation Errors - used for data structure validation
	ServicesListEmpty      = "Services definition is empty"
	ServiceNameEmpty       = "Service[%d]: service name is empty"
	ServicePackagePathEmpty = "Service[%d](%s): package path is empty"
	ServiceCreationFuncsEmpty = "Service[%d](%s): creation functions not defined"
	ServiceCleanupMethodsEmpty = "Service[%d](%s): cleanup methods not defined"
	CleanupMethodNameEmpty = "Service[%d](%s): cleanup method[%d] method name is empty"
	PackageExceptionNameEmpty = "Package exception[%d]: exception name is empty"
	PackageExceptionPatternEmpty = "Package exception[%d](%s): pattern is empty"
	InvalidExceptionType = "Package exception[%d](%s): invalid condition type: %s (valid types: %v)"

	// Type Validation Errors - used in analyzer/types.go
	VariableCannotBeNil    = "Variable cannot be nil"
	ServiceTypeCannotBeEmpty = "ServiceType cannot be empty"
	CleanupMethodCannotBeEmpty = "CleanupMethod cannot be empty"
	CancelFuncCannotBeNil = "CancelFunc cannot be nil"
	CancelVarNameCannotBeEmpty = "CancelVarName cannot be empty"
	DeferPosInvalid = "DeferPos is invalid"
	TransactionTypeMustBeValid = "TransactionType must be ReadWriteTransaction or ReadOnlyTransaction"
	AutoManagementReasonRequired = "AutoManagementReason cannot be empty for auto-managed transactions"

	// Help Messages - used in CLI interface
	ToolDescription      = "Detects missing Close/Stop/Cancel calls for GCP resource clients."
	UsageExamples        = "Usage Examples"
	RecommendedPractices = "Best Practices"

	// Suggested Fix Messages - used for automated fix suggestions
	AddDeferStatement    = "Add defer %s()"
	AddDeferMethodCall   = "Add defer %s.%s() for client cleanup"
)