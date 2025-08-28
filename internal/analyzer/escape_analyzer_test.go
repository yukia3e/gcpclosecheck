package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestNewEscapeAnalyzer(t *testing.T) {
	analyzer := NewEscapeAnalyzer()

	if analyzer == nil {
		t.Fatal("NewEscapeAnalyzer は nil を返すべきではありません")
	}

	if analyzer.escapeInfo == nil {
		t.Error("escapeInfo マップが初期化されていません")
	}
}

func TestEscapeAnalyzer_IsReturnedValue(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		varName string
		want    bool
	}{
		{
			name: "関数戻り値として返されるリソース",
			code: `
package test
import "cloud.google.com/go/spanner"
func createClient(ctx context.Context) (*spanner.Client, error) {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return nil, err
	}
	return client, nil // clientが戻り値として返される
}`,
			varName: "client",
			want:    true,
		},
		{
			name: "関数戻り値として返されないリソース",
			code: `
package test
import "cloud.google.com/go/spanner"
func useClient(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return err
	}
	defer client.Close()
	// clientは戻り値として返されない
	return nil
}`,
			varName: "client",
			want:    false,
		},
		{
			name: "複数戻り値の一つとして返される",
			code: `
package test
import "cloud.google.com/go/spanner"
func createMultiple(ctx context.Context) (*spanner.Client, *spanner.ReadOnlyTransaction, error) {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return nil, nil, err
	}
	txn := client.ReadOnlyTransaction()
	return client, txn, nil // 両方とも戻り値として返される
}`,
			varName: "txn",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// コードをパース
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("コードのパースに失敗: %v", err)
			}

			// EscapeAnalyzerを作成
			analyzer := NewEscapeAnalyzer()

			// 関数を取得
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

			// 変数を検索
			var targetVar *types.Var
			ast.Inspect(fn, func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok && ident.Name == tt.varName {
					// 簡易的にtypes.Varを作成（実際の実装では型情報が必要）
					targetVar = types.NewVar(ident.Pos(), nil, tt.varName, nil)
					return false
				}
				return true
			})

			// 戻り値チェックを実行
			got := analyzer.IsReturnedValue(targetVar, fn)
			if got != tt.want {
				t.Errorf("IsReturnedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeAnalyzer_IsFieldAssigned(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		varName string
		want    bool
	}{
		{
			name: "構造体フィールドに代入されるリソース",
			code: `
package test
import "cloud.google.com/go/spanner"
type Service struct {
	client *spanner.Client
}
func (s *Service) init(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return err
	}
	s.client = client // フィールドに代入
	return nil
}`,
			varName: "client",
			want:    true,
		},
		{
			name: "フィールドに代入されないリソース",
			code: `
package test
import "cloud.google.com/go/spanner"
func localUse(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return err
	}
	defer client.Close()
	// フィールドに代入されない
	return nil
}`,
			varName: "client",
			want:    false,
		},
		{
			name: "埋め込み構造体のフィールドに代入",
			code: `
package test
import "cloud.google.com/go/spanner"
type Config struct {
	DB struct {
		client *spanner.Client
	}
}
func (c *Config) setup(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return err
	}
	c.DB.client = client // ネストしたフィールドに代入
	return nil
}`,
			varName: "client",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// コードをパース
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("コードのパースに失敗: %v", err)
			}

			// EscapeAnalyzerを作成
			analyzer := NewEscapeAnalyzer()

			// 関数を取得
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok && f.Name != nil {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("関数が見つかりません")
			}

			// 変数を検索
			var targetVar *types.Var
			ast.Inspect(fn, func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok && ident.Name == tt.varName {
					// 簡易的にtypes.Varを作成
					targetVar = types.NewVar(ident.Pos(), nil, tt.varName, nil)
					return false
				}
				return true
			})

			// フィールド代入チェックを実行
			got := analyzer.IsFieldAssigned(targetVar, fn)
			if got != tt.want {
				t.Errorf("IsFieldAssigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeAnalyzer_AnalyzeEscape(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		varName          string
		wantIsReturned   bool
		wantIsAssigned   bool
		wantEscapeReason string
	}{
		{
			name: "戻り値として逃げるリソース",
			code: `
package test
import "cloud.google.com/go/spanner"
func createClient(ctx context.Context) (*spanner.Client, error) {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return nil, err
	}
	return client, nil
}`,
			varName:          "client",
			wantIsReturned:   true,
			wantIsAssigned:   false,
			wantEscapeReason: "returned from function",
		},
		{
			name: "フィールドに代入されて逃げるリソース",
			code: `
package test
import "cloud.google.com/go/spanner"
type Service struct {
	client *spanner.Client
}
func (s *Service) init(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return err
	}
	s.client = client
	return nil
}`,
			varName:          "client",
			wantIsReturned:   false,
			wantIsAssigned:   true,
			wantEscapeReason: "assigned to struct field",
		},
		{
			name: "逃げないローカルリソース",
			code: `
package test
import "cloud.google.com/go/spanner"
func localUse(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return err
	}
	defer client.Close()
	return nil
}`,
			varName:          "client",
			wantIsReturned:   false,
			wantIsAssigned:   false,
			wantEscapeReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// コードをパース
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("コードのパースに失敗: %v", err)
			}

			// EscapeAnalyzerを作成
			analyzer := NewEscapeAnalyzer()

			// 関数を取得
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok && f.Name != nil {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("関数が見つかりません")
			}

			// 変数を検索
			var targetVar *types.Var
			ast.Inspect(fn, func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok && ident.Name == tt.varName {
					targetVar = types.NewVar(ident.Pos(), nil, tt.varName, nil)
					return false
				}
				return true
			})

			if targetVar == nil {
				t.Fatalf("変数 %s が見つかりません", tt.varName)
			}

			// エスケープ分析を実行
			escapeInfo := analyzer.AnalyzeEscape(targetVar, fn)

			if escapeInfo.IsReturned != tt.wantIsReturned {
				t.Errorf("EscapeInfo.IsReturned = %v, want %v", escapeInfo.IsReturned, tt.wantIsReturned)
			}

			if escapeInfo.IsFieldAssigned != tt.wantIsAssigned {
				t.Errorf("EscapeInfo.IsFieldAssigned = %v, want %v", escapeInfo.IsFieldAssigned, tt.wantIsAssigned)
			}

			if escapeInfo.EscapeReason != tt.wantEscapeReason {
				t.Errorf("EscapeInfo.EscapeReason = %q, want %q", escapeInfo.EscapeReason, tt.wantEscapeReason)
			}
		})
	}
}

