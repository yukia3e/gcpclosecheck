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

	// Configuration Errors - used in config package for setup validation (lowercase for Go error convention)
	ConfigFileEmpty              = "configuration file path is empty"
	ConfigLoadFailed             = "failed to load configuration file: %w"
	ConfigYAMLParseFailed        = "failed to parse YAML configuration: %w"
	DefaultConfigLoadFailed      = "failed to load default configuration file: %w"
	DefaultConfigYAMLParseFailed = "failed to parse default YAML configuration: %w"

	// Validation Errors - used for data structure validation (lowercase for Go error convention)
	ServicesListEmpty            = "services definition is empty"
	ServiceNameEmpty             = "service[%d]: service name is empty"
	ServicePackagePathEmpty      = "service[%d](%s): package path is empty"
	ServiceCreationFuncsEmpty    = "service[%d](%s): creation functions not defined"
	ServiceCleanupMethodsEmpty   = "service[%d](%s): cleanup methods not defined"
	CleanupMethodNameEmpty       = "service[%d](%s): cleanup method[%d] method name is empty"
	PackageExceptionNameEmpty    = "package exception[%d]: exception name is empty"
	PackageExceptionPatternEmpty = "package exception[%d](%s): pattern is empty"
	InvalidExceptionType         = "package exception[%d](%s): invalid condition type: %s (valid types: %v)"

	// Type Validation Errors - used in analyzer/types.go (lowercase for Go error convention)
	VariableCannotBeNil          = "variable cannot be nil"
	ServiceTypeCannotBeEmpty     = "serviceType cannot be empty"
	CleanupMethodCannotBeEmpty   = "cleanupMethod cannot be empty"
	CancelFuncCannotBeNil        = "cancelFunc cannot be nil"
	CancelVarNameCannotBeEmpty   = "cancelVarName cannot be empty"
	DeferPosInvalid              = "deferPos is invalid"
	TransactionTypeMustBeValid   = "TransactionType must be ReadWriteTransaction or ReadOnlyTransaction"
	AutoManagementReasonRequired = "autoManagementReason cannot be empty for auto-managed transactions"

	// Help Messages - used in CLI interface
	ToolDescription      = "Detects missing Close/Stop/Cancel calls for GCP resource clients."
	UsageExamples        = "Usage Examples"
	RecommendedPractices = "Best Practices"

	// Suggested Fix Messages - used for automated fix suggestions
	AddDeferStatement  = "Add defer %s()"
	AddDeferMethodCall = "Add defer %s.%s() for client cleanup"
)
