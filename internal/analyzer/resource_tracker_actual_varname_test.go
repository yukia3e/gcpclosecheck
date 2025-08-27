package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestExtractActualVariableName_SpannerClosure(t *testing.T) {
	// KultureJPと同じパターンのコードを準備
	src := `package test
	
import (
	"context"
	"cloud.google.com/go/spanner"
)

func testFunction(c *spanner.Client) {
	ctx := context.Background()
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, transaction *spanner.ReadWriteTransaction) error {
		return nil
	})
	_ = err
}
`

	// ASTを解析
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 型情報を設定
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	// ResourceTrackerを作成
	rt := &ResourceTracker{
		typeInfo: info,
		variables: make(map[*types.Var]*ResourceInfo),
	}

	// ReadWriteTransactionの呼び出しを探す
	var callExpr *ast.CallExpr
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "ReadWriteTransaction" {
					callExpr = call
					return false
				}
			}
		}
		return true
	})

	if callExpr == nil {
		t.Fatal("ReadWriteTransaction call not found")
	}

	// 実際の変数名を抽出
	varName := rt.extractActualVariableName(callExpr)
	
	// 期待値は "transaction" （KultureJPコードと同じ）
	expected := "transaction"
	if varName != expected {
		t.Errorf("Expected variable name '%s', got '%s'", expected, varName)
	}
}

func TestExtractFromClosureParameters_ValidPattern(t *testing.T) {
	// 具体的なクロージャパターンをテスト
	src := `package test
func test() {
	readWriteTransaction(ctx, func(ctx context.Context, myTransaction *Transaction) error {
		return nil
	})
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	rt := &ResourceTracker{}

	// 関数呼び出しを探す
	var callExpr *ast.CallExpr
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			callExpr = call
			return false
		}
		return true
	})

	if callExpr == nil {
		t.Fatal("Function call not found")
	}

	// クロージャパラメータから変数名を抽出
	varName := rt.extractFromClosureParameters(callExpr)
	
	expected := "myTransaction"
	if varName != expected {
		t.Errorf("Expected variable name '%s', got '%s'", expected, varName)
	}
}

func TestExtractFromClosureParameters_InvalidPattern(t *testing.T) {
	// クロージャではない引数パターン
	src := `package test
func test() {
	someFunction("string arg", 123)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	rt := &ResourceTracker{}

	var callExpr *ast.CallExpr
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			callExpr = call
			return false
		}
		return true
	})

	if callExpr == nil {
		t.Fatal("Function call not found")
	}

	// クロージャパラメータから変数名を抽出（空文字列が期待される）
	varName := rt.extractFromClosureParameters(callExpr)
	
	if varName != "" {
		t.Errorf("Expected empty string, got '%s'", varName)
	}
}