func TestEscapeAnalyzer_ShouldSkipResource(t *testing.T) {
	tests := []struct {
		name         string
		resourceInfo ResourceInfo
		escapeInfo   EscapeInfo
		want         bool
		wantReason   string
	}{
		{
			name: "戻り値として返されるリソースはスキップ",
			resourceInfo: ResourceInfo{
				ServiceType:      "spanner",
				CreationFunction: "NewClient",
				CleanupMethod:    "Close",
				IsRequired:       true,
			},
			escapeInfo: EscapeInfo{
				IsReturned:      true,
				IsFieldAssigned: false,
				EscapeReason:    "returned from function",
			},
			want:       true,
			wantReason: "returned from function",
		},
		{
			name: "フィールドに代入されるリソースはスキップ",
			resourceInfo: ResourceInfo{
				ServiceType:      "spanner",
				CreationFunction: "NewClient",
				CleanupMethod:    "Close",
				IsRequired:       true,
			},
			escapeInfo: EscapeInfo{
				IsReturned:      false,
				IsFieldAssigned: true,
				EscapeReason:    "assigned to struct field",
			},
			want:       true,
			wantReason: "assigned to struct field",
		},
		{
			name: "RowIteratorは関数内で処理されるべき（スキップしない）",
			resourceInfo: ResourceInfo{
				ServiceType:      "spanner",
				CreationFunction: "Query",
				CleanupMethod:    "Stop",
				IsRequired:       true,
			},
			escapeInfo: EscapeInfo{
				IsReturned:      true,
				IsFieldAssigned: false,
				EscapeReason:    "returned from function",
			},
			want:       false, // RowIteratorは特別扱い
			wantReason: "",
		},
		{
			name: "ローカルリソースはスキップしない",
			resourceInfo: ResourceInfo{
				ServiceType:      "spanner",
				CreationFunction: "NewClient",
				CleanupMethod:    "Close",
				IsRequired:       true,
			},
			escapeInfo: EscapeInfo{
				IsReturned:      false,
				IsFieldAssigned: false,
				EscapeReason:    "",
			},
			want:       false,
			wantReason: "",
		},
	}

	analyzer := NewEscapeAnalyzer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldSkip, reason := analyzer.ShouldSkipResource(tt.resourceInfo, tt.escapeInfo)

			if shouldSkip != tt.want {
				t.Errorf("ShouldSkipResource() = %v, want %v", shouldSkip, tt.want)
			}

			if reason != tt.wantReason {
				t.Errorf("ShouldSkipResource() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestEscapeAnalyzer_IntegrationWithDeferAnalyzer(t *testing.T) {
	tests := []struct {
		name                string
		code                string
		varName             string
		expectDeferRequired bool
		reason              string
	}{
		{
			name: "戻り値として返されるクライアントはdefer不要",
			code: `
package test
import "cloud.google.com/go/spanner"
func createClient(ctx context.Context) (*spanner.Client, error) {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return nil, err
	}
	return client, nil // clientが戻り値として返される
}`,
			varName:             "client",
			expectDeferRequired: false,
			reason:              "returned from function",
		},
		{
			name: "フィールドに代入されるクライアントはdefer不要",
			code: `
package test
import "cloud.google.com/go/spanner"
type Service struct {
	client *spanner.Client
}
func (s *Service) init(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return err
	}
	s.client = client // フィールドに代入
	return nil
}`,
			varName:             "client",
			expectDeferRequired: false,
			reason:              "assigned to struct field",
		},
		{
			name: "RowIteratorは戻り値でもdefer必要（特例）",
			code: `
package test
import "cloud.google.com/go/spanner"
func queryData(ctx context.Context, txn *spanner.ReadOnlyTransaction) (*spanner.RowIterator, error) {
	iter := txn.Query(ctx, stmt)
	return iter, nil // Iteratorが戻り値として返される
}`,
			varName:             "iter",
			expectDeferRequired: true, // RowIteratorは特別扱い
			reason:              "",
		},
		{
			name: "ローカル使用のリソースはdefer必要",
			code: `
package test
import "cloud.google.com/go/spanner"
func localUse(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "test")
	if err != nil {
		return err
	}
	// ローカルで使用、戻り値やフィールドに代入されない
	return nil
}`,
			varName:             "client",
			expectDeferRequired: true,
			reason:              "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// コードをパース
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("コードのパースに失敗: %v", err)
			}

			analyzer := NewEscapeAnalyzer()

			// 関数を取得
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok && f.Name != nil {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("関数が見つかりません")
			}

			// 変数を検索
			var targetVar *types.Var
			ast.Inspect(fn, func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok && ident.Name == tt.varName {
					targetVar = types.NewVar(ident.Pos(), nil, tt.varName, nil)
					return false
				}
				return true
			})

			if targetVar == nil {
				t.Fatalf("変数 %s が見つかりません", tt.varName)
			}

			// エスケープ分析を実行
			escapeInfo := analyzer.AnalyzeEscape(targetVar, fn)

			// リソース情報を作成（テスト用）
			resourceInfo := ResourceInfo{
				ServiceType:      "spanner",
				CreationFunction: "NewClient",
				CleanupMethod:    "Close",
				IsRequired:       true,
			}

			// RowIteratorの場合は特別設定
			if tt.varName == "iter" {
				resourceInfo.CreationFunction = "Query"
				resourceInfo.CleanupMethod = "Stop"
			}

			// スキップ判定
			shouldSkip, reason := analyzer.ShouldSkipResource(resourceInfo, escapeInfo)
			deferRequired := !shouldSkip

			if deferRequired != tt.expectDeferRequired {
				t.Errorf("defer required = %v, want %v", deferRequired, tt.expectDeferRequired)
			}

			if reason != tt.reason {
				t.Errorf("skip reason = %q, want %q", reason, tt.reason)
			}
		})
	}
}

