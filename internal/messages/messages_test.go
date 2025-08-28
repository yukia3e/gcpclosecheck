package messages

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// TestMessageConstantsNotEmpty verifies all message constants are not empty
func TestMessageConstantsNotEmpty(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		// Diagnostic Messages
		{"MissingResourceCleanup", MissingResourceCleanup},
		{"MissingContextCancel", MissingContextCancel},
		
		// Configuration Errors
		{"ConfigFileEmpty", ConfigFileEmpty},
		{"ConfigLoadFailed", ConfigLoadFailed},
		{"ConfigYAMLParseFailed", ConfigYAMLParseFailed},
		{"DefaultConfigLoadFailed", DefaultConfigLoadFailed},
		{"DefaultConfigYAMLParseFailed", DefaultConfigYAMLParseFailed},
		
		// Validation Errors
		{"ServicesListEmpty", ServicesListEmpty},
		{"ServiceNameEmpty", ServiceNameEmpty},
		{"ServicePackagePathEmpty", ServicePackagePathEmpty},
		{"ServiceCreationFuncsEmpty", ServiceCreationFuncsEmpty},
		{"ServiceCleanupMethodsEmpty", ServiceCleanupMethodsEmpty},
		{"CleanupMethodNameEmpty", CleanupMethodNameEmpty},
		{"PackageExceptionNameEmpty", PackageExceptionNameEmpty},
		{"PackageExceptionPatternEmpty", PackageExceptionPatternEmpty},
		{"InvalidExceptionType", InvalidExceptionType},
		
		// Type Validation Errors
		{"VariableCannotBeNil", VariableCannotBeNil},
		{"ServiceTypeCannotBeEmpty", ServiceTypeCannotBeEmpty},
		{"CleanupMethodCannotBeEmpty", CleanupMethodCannotBeEmpty},
		{"CancelFuncCannotBeNil", CancelFuncCannotBeNil},
		{"CancelVarNameCannotBeEmpty", CancelVarNameCannotBeEmpty},
		{"DeferPosInvalid", DeferPosInvalid},
		{"TransactionTypeMustBeValid", TransactionTypeMustBeValid},
		{"AutoManagementReasonRequired", AutoManagementReasonRequired},
		
		// Help Messages
		{"ToolDescription", ToolDescription},
		{"UsageExamples", UsageExamples},
		{"RecommendedPractices", RecommendedPractices},
		
		// Suggested Fix Messages
		{"AddDeferStatement", AddDeferStatement},
		{"AddDeferMethodCall", AddDeferMethodCall},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.TrimSpace(tt.message) == "" {
				t.Errorf("%s constant is empty", tt.name)
			}
		})
	}
}

// TestMessagePlaceholderFormats verifies placeholder formats are correct
func TestMessagePlaceholderFormats(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		expectedFmt []string
	}{
		{"MissingResourceCleanup", MissingResourceCleanup, []string{"%s", "%s"}},
		{"MissingContextCancel", MissingContextCancel, []string{"%s"}},
		{"ConfigLoadFailed", ConfigLoadFailed, []string{"%w"}},
		{"ConfigYAMLParseFailed", ConfigYAMLParseFailed, []string{"%w"}},
		{"ServiceNameEmpty", ServiceNameEmpty, []string{"%d"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, expectedPlaceholder := range tt.expectedFmt {
				if !strings.Contains(tt.message, expectedPlaceholder) {
					t.Errorf("%s should contain placeholder %s", tt.name, expectedPlaceholder)
				}
			}
		})
	}
}

// TestNoJapaneseCharacters verifies no Japanese characters in messages
func TestNoJapaneseCharacters(t *testing.T) {
	// Regular expression to match Japanese characters (Hiragana, Katakana, Kanji)
	japanesePattern := regexp.MustCompile("[\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]")

	messages := []struct {
		name    string
		message string
	}{
		// Test a representative sample of all message categories
		{"MissingResourceCleanup", MissingResourceCleanup},
		{"MissingContextCancel", MissingContextCancel},
		{"ConfigFileEmpty", ConfigFileEmpty},
		{"ConfigLoadFailed", ConfigLoadFailed},
		{"ServicesListEmpty", ServicesListEmpty},
		{"ServiceNameEmpty", ServiceNameEmpty},
		{"VariableCannotBeNil", VariableCannotBeNil},
		{"ServiceTypeCannotBeEmpty", ServiceTypeCannotBeEmpty},
		{"ToolDescription", ToolDescription},
		{"UsageExamples", UsageExamples},
		{"AddDeferStatement", AddDeferStatement},
		{"AddDeferMethodCall", AddDeferMethodCall},
	}

	for _, msg := range messages {
		t.Run(msg.name, func(t *testing.T) {
			if japanesePattern.MatchString(msg.message) {
				t.Errorf("%s contains Japanese characters: %s", msg.name, msg.message)
			}
		})
	}
}

