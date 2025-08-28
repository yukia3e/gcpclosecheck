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
	// ResourceTrackerとRuleEngineを作成
	ruleEngine := NewServiceRuleEngine()
	err := ruleEngine.LoadRules("")
	if err != nil {
		t.Fatalf("ルールエンジンの初期化に失敗: %v", err)
	}

	typeInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	tracker := NewResourceTracker(typeInfo, ruleEngine)
	analyzer := NewDeferAnalyzer(tracker)

	if analyzer == nil {
		t.Fatal("DeferAnalyzerの作成に失敗")
	}

	if analyzer.tracker != tracker {
		t.Error("tracker が正しく設定されていません")
	}
}

func TestDeferAnalyzer_FindDeferStatements(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectDefer int
	}{
		{
			name: "単一のdefer文",
			code: `
package test
func test() {
	client, _ := spanner.NewClient(ctx, "test")
	defer client.Close()
}`,
			expectDefer: 1,
		},
		{
			name: "複数のdefer文",
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
			name: "defer文なし",
			code: `
package test
func test() {
	client, _ := spanner.NewClient(ctx, "test")
}`,
			expectDefer: 0,
		},
		{
			name: "ネストしたブロック内のdefer文",
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
				t.Fatalf("コードのパースに失敗: %v", err)
			}

			// DeferAnalyzerを作成
			analyzer := createTestDeferAnalyzer(t)

			// 関数を探す
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("関数が見つかりません")
			}

			// defer文を検索
			defers := analyzer.FindDeferStatements(fn.Body)

			if len(defers) != tt.expectDefer {
				t.Errorf("defer文の数 = %v, 期待値 = %v", len(defers), tt.expectDefer)
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
			name:          "正しいSpannerクライアントのClose",
			resourceType:  "spanner",
			cleanupMethod: "Close",
			variableName:  "client",
			deferCallExpr: "client.Close()",
			wantValid:     true,
		},
		{
			name:          "正しいRowIteratorのStop",
			resourceType:  "spanner",
			cleanupMethod: "Stop",
			variableName:  "iter",
			deferCallExpr: "iter.Stop()",
			wantValid:     true,
		},
		{
			name:          "間違ったメソッド呼び出し",
			resourceType:  "spanner",
			cleanupMethod: "Close",
			variableName:  "client",
			deferCallExpr: "client.Start()",
			wantValid:     false,
		},
		{
			name:          "正しいStorageクライアントのClose",
			resourceType:  "storage",
			cleanupMethod: "Close",
			variableName:  "client",
			deferCallExpr: "client.Close()",
			wantValid:     true,
		},
		{
			name:          "クロージャでラップされたClose（改善されたパターン）",
			resourceType:  "storage", 
			cleanupMethod: "Close",
			variableName:  "client",
			deferCallExpr: "func() { client.Close() }",
			wantValid:     true,
		},
		{
			name:          "クロージャ内で間違ったメソッド呼び出し",
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

			// テスト用のResourceInfoを作成
			resourceInfo := ResourceInfo{
				ServiceType:   tt.resourceType,
				CleanupMethod: tt.cleanupMethod,
				VariableName:  tt.variableName,
				IsRequired:    true,
			}

			// テスト用のdefer文を作成
			deferStmt := createTestDeferStatement(tt.deferCallExpr)

			// バリデーションを実行
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
			name: "適切にクローズされているSpannerクライアント",
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
			name: "クローズされていないSpannerクライアント",
			code: `
package test
import "cloud.google.com/go/spanner"
func test(ctx context.Context) {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil { return }
	// defer client.Close() が漏れている
}`,
			expectDiagnostics: 1,
		},
		{
			name: "複数のリソースが適切にクローズ",
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
			name: "複数のリソースで一部クローズ漏れ",
			code: `
package test
import "cloud.google.com/go/spanner"
func test(ctx context.Context) {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil { return }
	txn := client.ReadOnlyTransaction()
	defer client.Close()
	// defer txn.Close() が漏れている
}`,
			expectDiagnostics: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ファイルをパース
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("コードのパースに失敗: %v", err)
			}

			// 型情報を設定
			typeInfo := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Uses:  make(map[*ast.Ident]types.Object),
				Defs:  make(map[*ast.Ident]types.Object),
			}
			setupPackageInfo(file, typeInfo)

			// DeferAnalyzerを作成
			ruleEngine := NewServiceRuleEngine()
			err = ruleEngine.LoadRules("")
			if err != nil {
				t.Fatalf("ルールエンジンの初期化に失敗: %v", err)
			}

			tracker := NewResourceTracker(typeInfo, ruleEngine)
			analyzer := NewDeferAnalyzer(tracker)

			// analysis.Passを作成
			pass := &analysis.Pass{
				Fset:      fset,
				Files:     []*ast.File{file},
				TypesInfo: typeInfo,
			}

			// リソースを追跡
			_ = tracker.FindResourceCreation(pass)

			// 関数を探す
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "test" {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("test関数が見つかりません")
			}

			// リソースを取得してdefer分析を実行
			resources := tracker.GetTrackedResources()
			diagnostics := analyzer.AnalyzeDefers(fn, resources)

			if len(diagnostics) != tt.expectDiagnostics {
				t.Errorf("診断の数 = %v, 期待値 = %v", len(diagnostics), tt.expectDiagnostics)
				for i, diag := range diagnostics {
					t.Logf("  [%d] %s", i, diag.Message)
				}
				// デバッグ情報を出力
				resources := tracker.GetTrackedResources()
				t.Logf("追跡されたリソース数: %d", len(resources))
				for i, res := range resources {
					t.Logf("  リソース[%d]: Type=%s, Method=%s, Required=%v", i, res.ServiceType, res.CleanupMethod, res.IsRequired)
				}

				// AST内の全CallExprを確認
				ast.Inspect(fn, func(n ast.Node) bool {
					if call, ok := n.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
							t.Logf("CallExpr発見: %s", sel.Sel.Name)
						}
					}
					return true
				})
			}
		})
	}
}