// TestEscapeAnalyzer_DetectSpannerAutoManagement - Spannerクロージャ管理パターン検出のテスト（RED: 失敗テスト）
func TestEscapeAnalyzer_DetectSpannerAutoManagement(t *testing.T) {
	tests := []struct {
		name                     string
		code                     string
		varName                  string
		wantIsAutoManaged        bool
		wantTransactionType      string
		wantIsClosureManaged     bool
		wantAutoManagementReason string
	}{
		{
			name: "ReadWriteTransaction クロージャパターン",
			code: `
package test
import "cloud.google.com/go/spanner"
func testFunction() {
	client := &spanner.Client{}
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// txn は自動管理される
		return txn.Update(ctx, stmt)
	})
}`,
			varName:                  "txn",
			wantIsAutoManaged:        true,
			wantTransactionType:      "ReadWriteTransaction",
			wantIsClosureManaged:     true,
			wantAutoManagementReason: "ReadWriteTransactionクロージャ内で自動管理",
		},
		{
			name: "ReadOnlyTransaction クロージャパターン",
			code: `
package test
import "cloud.google.com/go/spanner"
func testFunction() {
	client := &spanner.Client{}
	txn := client.ReadOnlyTransaction()
	defer txn.Close()
}`,
			varName:                  "txn",
			wantIsAutoManaged:        false,
			wantTransactionType:      "",
			wantIsClosureManaged:     false,
			wantAutoManagementReason: "",
		},
		{
			name: "手動管理トランザクション（クロージャなし）",
			code: `
package test
import "cloud.google.com/go/spanner"
func testFunction() {
	client := &spanner.Client{}
	txn := client.ReadOnlyTransaction()
	// defer txn.Close() が必要
	_ = txn
}`,
			varName:                  "txn",
			wantIsAutoManaged:        false,
			wantTransactionType:      "",
			wantIsClosureManaged:     false,
			wantAutoManagementReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// コードをパースしてASTを作成
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("コードのパースに失敗: %v", err)
			}

			// EscapeAnalyzerを作成
			analyzer := NewEscapeAnalyzer()

			// 関数を取得
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok && f.Name != nil {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("関数が見つかりません")
			}

			// 変数を検索
			var targetVar *types.Var
			ast.Inspect(fn, func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok && ident.Name == tt.varName {
					targetVar = types.NewVar(ident.Pos(), nil, tt.varName, nil)
					return false
				}
				return true
			})

			// DetectSpannerAutoManagement を呼び出し（まだ未実装なので失敗する）
			spannerEscape := analyzer.DetectSpannerAutoManagement(targetVar, fn)

			// 検証
			if spannerEscape == nil {
				if tt.wantIsAutoManaged {
					t.Error("DetectSpannerAutoManagement() が nil を返しましたが、SpannerEscapeInfo が期待されます")
				}
				return
			}

			if spannerEscape.IsAutoManaged != tt.wantIsAutoManaged {
				t.Errorf("SpannerEscapeInfo.IsAutoManaged = %v, want %v", spannerEscape.IsAutoManaged, tt.wantIsAutoManaged)
			}

			if spannerEscape.TransactionType != tt.wantTransactionType {
				t.Errorf("SpannerEscapeInfo.TransactionType = %q, want %q", spannerEscape.TransactionType, tt.wantTransactionType)
			}

			if spannerEscape.IsClosureManaged != tt.wantIsClosureManaged {
				t.Errorf("SpannerEscapeInfo.IsClosureManaged = %v, want %v", spannerEscape.IsClosureManaged, tt.wantIsClosureManaged)
			}

			if spannerEscape.AutoManagementReason != tt.wantAutoManagementReason {
				t.Errorf("SpannerEscapeInfo.AutoManagementReason = %q, want %q", spannerEscape.AutoManagementReason, tt.wantAutoManagementReason)
			}
		})
	}
}

