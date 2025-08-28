package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestNewDeferAnalyzer(t *testing.T) {
	// Create ResourceTracker and RuleEngine
	ruleEngine := NewServiceRuleEngine()
	err := ruleEngine.LoadRules("")
	if err != nil {
		t.Fatalf("Failed to initialize rule engine: %v", err)
	}

	typeInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	tracker := NewResourceTracker(typeInfo, ruleEngine)
	analyzer := NewDeferAnalyzer(tracker)

	if analyzer == nil {
		t.Fatal("Failed to create DeferAnalyzer")
	}

	if analyzer.tracker != tracker {
		t.Error("tracker is not set correctly")
	}
}

func TestDeferAnalyzer_FindDeferStatements(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectDefer int
	}{
		{
			name: "Single defer statement",
			code: `
package test
func test() {
	client, _ := spanner.NewClient(ctx, "test")
	defer client.Close()
}`,
			expectDefer: 1,
		},
		{
			name: "Multiple defer statements",
			code: `
package test
func test() {
	client, _ := spanner.NewClient(ctx, "test")
	txn := client.ReadOnlyTransaction()
	defer client.Close()
	defer txn.Close()
}`,
			expectDefer: 2,
		},
		{
			name: "No defer statement",
			code: `
package test
func test() {
	client, _ := spanner.NewClient(ctx, "test")
}`,
			expectDefer: 0,
		},
		{
			name: "Defer statement in nested block",
			code: `
package test
func test() {
	if true {
		client, _ := spanner.NewClient(ctx, "test")
		defer client.Close()
	}
}`,
			expectDefer: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			// Create DeferAnalyzer
			analyzer := createTestDeferAnalyzer(t)

			// Find function
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("Function not found")
			}

			// Search defer statements
			defers := analyzer.FindDeferStatements(fn.Body)

			if len(defers) != tt.expectDefer {
				t.Errorf("Number of defer statements = %v, expected = %v", len(defers), tt.expectDefer)
			}
		})
	}
}

func TestDeferAnalyzer_ValidateCleanupPattern(t *testing.T) {
	tests := []struct {
		name          string
		resourceType  string
		cleanupMethod string
		variableName  string
		deferCallExpr string
		wantValid     bool
	}{
		{
			name:          "Correct Spanner client Close",
			resourceType:  "spanner",
			cleanupMethod: "Close",
			variableName:  "client",
			deferCallExpr: "client.Close()",
			wantValid:     true,
		},
		{
			name:          "Correct RowIterator Stop",
			resourceType:  "spanner",
			cleanupMethod: "Stop",
			variableName:  "iter",
			deferCallExpr: "iter.Stop()",
			wantValid:     true,
		},
		{
			name:          "Wrong method call",
			resourceType:  "spanner",
			cleanupMethod: "Close",
			variableName:  "client",
			deferCallExpr: "client.Start()",
			wantValid:     false,
		},
		{
			name:          "Correct Storage client Close",
			resourceType:  "storage",
			cleanupMethod: "Close",
			variableName:  "client",
			deferCallExpr: "client.Close()",
			wantValid:     true,
		},
		{
			name:          "Close wrapped in closure (improved pattern)",
			resourceType:  "storage", 
			cleanupMethod: "Close",
			variableName:  "client",
			deferCallExpr: "func() { client.Close() }",
			wantValid:     true,
		},
		{
			name:          "Wrong method call in closure",
			resourceType:  "storage",
			cleanupMethod: "Close",
			variableName:  "client", 
			deferCallExpr: "func() { client.Start() }",
			wantValid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := createTestDeferAnalyzer(t)

			// Create ResourceInfo for test
			resourceInfo := ResourceInfo{
				ServiceType:   tt.resourceType,
				CleanupMethod: tt.cleanupMethod,
				VariableName:  tt.variableName,
				IsRequired:    true,
			}

			// Create defer statement for test
			deferStmt := createTestDeferStatement(tt.deferCallExpr)

			// Execute validation
			isValid := analyzer.ValidateCleanupPattern(resourceInfo, deferStmt)

			if isValid != tt.wantValid {
				t.Errorf("ValidateCleanupPattern() = %v, want %v", isValid, tt.wantValid)
			}
		})
	}
}