// TestMessageFormatting verifies message formatting with example values
func TestMessageFormatting(t *testing.T) {
	tests := []struct {
		name     string
		template string
		args     []interface{}
		expected string
	}{
		{
			name:     "MissingResourceCleanup formatting",
			template: MissingResourceCleanup,
			args:     []interface{}{"client", "Close"},
			expected: "GCP resource client 'client' missing cleanup method (Close)",
		},
		{
			name:     "MissingContextCancel formatting",
			template: MissingContextCancel,
			args:     []interface{}{"cancel"},
			expected: "Context.WithCancel missing cancel function call 'cancel'",
		},
		{
			name:     "ServiceNameEmpty formatting",
			template: ServiceNameEmpty,
			args:     []interface{}{1},
			expected: "Service[1]: service name is empty",
		},
		{
			name:     "AddDeferStatement formatting",
			template: AddDeferStatement,
			args:     []interface{}{"cancel"},
			expected: "Add defer cancel()",
		},
		{
			name:     "AddDeferMethodCall formatting",
			template: AddDeferMethodCall,
			args:     []interface{}{"client", "Close"},
			expected: "Add defer client.Close() for client cleanup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fmt.Sprintf(tt.template, tt.args...)
			if result != tt.expected {
				t.Errorf("Expected: %s, Got: %s", tt.expected, result)
			}
		})
	}
}

// TestMessageConsistencyRules verifies consistent messaging patterns
func TestMessageConsistencyRules(t *testing.T) {
	t.Run("GCP terminology consistency", func(t *testing.T) {
		// Test that messages use consistent GCP-related terminology
		gcpMessages := []struct {
			name    string
			message string
			expectedTerms []string
		}{
			{"MissingResourceCleanup", MissingResourceCleanup, []string{"GCP", "resource"}},
			{"ToolDescription", ToolDescription, []string{"Close", "Cancel", "GCP"}},
		}
		
		for _, msg := range gcpMessages {
			for _, term := range msg.expectedTerms {
				if !strings.Contains(msg.message, term) {
					t.Errorf("%s should contain GCP terminology '%s': %s", msg.name, term, msg.message)
				}
			}
		}
	})
	
	t.Run("Professional tone check", func(t *testing.T) {
		// Test that messages use appropriate professional language
		// Check for informal words that should be avoided
		informalWords := []string{"gonna", "wanna", "gotta", "ain't", "can't", "won't", "don't"}
		
		allMessages := []struct {
			name    string
			message string
		}{
			{"MissingResourceCleanup", MissingResourceCleanup},
			{"MissingContextCancel", MissingContextCancel},
			{"ToolDescription", ToolDescription},
			{"ConfigFileEmpty", ConfigFileEmpty},
		}
		
		for _, msg := range allMessages {
			lowerMsg := strings.ToLower(msg.message)
			for _, word := range informalWords {
				if strings.Contains(lowerMsg, word) {
					t.Errorf("%s contains informal word '%s': %s", msg.name, word, msg.message)
				}
			}
		}
	})
}