func TestEscapeAnalyzer_IsSpannerClosurePattern(t *testing.T) {
	// IsSpannerClosurePattern のテスト（ヘルパーメソッド）
	tests := []struct {
		name     string
		code     string
		varName  string
		wantType string
		want     bool
	}{
		{
			name: "ReadWriteTransaction クロージャ検出",
			code: `
package test
import "cloud.google.com/go/spanner"
func testFunction() {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return nil
	})
}`,
			varName:  "txn",
			wantType: ReadWriteTransactionType,
			want:     true,
		},
		{
			name: "通常の変数代入（クロージャなし）",
			code: `
package test
import "cloud.google.com/go/spanner"
func testFunction() {
	txn := client.ReadOnlyTransaction()
	defer txn.Close()
}`,
			varName:  "txn",
			wantType: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// コードをパースしてASTを作成
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("コードのパースに失敗: %v", err)
			}

			// EscapeAnalyzerを作成
			analyzer := NewEscapeAnalyzer()

			// 関数を取得
			var fn *ast.FuncDecl
			for _, decl := range file.Decls {
				if f, ok := decl.(*ast.FuncDecl); ok && f.Name != nil {
					fn = f
					break
				}
			}

			if fn == nil {
				t.Fatal("関数が見つかりません")
			}

			// 変数を検索
			var targetVar *types.Var
			ast.Inspect(fn, func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok && ident.Name == tt.varName {
					targetVar = types.NewVar(ident.Pos(), nil, tt.varName, nil)
					return false
				}
				return true
			})

			// IsSpannerClosurePattern を呼び出し（未実装なので失敗する）
			isPattern, transactionType := analyzer.IsSpannerClosurePattern(targetVar, fn)

			if isPattern != tt.want {
				t.Errorf("IsSpannerClosurePattern() = %v, want %v", isPattern, tt.want)
			}

			if transactionType != tt.wantType {
				t.Errorf("transaction type = %q, want %q", transactionType, tt.wantType)
			}
		})
	}
}

