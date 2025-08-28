package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/yukia3e/gcpclosecheck/internal/messages"
)

func TestDiagnosticGenerator_ReportMissingDefer(t *testing.T) {
	// Test sample code
	testCode := `
package test

import (
	"context"
	"cloud.google.com/go/spanner"
)

func testFunction(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	// defer client.Close() is missing
	return nil
}
`

	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "test.go", testCode, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test code: %v", err)
	}

	generator := NewDiagnosticGenerator(fset)

	// Create ResourceInfo (for testing)
	resource := ResourceInfo{
		Variable:         types.NewVar(token.Pos(100), nil, "client", nil),
		CreationPos:      token.Pos(100),
		ServiceType:      "spanner",
		CreationFunction: "NewClient",
		CleanupMethod:    "Close",
		IsRequired:       true,
	}

	diagnostic := generator.ReportMissingDefer(resource)

	// Verify diagnostic content
	if diagnostic.Category != "resource-leak" {
		t.Errorf("Expected category 'resource-leak', got %q", diagnostic.Category)
	}

	// Expect English message format (updated for Task 15)
	expectedMessage := "GCP resource client 'client' missing cleanup method (Close)"
	if diagnostic.Message != expectedMessage {
		t.Errorf("Expected message %q, got %q", expectedMessage, diagnostic.Message)
	}

	// Verify SuggestedFix existence
	if len(diagnostic.SuggestedFixes) == 0 {
		t.Error("Expected at least one SuggestedFix")
	}

	// Expect English suggested fix message (updated for Task 15)
	fix := diagnostic.SuggestedFixes[0]
	expectedFixMessage := "Add defer client.Close() for client cleanup"
	if fix.Message != expectedFixMessage {
		t.Errorf("Expected SuggestedFix message %q, got %q", expectedFixMessage, fix.Message)
	}
}

func TestDiagnosticGenerator_ReportMissingContextCancel(t *testing.T) {
	testCode := `
package test

import "context"

func testFunction(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	// defer cancel() is missing
	return nil
}
`

	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "test.go", testCode, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test code: %v", err)
	}

	generator := NewDiagnosticGenerator(fset)

	// Create ContextInfo (for testing)
	contextInfo := ContextInfo{
		Variable:    types.NewVar(token.Pos(80), nil, "ctx", nil),
		CancelFunc:  types.NewVar(token.Pos(90), nil, "cancel", nil),
		CreationPos: token.Pos(80),
		IsDeferred:  false,
	}

	diagnostic := generator.ReportMissingContextCancel(contextInfo)

	// Verify diagnostic content
	if diagnostic.Category != "context-leak" {
		t.Errorf("Expected category 'context-leak', got %q", diagnostic.Category)
	}

	// Expect English message format (updated for Task 15)
	expectedMessage := "Context.WithCancel missing cancel function call 'cancel'"
	if diagnostic.Message != expectedMessage {
		t.Errorf("Expected message %q, got %q", expectedMessage, diagnostic.Message)
	}

	// Verify SuggestedFix existence
	if len(diagnostic.SuggestedFixes) == 0 {
		t.Error("Expected at least one SuggestedFix")
	}

	// Expect English suggested fix message
	fix := diagnostic.SuggestedFixes[0]
	expectedFixMessage := "Add defer cancel()"
	if fix.Message != expectedFixMessage {
		t.Errorf("Expected SuggestedFix message %q, got %q", expectedFixMessage, fix.Message)
	}
}

func TestDiagnosticGenerator_CreateSuggestedFix(t *testing.T) {
	testCode := `
package test

import (
	"context"
	"cloud.google.com/go/spanner"
)

func testFunction(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	// Here needs to add defer client.Close()
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", testCode, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test code: %v", err)
	}

	generator := NewDiagnosticGenerator(fset)

	// Identify resource creation position
	var creationPos token.Pos
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "NewClient" {
					creationPos = call.Pos()
					return false
				}
			}
		}
		return true
	})

	suggestedFix := generator.CreateSuggestedFix("client", "Close", creationPos)

	// Expect English SuggestedFix message (updated for Task 15)
	expectedMessage := "Add defer client.Close() for client cleanup"
	if suggestedFix.Message != expectedMessage {
		t.Errorf("Expected SuggestedFix message %q, got %q", expectedMessage, suggestedFix.Message)
	}

	// Verify TextEdits existence
	if len(suggestedFix.TextEdits) == 0 {
		t.Error("Expected at least one TextEdit")
	}
}

func TestDiagnosticGenerator_ShouldIgnoreNolint(t *testing.T) {
	testCode := `
package test

import (
	"context"
	"cloud.google.com/go/spanner"
)

func testFunction(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test-project") //nolint:gcpclosecheck
	if err != nil {
		return err
	}
	// defer client.Close() intentionally omitted
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", testCode, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test code: %v", err)
	}

	generator := NewDiagnosticGenerator(fset)

	// Identify resource creation position
	var creationPos token.Pos
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "NewClient" {
					creationPos = call.Pos()
					return false
				}
			}
		}
		return true
	})

	shouldIgnore := generator.ShouldIgnoreNolint(file, creationPos)
	if !shouldIgnore {
		t.Error("Expected to ignore resource with //nolint:gcpclosecheck")
	}
}