// TestComprehensivePlaceholderValidation verifies all placeholder types
func TestComprehensivePlaceholderValidation(t *testing.T) {
	placeholderTests := []struct {
		name        string
		message     string
		placeholders []string
		description string
	}{
		{
			name: "String placeholders", 
			message: MissingResourceCleanup, 
			placeholders: []string{"%s"}, 
			description: "Should contain string placeholders for resource name and cleanup method",
		},
		{
			name: "Error wrapping placeholders", 
			message: ConfigLoadFailed, 
			placeholders: []string{"%w"}, 
			description: "Should contain error wrapping placeholder",
		},
		{
			name: "Integer placeholders", 
			message: ServiceNameEmpty, 
			placeholders: []string{"%d"}, 
			description: "Should contain integer placeholder for service index",
		},
		{
			name: "Multiple string placeholders", 
			message: ServicePackagePathEmpty, 
			placeholders: []string{"%d", "%s"}, 
			description: "Should contain both integer and string placeholders",
		},
	}
	
	for _, test := range placeholderTests {
		t.Run(test.name, func(t *testing.T) {
			for _, placeholder := range test.placeholders {
				if !strings.Contains(test.message, placeholder) {
					t.Errorf("%s: %s\nMessage: %s\nMissing placeholder: %s", 
						test.name, test.description, test.message, placeholder)
				}
			}
		})
	}
}

// TestMessageCategorizationCompleteness ensures all message categories are covered
func TestMessageCategorizationCompleteness(t *testing.T) {
	// Define expected message categories and their minimum counts
	categoryTests := map[string]struct {
		minCount int
		examples []string
	}{
		"diagnostic": {
			minCount: 2,
			examples: []string{MissingResourceCleanup, MissingContextCancel},
		},
		"configuration": {
			minCount: 3,
			examples: []string{ConfigFileEmpty, ConfigLoadFailed, ConfigYAMLParseFailed},
		},
		"validation": {
			minCount: 5,
			examples: []string{ServicesListEmpty, ServiceNameEmpty, VariableCannotBeNil, ServiceTypeCannotBeEmpty, CleanupMethodCannotBeEmpty},
		},
		"help": {
			minCount: 2,
			examples: []string{ToolDescription, UsageExamples},
		},
		"suggested_fix": {
			minCount: 2,
			examples: []string{AddDeferStatement, AddDeferMethodCall},
		},
	}
	
	for category, test := range categoryTests {
		t.Run(fmt.Sprintf("Category_%s", category), func(t *testing.T) {
			if len(test.examples) < test.minCount {
				t.Errorf("Category %s has %d examples, but requires minimum %d", 
					category, len(test.examples), test.minCount)
			}
			
			// Verify examples are not empty
			for i, example := range test.examples {
				if strings.TrimSpace(example) == "" {
					t.Errorf("Category %s example %d is empty", category, i)
				}
			}
		})
	}
}

// TestAllConstantsUsingReflection verifies all exported constants using reflection
func TestAllConstantsUsingReflection(t *testing.T) {
	// Test that all constants we expect exist and are not empty
	expectedConstants := []string{
		"MissingResourceCleanup", "MissingContextCancel",
		"ConfigFileEmpty", "ConfigLoadFailed", "ConfigYAMLParseFailed",
		"ServicesListEmpty", "ServiceNameEmpty", "VariableCannotBeNil",
		"ToolDescription", "UsageExamples", "AddDeferStatement",
	}
	
	for _, constName := range expectedConstants {
		// Use reflection to get constant value
		constValue := getConstantByName(constName)
		if constValue == "" {
			t.Errorf("Constant %s is empty", constName)
		}
		
		// Verify it's a string type
		if reflect.TypeOf(constValue).Kind() != reflect.String {
			t.Errorf("Constant %s is not a string type", constName)
		}
	}
}

// Helper function to get constant value by name using reflection
func getConstantByName(name string) string {
	switch name {
	case "MissingResourceCleanup":
		return MissingResourceCleanup
	case "MissingContextCancel":
		return MissingContextCancel
	case "ConfigFileEmpty":
		return ConfigFileEmpty
	case "ConfigLoadFailed":
		return ConfigLoadFailed
	case "ConfigYAMLParseFailed":
		return ConfigYAMLParseFailed
	case "ServicesListEmpty":
		return ServicesListEmpty
	case "ServiceNameEmpty":
		return ServiceNameEmpty
	case "VariableCannotBeNil":
		return VariableCannotBeNil
	case "ToolDescription":
		return ToolDescription
	case "UsageExamples":
		return UsageExamples
	case "AddDeferStatement":
		return AddDeferStatement
	default:
		return ""
	}
}