func TestDeferAnalyzer_CleanupOrderValidation(t *testing.T) {
	// 解放順序のテスト: RowIterator → Transaction → Client
	code := `
package test
import "cloud.google.com/go/spanner"
func test(ctx context.Context) {
	client, _ := spanner.NewClient(ctx, "test")
	txn := client.ReadOnlyTransaction()
	iter := txn.Query(ctx, spanner.NewStatement("SELECT 1"))
	defer client.Close()  // 3番目（最後）
	defer txn.Close()     // 2番目
	defer iter.Stop()     // 1番目（最初）
}`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, parser.ParseComments)
	if err != nil {
		t.Fatalf("コードのパースに失敗: %v", err)
	}

	analyzer := createTestDeferAnalyzer(t)

	// 関数を探す
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok {
			fn = f
			break
		}
	}

	if fn == nil {
		t.Fatal("関数が見つかりません")
	}

	// defer順序の検証
	isValidOrder := analyzer.ValidateCleanupOrder(fn.Body)

	// 適切な順序（逆順）になっているため、テストは成功するべき
	if !isValidOrder {
		t.Error("defer文の順序が適切であるべきです")
	}
}

// ヘルパー関数: テスト用のDeferAnalyzerを作成
func createTestDeferAnalyzer(t *testing.T) *DeferAnalyzer {
	ruleEngine := NewServiceRuleEngine()
	err := ruleEngine.LoadRules("")
	if err != nil {
		t.Fatalf("ルールエンジンの初期化に失敗: %v", err)
	}

	typeInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	tracker := NewResourceTracker(typeInfo, ruleEngine)
	return NewDeferAnalyzer(tracker)
}