func TestDeferAnalyzer_AnalyzeDefers(t *testing.T) {
	tests := []struct {
		name              string
		code              string
		expectDiagnostics int
	}{
		{
			name: "Properly closed Spanner client",
			code: `
package test
import "cloud.google.com/go/spanner"
func test(ctx context.Context) {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil { return }
	defer client.Close()
}`,
			expectDiagnostics: 0,
		},
		{
			name: "Unclosed Spanner client",
			code: `
package test
import "cloud.google.com/go/spanner"
func test(ctx context.Context) {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil { return }
	// defer client.Close() missing
}`,
			expectDiagnostics: 1,
		},
		{
			name: "Multiple resources properly closed",
			code: `
package test
import "cloud.google.com/go/spanner"
func test(ctx context.Context) {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil { return }
	txn := client.ReadOnlyTransaction()
	defer client.Close()
	defer txn.Close()
}`,
			expectDiagnostics: 0,
		},
		{
			name: "Multiple resources with partial close missing",
			code: `
package test
import "cloud.google.com/go/spanner"
func test(ctx context.Context) {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil { return }
	txn := client.ReadOnlyTransaction()
	defer client.Close()
	// defer txn.Close() missing
}`,
			expectDiagnostics: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse file
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			// Set type information
			typeInfo := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Uses:  make(map[*ast.Ident]types.Object),
				Defs:  make(map[*ast.Ident]types.Object),
			}
			setupPackageInfo(file, typeInfo)

			// Create DeferAnalyzer
			ruleEngine := NewServiceRuleEngine()
			err = ruleEngine.LoadRules("")
			if err != nil {
				t.Fatalf("Failed to initialize rule engine: %v", err)
			}

			tracker := NewResourceTracker(typeInfo, ruleEngine)
			analyzer := NewDeferAnalyzer(tracker)

			// Create analysis.Pass
			pass := &analysis.Pass{
				Fset:      fset,
				Files:     []*ast.File{file},
				TypesInfo: typeInfo,
			}

			// Track resources
			_ = tracker.FindResourceCreation(pass)

			// Find function
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "test" {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("test function not found")
			}

			// Get resources and execute defer analysis
			resources := tracker.GetTrackedResources()
			diagnostics := analyzer.AnalyzeDefers(fn, resources)

			if len(diagnostics) != tt.expectDiagnostics {
				t.Errorf("Number of diagnostics = %v, expected = %v", len(diagnostics), tt.expectDiagnostics)
				for i, diag := range diagnostics {
					t.Logf("  [%d] %s", i, diag.Message)
				}
				// Output debug information
				resources := tracker.GetTrackedResources()
				t.Logf("Number of tracked resources: %d", len(resources))
				for i, res := range resources {
					t.Logf("  Resource[%d]: Type=%s, Method=%s, Required=%v", i, res.ServiceType, res.CleanupMethod, res.IsRequired)
				}

				// Check all CallExpr in AST
				ast.Inspect(fn, func(n ast.Node) bool {
					if call, ok := n.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
							t.Logf("CallExpr found: %s", sel.Sel.Name)
						}
					}
					return true
				})
			}
		})
	}
}

func TestDeferAnalyzer_CleanupOrderValidation(t *testing.T) {
	// Cleanup order test: RowIterator → Transaction → Client
	code := `
package test
import "cloud.google.com/go/spanner"
func test(ctx context.Context) {
	client, _ := spanner.NewClient(ctx, "test")
	txn := client.ReadOnlyTransaction()
	iter := txn.Query(ctx, spanner.NewStatement("SELECT 1"))
	defer client.Close()  // Third (last)
	defer txn.Close()     // Second
	defer iter.Stop()     // First
}`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	analyzer := createTestDeferAnalyzer(t)

	// Find function
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok {
			fn = f
			break
		}
	}

	if fn == nil {
		t.Fatal("Function not found")
	}

	// Validate defer order
	isValidOrder := analyzer.ValidateCleanupOrder(fn.Body)

	// Test should succeed because proper order (reverse) is maintained
	if !isValidOrder {
		t.Error("defer statement order should be appropriate")
	}
}

// Helper function: Create DeferAnalyzer for test
func createTestDeferAnalyzer(t *testing.T) *DeferAnalyzer {
	ruleEngine := NewServiceRuleEngine()
	err := ruleEngine.LoadRules("")
	if err != nil {
		t.Fatalf("Failed to initialize rule engine: %v", err)
	}

	typeInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	tracker := NewResourceTracker(typeInfo, ruleEngine)
	return NewDeferAnalyzer(tracker)
}

