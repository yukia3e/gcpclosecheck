package analyzer

import (
	"go/ast"
	"go/types"
)

// EscapeAnalyzer はリソースの逃げパス（戻り値、フィールド代入）を解析する
type EscapeAnalyzer struct {
	escapeInfo map[*types.Var]*EscapeInfo
}

// NewEscapeAnalyzer は新しいEscapeAnalyzerを作成する
func NewEscapeAnalyzer() *EscapeAnalyzer {
	return &EscapeAnalyzer{
		escapeInfo: make(map[*types.Var]*EscapeInfo),
	}
}

// AnalyzeEscape は変数のエスケープパターンを解析する
func (ea *EscapeAnalyzer) AnalyzeEscape(variable *types.Var, fn *ast.FuncDecl) EscapeInfo {
	if variable == nil || fn == nil {
		return EscapeInfo{}
	}

	escapeInfo := EscapeInfo{
		IsReturned:      ea.IsReturnedValue(variable, fn),
		IsFieldAssigned: ea.IsFieldAssigned(variable, fn),
	}

	// エスケープ理由を設定
	if escapeInfo.IsReturned {
		escapeInfo.EscapeReason = "returned from function"
	} else if escapeInfo.IsFieldAssigned {
		escapeInfo.EscapeReason = "assigned to struct field"
	}

	// 結果をキャッシュ
	ea.escapeInfo[variable] = &escapeInfo

	return escapeInfo
}

// IsReturnedValue は変数が関数の戻り値として返されるかどうかを判定する
func (ea *EscapeAnalyzer) IsReturnedValue(variable *types.Var, fn *ast.FuncDecl) bool {
	if variable == nil || fn == nil || fn.Body == nil {
		return false
	}

	varName := variable.Name()

	// 関数内のreturn文を検索
	var isReturned bool
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if retStmt, ok := n.(*ast.ReturnStmt); ok {
			// return文の式を確認
			for _, expr := range retStmt.Results {
				if ident, ok := expr.(*ast.Ident); ok {
					if ident.Name == varName {
						isReturned = true
						return false // 見つかったので走査終了
					}
				}
			}
		}
		return true
	})

	return isReturned
}

// IsFieldAssigned は変数が構造体のフィールドに代入されるかどうかを判定する
func (ea *EscapeAnalyzer) IsFieldAssigned(variable *types.Var, fn *ast.FuncDecl) bool {
	if variable == nil || fn == nil || fn.Body == nil {
		return false
	}

	varName := variable.Name()

	// 関数内の代入文を検索
	var isAssigned bool
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			// 右辺に対象変数があるかチェック
			for _, rhs := range assignStmt.Rhs {
				if ident, ok := rhs.(*ast.Ident); ok && ident.Name == varName {
					// 左辺がフィールドアクセスかチェック
					for _, lhs := range assignStmt.Lhs {
						if ea.isSelectorExpr(lhs) {
							isAssigned = true
							return false // 見つかったので走査終了
						}
					}
				}
			}
		}
		return true
	})

	return isAssigned
}

// ShouldSkipResource はリソースをスキップすべきかどうかを判定する
func (ea *EscapeAnalyzer) ShouldSkipResource(resource ResourceInfo, escape EscapeInfo) (bool, string) {
	// RowIteratorは特別扱い：戻り値として返されても関数内で処理すべき
	if resource.CreationFunction == "Query" || resource.CreationFunction == "Read" {
		// IteratorやReader系は基本的に関数内で処理
		return false, ""
	}

	// 戻り値として返される場合はスキップ
	if escape.IsReturned {
		return true, escape.EscapeReason
	}

	// フィールドに代入される場合はスキップ
	if escape.IsFieldAssigned {
		return true, escape.EscapeReason
	}

	// その他の場合はスキップしない
	return false, ""
}

// isSelectorExpr は式がセレクタ式（フィールドアクセス）かどうかを判定する
func (ea *EscapeAnalyzer) isSelectorExpr(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.SelectorExpr:
		return true
	default:
		return false
	}
}

// DetectSpannerAutoManagement は Spannerトランザクションの自動管理パターンを検出する
func (ea *EscapeAnalyzer) DetectSpannerAutoManagement(variable *types.Var, fn *ast.FuncDecl) *SpannerEscapeInfo {
	if variable == nil || fn == nil {
		return nil
	}

	// クロージャパターンを検出
	isPattern, transactionType := ea.IsSpannerClosurePattern(variable, fn)
	if !isPattern {
		return nil
	}

	// 自動管理のSpannerEscapeInfoを作成
	reason := transactionType + "クロージャ内で自動管理"
	return NewSpannerEscapeInfo(transactionType, true, reason)
}

// IsSpannerClosurePattern は変数がSpannerクロージャパターンで使用されているかを検出する
func (ea *EscapeAnalyzer) IsSpannerClosurePattern(variable *types.Var, fn *ast.FuncDecl) (bool, string) {
	if variable == nil || fn == nil || fn.Body == nil {
		return false, ""
	}

	varName := variable.Name()

	// 関数内でReadWriteTransaction/ReadOnlyTransactionクロージャを検索
	var foundPattern bool
	var transactionType string

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		// 関数呼び出しを探す
		if callExpr, ok := n.(*ast.CallExpr); ok {
			// SelectorExpr（method call）をチェック
			if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				methodName := sel.Sel.Name

				// ReadWriteTransaction/ReadOnlyTransaction メソッド呼び出しをチェック
				if ea.isSpannerTransactionMethod(methodName) {
					// 引数にクロージャがあるかチェック
					for _, arg := range callExpr.Args {
						if funcLit, ok := arg.(*ast.FuncLit); ok {
							// クロージャのパラメータで対象変数名をチェック
							if ea.findVariableInClosureParams(funcLit, varName) {
								foundPattern = true
								transactionType = ea.mapMethodToTransactionType(methodName)
								return false // 見つかったので走査終了
							}
						}
					}
				}
			}
		}
		return !foundPattern // パターンが見つかるまで続行
	})

	return foundPattern, transactionType
}

// isSpannerTransactionMethod はメソッドがSpannerトランザクションメソッドかどうかを判定する
func (ea *EscapeAnalyzer) isSpannerTransactionMethod(methodName string) bool {
	return methodName == "ReadWriteTransaction" || methodName == "ReadOnlyTransaction"
}

// findVariableInClosureParams はクロージャのパラメータに指定した変数名があるかを検索する
func (ea *EscapeAnalyzer) findVariableInClosureParams(funcLit *ast.FuncLit, varName string) bool {
	if funcLit == nil || funcLit.Type == nil || funcLit.Type.Params == nil {
		return false
	}

	for _, param := range funcLit.Type.Params.List {
		if param == nil {
			continue
		}
		for _, name := range param.Names {
			if name != nil && name.Name == varName {
				return true
			}
		}
	}
	return false
}

// mapMethodToTransactionType はメソッド名をトランザクション種別にマッピングする
func (ea *EscapeAnalyzer) mapMethodToTransactionType(methodName string) string {
	switch methodName {
	case "ReadWriteTransaction":
		return ReadWriteTransactionType
	case "ReadOnlyTransaction":
		return ReadOnlyTransactionType
	default:
		return ""
	}
}

// HasSpannerEscapeInfo はSpannerエスケープ情報がキャッシュされているかを確認する
func (ea *EscapeAnalyzer) HasSpannerEscapeInfo(variable *types.Var) bool {
	if variable == nil {
		return false
	}
	_, exists := ea.escapeInfo[variable]
	return exists
}