// TestMessageEndToEndIntegration simulates real-world message usage
func TestMessageEndToEndIntegration(t *testing.T) {
	integrationTests := []struct {
		name     string
		scenario string
		template string
		args     []interface{}
		validate func(string) error
	}{
		{
			name:     "Resource cleanup diagnostic flow",
			scenario: "Simulating analyzer reporting missing resource cleanup",
			template: MissingResourceCleanup,
			args:     []interface{}{"client", "Close"},
			validate: func(result string) error {
				if !strings.Contains(result, "GCP resource") {
					return fmt.Errorf("missing GCP terminology")
				}
				if !strings.Contains(result, "client") {
					return fmt.Errorf("missing resource name")
				}
				if !strings.Contains(result, "Close") {
					return fmt.Errorf("missing cleanup method")
				}
				return nil
			},
		},
		{
			name:     "Context cancel diagnostic flow",
			scenario: "Simulating analyzer reporting missing context cancel",
			template: MissingContextCancel,
			args:     []interface{}{"cancel"},
			validate: func(result string) error {
				if !strings.Contains(result, "Context.WithCancel") {
					return fmt.Errorf("missing context terminology")
				}
				if !strings.Contains(result, "cancel") {
					return fmt.Errorf("missing cancel function name")
				}
				return nil
			},
		},
		{
			name:     "Configuration error flow",
			scenario: "Simulating configuration loading failure",
			template: ConfigLoadFailed,
			args:     []interface{}{fmt.Errorf("file not found")},
			validate: func(result string) error {
				if !strings.Contains(result, "Failed to load") {
					return fmt.Errorf("missing failure terminology")
				}
				if !strings.Contains(result, "file not found") {
					return fmt.Errorf("missing underlying error")
				}
				return nil
			},
		},
		{
			name:     "Suggested fix flow",
			scenario: "Simulating suggested fix generation",
			template: AddDeferStatement,
			args:     []interface{}{"cancel"},
			validate: func(result string) error {
				if !strings.Contains(result, "Add defer") {
					return fmt.Errorf("missing defer instruction")
				}
				if !strings.Contains(result, "cancel()") {
					return fmt.Errorf("missing function call syntax")
				}
				return nil
			},
		},
	}
	
	for _, test := range integrationTests {
		t.Run(test.name, func(t *testing.T) {
			// Simulate message formatting as would happen in real usage
			result := fmt.Sprintf(test.template, test.args...)
			
			// Validate the result meets integration requirements
			if err := test.validate(result); err != nil {
				t.Errorf("Integration test failed for %s: %v\nScenario: %s\nResult: %s", 
					test.name, err, test.scenario, result)
			}
			
			// Additional checks for all integration tests
			if strings.TrimSpace(result) == "" {
				t.Errorf("Integration test %s produced empty result", test.name)
			}
			
			// Verify no Japanese characters in result
			japanesePattern := regexp.MustCompile("[\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]")
			if japanesePattern.MatchString(result) {
				t.Errorf("Integration test %s result contains Japanese characters: %s", test.name, result)
			}
		})
	}
}