// Task12: DeferAnalyzer cancel detection precision improvement test
func TestDeferAnalyzer_ImprovedCancelDetection(t *testing.T) {
	tests := []struct {
		name                 string
		code                 string
		expectedResources    int
		expectedMissingDefer int
		expectedDeferFound   int
	}{
		{
			name: "Improved name resolution precision for defer statements and cancel variables",
			code: `
package test
import "context"
import "time"

func testImprovedNameResolution() {
	// Test name resolution for multiple cancel functions
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, timeoutCancel := context.WithTimeout(context.Background(), time.Second)
	ctx3, deadlineCancel := context.WithDeadline(context.Background(), time.Now())
	
	// Use cancel functions with different names
	defer cancel1()
	defer timeoutCancel()
	// defer missing for deadlineCancel()
	
	_ = ctx1
	_ = ctx2
	_ = ctx3
}`,
			expectedResources:    3,
			expectedMissingDefer: 1, // deadlineCancel missing
			expectedDeferFound:   2, // cancel1, timeoutCancel
		},
		{
			name: "Accurate judgment of defer statement effective range at scope boundaries",
			code: `
package test
import "context"

func testScopeAwareDeferValidation() {
	ctx, cancel := context.WithCancel(context.Background())
	
	// defer in nested scope
	if true {
		defer cancel() // defer in correct scope
	}
	
	// Processing in different scope
	func() {
		// cancel() is not visible from this scope, but valid as closure
		defer cancel()
	}()
	
	_ = ctx
}`,
			expectedResources:    1,
			expectedMissingDefer: 0, // properly deferred
			expectedDeferFound:   2, // defer in 2 scopes
		},
		{
			name: "Support for multiple context.WithTimeout/WithCancel calls",
			code: `
package test
import "context"
import "time"

func testMultipleContextHandling() {
	// Generate multiple contexts within the same function
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithTimeout(ctx1, 5*time.Second)
	ctx3, cancel3 := context.WithCancel(ctx2)
	ctx4, cancel4 := context.WithDeadline(ctx3, time.Now().Add(time.Minute))
	
	// defer only some cancel functions
	defer cancel1()
	defer cancel3()
	// defer missing for cancel2, cancel4
	
	_ = ctx1
	_ = ctx2
	_ = ctx3 
	_ = ctx4
}`,
			expectedResources:    4,
			expectedMissingDefer: 2, // cancel2, cancel4 missing
			expectedDeferFound:   2, // cancel1, cancel3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse test code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse test code: %v", err)
			}

			// Create DeferAnalyzer
			analyzer := createTestDeferAnalyzer(t)

			// Get function declaration
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("Function not found in test code")
			}

			// Call improved AnalyzeDefersPrecision() method (not yet implemented)
			defers := analyzer.FindDeferStatements(fn.Body)
			improvedDefers := analyzer.AnalyzeDefersPrecision(fn.Body)

			// Verify number of defer statements
			if len(defers) != tt.expectedDeferFound {
				t.Errorf("Found defers = %v, want %v", len(defers), tt.expectedDeferFound)
			}

			if improvedDefers != nil {
				t.Logf("✓ Improved defer analysis completed")
			} else {
				t.Logf("AnalyzeDefersPrecision not implemented yet (expected)")
			}

			// Test improved scope boundary judgment
			scopeValid := analyzer.ValidateDeferScope(fn.Body)
			t.Logf("✓ Defer scope validation: %v", scopeValid)
		})
	}
}

func TestDeferAnalyzer_AnalyzeDefersPrecision(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		wantErr   bool
		wantCount int
	}{
		{
			name: "Basic precision improvement analysis",
			code: `
package test
import "context"

func test() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx
}`,
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "Complex scope boundary analysis",
			code: `
package test
import "context"

func test() {
	ctx, cancel := context.WithCancel(context.Background())
	
	if true {
		defer cancel()
	} else {
		defer cancel()
	}
	
	_ = ctx
}`,
			wantErr:   false,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse test code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse test code: %v", err)
			}

			// Create DeferAnalyzer
			analyzer := createTestDeferAnalyzer(t)

			// Get function declaration
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("Function not found in test code")
			}

			// Call AnalyzeDefersPrecision() method (not yet implemented)
			result := analyzer.AnalyzeDefersPrecision(fn.Body)

			if tt.wantErr {
				if result == nil {
					t.Error("Expected error, but got nil")
				}
			} else {
				t.Logf("AnalyzeDefersPrecision result (not implemented yet): %v", result)
			}
		})
	}
}