func TestDiagnosticGenerator_NoNolintShouldNotIgnore(t *testing.T) {
	testCode := `
package test

import (
	"context"
	"cloud.google.com/go/spanner"
)

func testFunction(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", testCode, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test code: %v", err)
	}

	generator := NewDiagnosticGenerator(fset)

	// Identify resource creation position
	var creationPos token.Pos
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "NewClient" {
					creationPos = call.Pos()
					return false
				}
			}
		}
		return true
	})

	shouldIgnore := generator.ShouldIgnoreNolint(file, creationPos)
	if shouldIgnore {
		t.Error("Expected not to ignore resource without //nolint:gcpclosecheck")
	}
}

func TestDiagnosticGenerator_Integration(t *testing.T) {
	// Complex test case containing multiple resources
	testCode := `
package test

import (
	"context"
	"cloud.google.com/go/spanner"
)

func complexFunction(ctx context.Context) error {
	// Context with cancel - missing defer
	ctx, cancel := context.WithCancel(ctx)
	
	// Spanner client - missing defer
	client, err := spanner.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	
	// Transaction - missing defer  
	txn := client.ReadOnlyTransaction()
	
	// Iterator - missing defer
	iter := txn.Query(ctx, spanner.NewStatement("SELECT 1"))
	
	return nil
}
`

	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "test.go", testCode, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test code: %v", err)
	}

	generator := NewDiagnosticGenerator(fset)

	// Test diagnostic generation for multiple resources
	// In this case, 4 diagnostics are expected (cancel, client.Close, txn.Close, iter.Stop)

	// Note: This is an integration test focusing on DiagnosticGenerator functionality
	// The actual resource detection is handled by other components

	// Test successful generator creation
	if generator == nil {
		t.Error("Expected non-nil DiagnosticGenerator")
	}

	// Test FileSet accessibility
	if generator.fset == nil {
		t.Error("Expected DiagnosticGenerator to have FileSet")
	}
}

// TestDiagnosticGenerator_MessagesIntegration validates using messages package constants
func TestDiagnosticGenerator_MessagesIntegration(t *testing.T) {
	fset := token.NewFileSet()
	generator := NewDiagnosticGenerator(fset)

	// Test ResourceInfo with messages constants validation
	resource := ResourceInfo{
		Variable:         types.NewVar(token.Pos(100), nil, "client", nil),
		CreationPos:      token.Pos(100),
		ServiceType:      "spanner",
		CreationFunction: "NewClient",
		CleanupMethod:    "Close",
		IsRequired:       true,
	}

	diagnostic := generator.ReportMissingDefer(resource)

	// Verify that the message matches the messages package constant format
	expectedPattern := messages.MissingResourceCleanup
	if expectedPattern == "" {
		t.Error("messages.MissingResourceCleanup should not be empty")
	}

	// Verify message contains expected placeholders
	if !containsPlaceholders(expectedPattern, "%s", "%s") {
		t.Errorf("Expected pattern %q to contain %%s placeholders", expectedPattern)
	}

	// Test ContextInfo with messages constants validation
	contextInfo := ContextInfo{
		Variable:    types.NewVar(token.Pos(80), nil, "ctx", nil),
		CancelFunc:  types.NewVar(token.Pos(90), nil, "cancel", nil),
		CreationPos: token.Pos(80),
		IsDeferred:  false,
	}

	contextDiagnostic := generator.ReportMissingContextCancel(contextInfo)

	// Verify context message pattern
	expectedContextPattern := messages.MissingContextCancel
	if expectedContextPattern == "" {
		t.Error("messages.MissingContextCancel should not be empty")
	}

	if !containsPlaceholders(expectedContextPattern, "%s") {
		t.Errorf("Expected context pattern %q to contain %%s placeholder", expectedContextPattern)
	}

	// Verify SuggestedFix messages use correct constants
	resourceFix := diagnostic.SuggestedFixes[0]
	contextFix := contextDiagnostic.SuggestedFixes[0]

	// Verify suggested fix patterns exist and are not empty
	if messages.AddDeferMethodCall == "" {
		t.Error("messages.AddDeferMethodCall should not be empty")
	}
	if messages.AddDeferStatement == "" {
		t.Error("messages.AddDeferStatement should not be empty")
	}

	// Verify fixes contain expected content structure
	if resourceFix.Message == "" {
		t.Error("Resource fix message should not be empty")
	}
	if contextFix.Message == "" {
		t.Error("Context fix message should not be empty")
	}
}

// Helper function to check if a pattern contains expected placeholders
func containsPlaceholders(pattern string, placeholders ...string) bool {
	for _, placeholder := range placeholders {
		found := false
		for i := 0; i < len(pattern)-len(placeholder)+1; i++ {
			if pattern[i:i+len(placeholder)] == placeholder {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