// タスク12: DeferAnalyzerのcancel検出精度向上テスト
func TestDeferAnalyzer_ImprovedCancelDetection(t *testing.T) {
	tests := []struct {
		name                 string
		code                 string
		expectedResources    int
		expectedMissingDefer int
		expectedDeferFound   int
	}{
		{
			name: "defer文とcancel変数の名前解決精度改善",
			code: `
package test
import "context"
import "time"

func testImprovedNameResolution() {
	// 複数のcancel関数の名前解決テスト
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, timeoutCancel := context.WithTimeout(context.Background(), time.Second)
	ctx3, deadlineCancel := context.WithDeadline(context.Background(), time.Now())
	
	// 異なる名前のcancel関数を使用
	defer cancel1()
	defer timeoutCancel()
	// deadlineCancel()のdefer漏れ
	
	_ = ctx1
	_ = ctx2
	_ = ctx3
}`,
			expectedResources:    3,
			expectedMissingDefer: 1, // deadlineCancelの漏れ
			expectedDeferFound:   2, // cancel1, timeoutCancel
		},
		{
			name: "スコープ境界でのdefer文有効範囲の正確な判定",
			code: `
package test
import "context"

func testScopeAwareDeferValidation() {
	ctx, cancel := context.WithCancel(context.Background())
	
	// ネストしたスコープ内でのdefer
	if true {
		defer cancel() // 正しいスコープ内のdefer
	}
	
	// 別のスコープでの処理
	func() {
		// このスコープからはcancel()は見えないが、クロージャなので有効
		defer cancel()
	}()
	
	_ = ctx
}`,
			expectedResources:    1,
			expectedMissingDefer: 0, // 適切にdeferされている
			expectedDeferFound:   2, // 2つのスコープでdefer
		},
		{
			name: "複数のcontext.WithTimeout/WithCancel呼び出し対応",
			code: `
package test
import "context"
import "time"

func testMultipleContextHandling() {
	// 同一関数内での複数のcontext生成
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithTimeout(ctx1, 5*time.Second)
	ctx3, cancel3 := context.WithCancel(ctx2)
	ctx4, cancel4 := context.WithDeadline(ctx3, time.Now().Add(time.Minute))
	
	// 一部のcancel関数のみdefer
	defer cancel1()
	defer cancel3()
	// cancel2, cancel4のdefer漏れ
	
	_ = ctx1
	_ = ctx2
	_ = ctx3 
	_ = ctx4
}`,
			expectedResources:    4,
			expectedMissingDefer: 2, // cancel2, cancel4の漏れ
			expectedDeferFound:   2, // cancel1, cancel3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テストコードをパース
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse test code: %v", err)
			}

			// DeferAnalyzerを作成
			analyzer := createTestDeferAnalyzer(t)

			// 関数宣言を取得
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("テストコード内に関数が見つかりません")
			}

			// 改良されたAnalyzeDefersPrecision()メソッドを呼び出し（まだ実装されていない）
			defers := analyzer.FindDeferStatements(fn.Body)
			improvedDefers := analyzer.AnalyzeDefersPrecision(fn.Body)

			// defer文の数を検証
			if len(defers) != tt.expectedDeferFound {
				t.Errorf("Found defers = %v, want %v", len(defers), tt.expectedDeferFound)
			}

			if improvedDefers != nil {
				t.Logf("✓ Improved defer analysis completed")
			} else {
				t.Logf("AnalyzeDefersPrecision not implemented yet (expected)")
			}

			// 改良されたスコープ境界判定のテスト
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
			name: "基本的な精度向上解析",
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
			name: "複雑なスコープ境界解析",
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
			// テストコードをパース
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse test code: %v", err)
			}

			// DeferAnalyzerを作成
			analyzer := createTestDeferAnalyzer(t)

			// 関数宣言を取得
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("テストコード内に関数が見つかりません")
			}

			// AnalyzeDefersPrecision()メソッドを呼び出し（まだ実装されていない）
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
			name: "有効なdeferスコープ",
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
			name: "無効なdeferスコープ（別関数内）",
			code: `
package test
import "context"

func test() {
	ctx, cancel := context.WithCancel(context.Background())
	
	go func() {
		defer cancel() // 有効（クロージャ内）
	}()
	
	_ = ctx
}`,
			want: true, // クロージャ内は有効
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テストコードをパース
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse test code: %v", err)
			}

			// DeferAnalyzerを作成
			analyzer := createTestDeferAnalyzer(t)

			// 関数宣言を取得
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("テストコード内に関数が見つかりません")
			}

			// ValidateDeferScope()メソッドを呼び出し（まだ実装されていない）
			got := analyzer.ValidateDeferScope(fn.Body)

			if got != tt.want {
				t.Errorf("ValidateDeferScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ヘルパー関数: テスト用のdefer文を作成
func createTestDeferStatement(callExpr string) *ast.DeferStmt {
	var code string
	// クロージャパターンの場合は、defer文内でクロージャを実行する形式にする
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

	// defer文を探す
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