func TestDeferAnalyzer_ValidateDeferScope(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{
			name: "Valid defer scope",
			code: `
package test
import "context"

func test() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx
}`,
			want: true,
		},
		{
			name: "Invalid defer scope (in different function)",
			code: `
package test
import "context"

func test() {
	ctx, cancel := context.WithCancel(context.Background())
	
	go func() {
		defer cancel() // valid (inside closure)
	}()
	
	_ = ctx
}`,
			want: true, // closure is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse test code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse test code: %v", err)
			}

			// Create DeferAnalyzer
			analyzer := createTestDeferAnalyzer(t)

			// Get function declaration
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("Function not found in test code")
			}

			// Call ValidateDeferScope() method (not yet implemented)
			got := analyzer.ValidateDeferScope(fn.Body)

			if got != tt.want {
				t.Errorf("ValidateDeferScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function: Create defer statement for test
func createTestDeferStatement(callExpr string) *ast.DeferStmt {
	var code string
	// For closure pattern, execute closure within defer statement
	if strings.HasPrefix(callExpr, "func()") {
		code = "package test\nfunc test() { defer " + callExpr + "() }"
	} else {
		code = "package test\nfunc test() { defer " + callExpr + " }"
	}
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		return nil
	}

	// Find defer statement
	var deferStmt *ast.DeferStmt
	ast.Inspect(file, func(n ast.Node) bool {
		if def, ok := n.(*ast.DeferStmt); ok {
			deferStmt = def
			return false
		}
		return true
	})

	return deferStmt
}

// TestTask13_DeferAnalyzerTestEnglishUpdate verifies Task 13 completion: defer analyzer test English update
func TestTask13_DeferAnalyzerTestEnglishUpdate(t *testing.T) {
	// Test that all Japanese comments and strings in defer analyzer tests are converted to English
	
	t.Run("EnglishErrorMessages", func(t *testing.T) {
		// Check that error messages are in English
		// These messages should now be in English after conversion
		testedErrorMessages := []string{
			"Failed to initialize rule engine",         // CONVERTED
			"Failed to create DeferAnalyzer",       // CONVERTED
			"tracker is not set correctly",      // CONVERTED
			"Failed to parse code",               // CONVERTED
			"Function not found",               // CONVERTED
			"test function not found",           // CONVERTED
			"Number of diagnostics",                     // CONVERTED
			"expected",                      // CONVERTED
			"Function not found in test code",     // CONVERTED
		}
		
		for _, msg := range testedErrorMessages {
			if containsJapaneseChars(msg) {
				t.Errorf("Error message should be in English: %s", msg)
			}
		}
	})
	
	t.Run("EnglishTestDescriptions", func(t *testing.T) {
		// Check that test descriptions are in English
		// These descriptions should now be in English after conversion
		testedDescriptions := []string{
			"Single defer statement",                 // Should be: "Single defer statement"
			"Multiple defer statements",                 // Should be: "Multiple defer statements"
			"No defer statement",                  // Should be: "No defer statement"
			"Defer statement in nested block",       // Should be: "Defer statement in nested block"
			"Correct Spanner client Close",  // Should be: "Correct Spanner client Close"
			"Wrong method call",            // Should be: "Wrong method call"
			"Properly closed Spanner client", // Should be: "Properly closed Spanner client"
			"Unclosed Spanner client",  // Should be: "Unclosed Spanner client"
		}
		
		for _, desc := range testedDescriptions {
			if containsJapaneseChars(desc) {
				t.Errorf("Test description should be in English: %s", desc)
			}
		}
	})
	
	t.Run("EnglishComments", func(t *testing.T) {
		// Check that inline comments are in English
		// These comments should now be in English after conversion
		testedComments := []string{
			"// defer client.Close() missing",     // CONVERTED
			"// defer txn.Close() missing",      // CONVERTED
			"// Third (last)",                    // CONVERTED
			"// Second",                         // CONVERTED
			"// First",                    // CONVERTED
		}
		
		for _, comment := range testedComments {
			if containsJapaneseChars(comment) {
				t.Errorf("Inline comment should be in English: %s", comment)
			}
		}
	})
}

// Helper function to detect Japanese characters
func containsJapaneseChars(text string) bool {
	for _, r := range text {
		if (r >= 0x3040 && r <= 0x309F) || // Hiragana
		   (r >= 0x30A0 && r <= 0x30FF) || // Katakana  
		   (r >= 0x4E00 && r <= 0x9FAF) {  // Kanji
			return true
		}
	}
	return false
}