func TestEscapeAnalyzer_SpannerHelperMethods(t *testing.T) {
	// REFACTORフェーズで追加したヘルパーメソッドのテスト
	analyzer := NewEscapeAnalyzer()

	// isSpannerTransactionMethod のテスト
	tests := []struct {
		method string
		want   bool
	}{
		{"ReadWriteTransaction", true},
		{"ReadOnlyTransaction", true},
		{"NewClient", false},
		{"Close", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run("isSpannerTransactionMethod_"+tt.method, func(t *testing.T) {
			got := analyzer.isSpannerTransactionMethod(tt.method)
			if got != tt.want {
				t.Errorf("isSpannerTransactionMethod(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}

	// mapMethodToTransactionType のテスト
	mappingTests := []struct {
		method string
		want   string
	}{
		{"ReadWriteTransaction", ReadWriteTransactionType},
		{"ReadOnlyTransaction", ReadOnlyTransactionType},
		{"InvalidMethod", ""},
		{"", ""},
	}

	for _, tt := range mappingTests {
		t.Run("mapMethodToTransactionType_"+tt.method, func(t *testing.T) {
			got := analyzer.mapMethodToTransactionType(tt.method)
			if got != tt.want {
				t.Errorf("mapMethodToTransactionType(%q) = %q, want %q", tt.method, got, tt.want)
			}
		})
	}
}

func TestEscapeAnalyzer_EdgeCases(t *testing.T) {
	// エッジケースのテスト
	analyzer := NewEscapeAnalyzer()

	// nil パラメータのテスト
	spannerEscape := analyzer.DetectSpannerAutoManagement(nil, nil)
	if spannerEscape != nil {
		t.Error("DetectSpannerAutoManagement(nil, nil) は nil を返すべき")
	}

	// HasSpannerEscapeInfo のテスト
	if analyzer.HasSpannerEscapeInfo(nil) {
		t.Error("HasSpannerEscapeInfo(nil) は false を返すべき")
	}

	// 存在しない変数のテスト
	dummyVar := types.NewVar(0, nil, "dummy", nil)
	if analyzer.HasSpannerEscapeInfo(dummyVar) {
		t.Error("未キャッシュ変数に対してHasSpannerEscapeInfoは false を返すべき")
	}

	// findVariableInClosureParams のエッジケース
	// nilファンクションリテラル
	if analyzer.findVariableInClosureParams(nil, "test") {
		t.Error("findVariableInClosureParams(nil, \"test\") は false を返すべき")
	}

	// 空のパラメータリスト
	funcLit := &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{},
			},
		},
	}
	if analyzer.findVariableInClosureParams(funcLit, "test") {
		t.Error("空のパラメータリストで findVariableInClosureParams は false を返すべき")
	}
}