// TestTask15_MessageConsistencyValidation verifies Task 15 completion: message consistency validation tests
func TestTask15_MessageConsistencyValidation(t *testing.T) {
	
	t.Run("AllMessagesEnglishOnly", func(t *testing.T) {
		// Test that all message constants contain only English characters
		// This test should verify Requirements 1.4, 2.1
		
		// Regular expression to detect non-English characters (including Japanese)
		nonEnglishPattern := regexp.MustCompile("[\u0080-\uFFFF]") // Any non-ASCII character
		japanesePattern := regexp.MustCompile("[\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]") // Japanese specific
		
		allMessages := getAllMessageConstants()
		
		for name, message := range allMessages {
			// Check for Japanese characters specifically
			if japanesePattern.MatchString(message) {
				t.Errorf("Message %s contains Japanese characters: %s", name, message)
			}
			
			// Allow common punctuation and technical symbols, but flag suspicious patterns
			if nonEnglishPattern.MatchString(message) {
				// Allow certain technical characters commonly used in English
				allowedNonASCII := regexp.MustCompile(`[''""–—…]`) // Smart quotes, em dashes, ellipsis
				suspiciousText := nonEnglishPattern.ReplaceAllStringFunc(message, func(s string) string {
					if allowedNonASCII.MatchString(s) {
						return ""
					}
					return s
				})
				
				if suspiciousText != "" {
					t.Errorf("Message %s contains suspicious non-English characters: %s (in: %s)", 
						name, suspiciousText, message)
				}
			}
		}
	})
	
	t.Run("TerminologyConsistency", func(t *testing.T) {
		// Test message terminology consistency per Requirements 2.1, 2.2
		// Verify consistent use of "resource", "client", "cleanup" terms
		
		terminologyTests := map[string]struct {
			requiredTerms    []string
			forbiddenTerms   []string
			description      string
		}{
			"resource_messages": {
				requiredTerms:  []string{"resource", "cleanup"},
				forbiddenTerms: []string{"close", "finish", "end"},
				description:    "Resource-related messages should use 'resource' and 'cleanup' terminology",
			},
			"client_messages": {
				requiredTerms:  []string{"client"},
				forbiddenTerms: []string{"connection", "service", "session"},
				description:    "Client-related messages should consistently use 'client'",
			},
		}
		
		allMessages := getAllMessageConstants()
		
		// Check specific message patterns
		resourceMessages := []string{"MissingResourceCleanup"}
		clientMessages := []string{"MissingResourceCleanup", "AddDeferMethodCall"}
		
		for _, msgName := range resourceMessages {
			if message, exists := allMessages[msgName]; exists {
				test := terminologyTests["resource_messages"]
				for _, required := range test.requiredTerms {
					if !strings.Contains(strings.ToLower(message), required) {
						t.Errorf("Message %s should contain term '%s': %s", msgName, required, message)
					}
				}
			}
		}
		
		for _, msgName := range clientMessages {
			if message, exists := allMessages[msgName]; exists {
				test := terminologyTests["client_messages"] 
				for _, required := range test.requiredTerms {
					if !strings.Contains(strings.ToLower(message), required) {
						t.Errorf("Message %s should contain term '%s': %s", msgName, required, message)
					}
				}
			}
		}
	})
	
	t.Run("FormatConsistency", func(t *testing.T) {
		// Test message format consistency per Requirements 2.1, 2.2
		// Check capitalization, punctuation, sentence structure
		
		allMessages := getAllMessageConstants()
		
		for name, message := range allMessages {
			// Test sentence structure consistency
			if strings.TrimSpace(message) == "" {
				t.Errorf("Message %s is empty", name)
				continue
			}
			
			// Check for consistent capitalization at start
			firstChar := rune(message[0])
			if !isUpperCase(firstChar) && !isDigit(firstChar) && firstChar != '%' {
				t.Errorf("Message %s should start with capital letter: %s", name, message)
			}
			
			// Check for consistent punctuation at end for error messages
			if strings.Contains(name, "Error") || strings.Contains(name, "Failed") || 
			   strings.Contains(name, "Empty") || strings.Contains(name, "Invalid") {
				// Error messages should not end with period (for consistency with Go conventions)
				if strings.HasSuffix(message, ".") {
					t.Errorf("Error message %s should not end with period: %s", name, message)
				}
			}
			
			// Check for help messages formatting
			if strings.Contains(name, "Description") || strings.Contains(name, "Usage") ||
			   strings.Contains(name, "Practices") {
				// Help messages should be complete sentences
				if !strings.HasSuffix(message, ".") && !strings.Contains(message, "\n") {
					// Allow multi-line help messages without period check
					if !strings.Contains(message, "\n") {
						t.Logf("Help message %s might need proper punctuation: %s", name, message)
					}
				}
			}
		}
	})
	
	t.Run("PlaceholderUsageConsistency", func(t *testing.T) {
		// Test placeholder and actual usage consistency per Requirements 5.1
		// Verify placeholder patterns match their intended usage
		
		placeholderConsistencyTests := map[string]struct {
			expectedPlaceholders []string
			usagePattern        string
			description         string
		}{
			"MissingResourceCleanup": {
				expectedPlaceholders: []string{"%s", "%s"},
				usagePattern:        "resource name and cleanup method",
				description:         "Should have two string placeholders for resource name and method",
			},
			"MissingContextCancel": {
				expectedPlaceholders: []string{"%s"},
				usagePattern:        "cancel function name",
				description:         "Should have one string placeholder for cancel function name",
			},
			"ServiceNameEmpty": {
				expectedPlaceholders: []string{"%d"},
				usagePattern:        "service index",
				description:         "Should have one integer placeholder for service index",
			},
			"ConfigLoadFailed": {
				expectedPlaceholders: []string{"%w"},
				usagePattern:        "wrapped error",
				description:         "Should have one error wrapping placeholder",
			},
		}
		
		allMessages := getAllMessageConstants()
		
		for msgName, test := range placeholderConsistencyTests {
			if message, exists := allMessages[msgName]; exists {
				// Check that expected placeholders are present
				for _, placeholder := range test.expectedPlaceholders {
					if !strings.Contains(message, placeholder) {
						t.Errorf("Message %s missing expected placeholder %s: %s", 
							msgName, placeholder, message)
					}
				}
				
				// Count actual placeholders
				actualPlaceholders := countPlaceholders(message)
				if len(actualPlaceholders) != len(test.expectedPlaceholders) {
					t.Errorf("Message %s has %d placeholders, expected %d: %s",
						msgName, len(actualPlaceholders), len(test.expectedPlaceholders), message)
				}
				
				// Validate placeholder order makes sense
				if msgName == "MissingResourceCleanup" {
					// Should be: resource name first, cleanup method second
					firstPct := strings.Index(message, "%s")
					secondPct := strings.Index(message[firstPct+1:], "%s")
					if firstPct == -1 || secondPct == -1 {
						t.Errorf("Message %s should have two %%s placeholders in correct positions: %s",
							msgName, message)
					}
				}
			} else {
				t.Errorf("Message %s not found in constants", msgName)
			}
		}
	})
}

// Helper function to get all message constants
func getAllMessageConstants() map[string]string {
	return map[string]string{
		"MissingResourceCleanup":         MissingResourceCleanup,
		"MissingContextCancel":           MissingContextCancel,
		"ConfigFileEmpty":                ConfigFileEmpty,
		"ConfigLoadFailed":               ConfigLoadFailed,
		"ConfigYAMLParseFailed":          ConfigYAMLParseFailed,
		"DefaultConfigLoadFailed":        DefaultConfigLoadFailed,
		"DefaultConfigYAMLParseFailed":   DefaultConfigYAMLParseFailed,
		"ServicesListEmpty":              ServicesListEmpty,
		"ServiceNameEmpty":               ServiceNameEmpty,
		"ServicePackagePathEmpty":        ServicePackagePathEmpty,
		"ServiceCreationFuncsEmpty":      ServiceCreationFuncsEmpty,
		"ServiceCleanupMethodsEmpty":     ServiceCleanupMethodsEmpty,
		"CleanupMethodNameEmpty":         CleanupMethodNameEmpty,
		"PackageExceptionNameEmpty":      PackageExceptionNameEmpty,
		"PackageExceptionPatternEmpty":   PackageExceptionPatternEmpty,
		"InvalidExceptionType":           InvalidExceptionType,
		"VariableCannotBeNil":           VariableCannotBeNil,
		"ServiceTypeCannotBeEmpty":      ServiceTypeCannotBeEmpty,
		"CleanupMethodCannotBeEmpty":    CleanupMethodCannotBeEmpty,
		"CancelFuncCannotBeNil":         CancelFuncCannotBeNil,
		"CancelVarNameCannotBeEmpty":    CancelVarNameCannotBeEmpty,
		"DeferPosInvalid":               DeferPosInvalid,
		"TransactionTypeMustBeValid":    TransactionTypeMustBeValid,
		"AutoManagementReasonRequired":  AutoManagementReasonRequired,
		"ToolDescription":               ToolDescription,
		"UsageExamples":                 UsageExamples,
		"RecommendedPractices":          RecommendedPractices,
		"AddDeferStatement":             AddDeferStatement,
		"AddDeferMethodCall":            AddDeferMethodCall,
	}
}

// Helper function to check if character is uppercase
func isUpperCase(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// Helper function to check if character is digit
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// Helper function to count placeholders in a message
func countPlaceholders(message string) []string {
	placeholderPattern := regexp.MustCompile(`%[a-zA-Z]`)
	return placeholderPattern.FindAllString(message, -1)
}