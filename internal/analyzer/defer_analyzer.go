package analyzer

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// DeferAnalyzer はdefer文を解析してリソースの適切な解放を検証する
type DeferAnalyzer struct {
	tracker    *ResourceTracker
	scopeStack []*types.Scope
	resources  []ResourceInfo // 検出されたリソース
}

// NewDeferAnalyzer は新しいDeferAnalyzerを作成する
func NewDeferAnalyzer(tracker *ResourceTracker) *DeferAnalyzer {
	return &DeferAnalyzer{
		tracker:    tracker,
		scopeStack: make([]*types.Scope, 0),
		resources:  make([]ResourceInfo, 0),
	}
}

// AnalyzeDefers は関数内のdefer文を解析して診断を生成する（外部からリソースリストを受け取る）
func (da *DeferAnalyzer) AnalyzeDefers(fn *ast.FuncDecl, resources []ResourceInfo) []analysis.Diagnostic {
	if fn == nil || fn.Body == nil {
		return nil
	}

	var diagnostics []analysis.Diagnostic

	// defer文を検索
	defers := da.FindDeferStatements(fn.Body)

	// デバッグ出力を削除（本番では不要）

	// 各リソースについてdefer文の存在を確認
	for _, resource := range resources {
		if resource.IsRequired {
			// デバッグコード削除（本番では不要）

			found := false

			// 位置ベースの精密マッチング
			bestMatchDefer := da.FindBestMatchingDefer(resource, defers)
			if bestMatchDefer != nil && da.ValidateCleanupPattern(resource, bestMatchDefer) {
				found = true
			}

			// 従来の方式による全defer文のチェック（フォールバック）
			if !found {
				for _, deferStmt := range defers {
					if da.ValidateCleanupPattern(resource, deferStmt) {
						found = true
						break
					}
				}
			}

			// defers配列への追加もチェック
			if !found {
				found = da.IsAddedToDeferArray(fn.Body, resource)
			}

			if !found {
				diag := analysis.Diagnostic{
					Pos:     resource.CreationPos,
					End:     resource.CreationPos,
					Message: da.generateDiagnosticMessage(resource),
				}
				diagnostics = append(diagnostics, diag)
			}
		}
	}

	return diagnostics
}

// FindDeferStatements はブロック内のdefer文を再帰的に検索する
func (da *DeferAnalyzer) FindDeferStatements(block *ast.BlockStmt) []*ast.DeferStmt {
	if block == nil {
		return nil
	}

	var defers []*ast.DeferStmt

	// ブロック内を走査
	for _, stmt := range block.List {
		da.collectDeferStatements(stmt, &defers)
	}

	return defers
}

// ValidateCleanupPattern はリソースとdefer文が適切にマッチするかを検証する
func (da *DeferAnalyzer) ValidateCleanupPattern(resource ResourceInfo, deferStmt *ast.DeferStmt) bool {
	if deferStmt == nil || deferStmt.Call == nil {
		return false
	}

	// defer文の呼び出しを解析
	call := deferStmt.Call

	// 新しいisResourceCloseCallロジックを使用してクロージャパターンも検出
	return da.isResourceCloseCall(call.Fun, resource)
}

// FindBestMatchingDefer は位置に基づいてリソースに最適なdefer文を見つける
func (da *DeferAnalyzer) FindBestMatchingDefer(resource ResourceInfo, defers []*ast.DeferStmt) *ast.DeferStmt {
	var bestMatch *ast.DeferStmt
	bestDistance := int(^uint(0) >> 1) // int の最大値

	for _, deferStmt := range defers {
		// defer文がリソースの後に来ているかをチェック
		if deferStmt.Pos() <= resource.CreationPos {
			continue
		}

		// 期待されるクリーンアップメソッドかをチェック
		if !da.IsExpectedCleanupMethod(deferStmt, resource.CleanupMethod) {
			continue
		}

		// 変数名が一致するかチェック
		if !da.HasMatchingVariableName(deferStmt, resource) {
			continue
		}

		// iterator系リソースの場合、変数名完全一致を必須とする
		if da.isIteratorResource(resource) {
			if resource.VariableName == "" || !da.hasExactVariableName(deferStmt, resource.VariableName) {
				continue
			}
		}

		// 距離を計算（近いほど良い）
		distance := int(deferStmt.Pos() - resource.CreationPos)
		if distance < bestDistance {
			bestDistance = distance
			bestMatch = deferStmt
		}
	}

	return bestMatch
}

// hasExactVariableName は変数名の完全一致をチェック
func (da *DeferAnalyzer) hasExactVariableName(deferStmt *ast.DeferStmt, expectedVarName string) bool {
	if deferStmt.Call == nil {
		return false
	}

	if sel, ok := deferStmt.Call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == expectedVarName
		}
	}

	return false
}

// IsExpectedCleanupMethod はdefer文が期待されるクリーンアップメソッドかチェック
func (da *DeferAnalyzer) IsExpectedCleanupMethod(deferStmt *ast.DeferStmt, expectedMethod string) bool {
	if deferStmt.Call == nil {
		return false
	}

	if sel, ok := deferStmt.Call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name == expectedMethod
	}

	return false
}

// HasMatchingVariableName はdefer文の変数名がリソースと一致するかチェック
func (da *DeferAnalyzer) HasMatchingVariableName(deferStmt *ast.DeferStmt, resource ResourceInfo) bool {
	if deferStmt.Call == nil {
		return false
	}

	if sel, ok := deferStmt.Call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			varName := ident.Name

			// 1. 完全一致を最優先
			if resource.VariableName != "" && varName == resource.VariableName {
				return true
			}

			// 2. パターンマッチング
			if da.isValidVariableNamePattern(resource.CreationFunction, varName) {
				return true
			}
		}
	}

	return false
}

// isIteratorResource はリソースがiteratorタイプかどうかを判定
func (da *DeferAnalyzer) isIteratorResource(resource ResourceInfo) bool {
	switch resource.CreationFunction {
	case "Query", "QueryWithOptions", "Read", "ReadWithOptions":
		return true
	default:
		return false
	}
}



// IsAddedToDeferArray はリソースがdefers配列に追加されているかチェック
func (da *DeferAnalyzer) IsAddedToDeferArray(block *ast.BlockStmt, resource ResourceInfo) bool {
	if block == nil || resource.VariableName == "" {
		return false
	}

	// defers = append(defers, resourceVar.Close) パターンを検索
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			// append呼び出しをチェック
			if da.isAppendToDeferArray(assignStmt, resource) {
				found = true
				return false
			}
		}
		return true
	})

	return found
}

// isAppendToDeferArray は代入文がdefers配列への追加かチェック
func (da *DeferAnalyzer) isAppendToDeferArray(assignStmt *ast.AssignStmt, resource ResourceInfo) bool {
	// defers = append(defers, resourceVar.Close) の形式をチェック
	if len(assignStmt.Lhs) != 1 || len(assignStmt.Rhs) != 1 {
		return false
	}

	// 左辺が "defers" 変数かチェック
	if lhsIdent, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
		if lhsIdent.Name != "defers" {
			return false
		}
	} else {
		return false
	}

	// 右辺が append(defers, ...) かチェック
	if callExpr, ok := assignStmt.Rhs[0].(*ast.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*ast.Ident); ok {
			if ident.Name == "append" && len(callExpr.Args) >= 2 {
				// 第一引数が "defers" かチェック
				if firstArg, ok := callExpr.Args[0].(*ast.Ident); ok {
					if firstArg.Name != "defers" {
						return false
					}
				} else {
					return false
				}

				// 第二引数以降でresourceVar.Close を探す
				for i := 1; i < len(callExpr.Args); i++ {
					if da.isResourceCloseCall(callExpr.Args[i], resource) {
						return true
					}
				}
			}
		}
	}

	return false
}

// isResourceCloseCall は式がリソースのCloseメソッド呼び出しかチェック
func (da *DeferAnalyzer) isResourceCloseCall(expr ast.Expr, resource ResourceInfo) bool {
	// パターン1: 直接的なメソッド呼び出し resourceVar.Close
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		return da.isDirectMethodCall(sel, resource)
	}

	// パターン2: クロージャ func() { resourceVar.Close() }
	if funcLit, ok := expr.(*ast.FuncLit); ok {
		return da.isClosureWithResourceClose(funcLit, resource)
	}

	return false
}

// isDirectMethodCall は直接的なメソッド呼び出し resourceVar.Close をチェック
func (da *DeferAnalyzer) isDirectMethodCall(sel *ast.SelectorExpr, resource ResourceInfo) bool {
	// メソッド名がクリーンアップメソッドかチェック
	if sel.Sel.Name != resource.CleanupMethod {
		return false
	}

	// 変数名がリソース変数名と一致するかチェック
	if ident, ok := sel.X.(*ast.Ident); ok {
		return ident.Name == resource.VariableName
	}

	return false
}

// isClosureWithResourceClose はクロージャ内でリソースのCloseが呼ばれているかチェック
func (da *DeferAnalyzer) isClosureWithResourceClose(funcLit *ast.FuncLit, resource ResourceInfo) bool {
	if funcLit == nil || funcLit.Body == nil {
		return false
	}

	// クロージャ内でresourceVar.Close()が呼ばれているかを検索
	found := false
	ast.Inspect(funcLit.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			// メソッド呼び出しをチェック
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if da.isDirectMethodCall(sel, resource) {
					found = true
					return false // 見つかったので走査終了
				}
			}
		case *ast.ExprStmt:
			// 式文内のメソッド呼び出しもチェック
			if call, ok := node.X.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if da.isDirectMethodCall(sel, resource) {
						found = true
						return false // 見つかったので走査終了
					}
				}
			}
		}
		return !found // 見つかるまで継続
	})

	return found
}

// isSameResourceType は2つのリソースが同じ型かチェックする

// isValidVariableNamePattern は生成関数と変数名の妥当性をチェックする
func (da *DeferAnalyzer) isValidVariableNamePattern(creationFunction, varName string) bool {
	validators := map[string]func(string) bool{
		"NewClient":              da.isValidClientVariableName,
		"NewWriter":              da.isValidWriterVariableName,
		"NewReader":              da.isValidReaderVariableName,
		"Query":                  da.isValidQueryVariableName,
		"QueryWithOptions":       da.isValidQueryVariableName,
		"Read":                   da.isValidQueryVariableName,
		"ReadWithOptions":        da.isValidQueryVariableName,
		"ReadWriteTransaction":   da.isValidTransactionVariableName,
		"ReadOnlyTransaction":    da.isValidTransactionVariableName,
	}

	if validator, exists := validators[creationFunction]; exists {
		return validator(varName)
	}
	return false
}

func (da *DeferAnalyzer) isValidClientVariableName(varName string) bool {
	return strings.Contains(varName, "client") || strings.Contains(varName, "Client")
}

func (da *DeferAnalyzer) isValidWriterVariableName(varName string) bool {
	return strings.Contains(varName, "writer") || strings.Contains(varName, "Writer") ||
		varName == "dst" || varName == "w"
}

func (da *DeferAnalyzer) isValidReaderVariableName(varName string) bool {
	return strings.Contains(varName, "reader") || strings.Contains(varName, "Reader") ||
		varName == "src" || varName == "r"
}

func (da *DeferAnalyzer) isValidQueryVariableName(varName string) bool {
	return strings.Contains(varName, "iter") || strings.Contains(varName, "Iter") ||
		strings.Contains(varName, "rows") || strings.Contains(varName, "Rows") ||
		strings.Contains(varName, "result") || strings.Contains(varName, "Result") ||
		varName == "it" || varName == "rs"
}

func (da *DeferAnalyzer) isValidTransactionVariableName(varName string) bool {
	return varName == "tx" || varName == "txn" ||
		strings.Contains(varName, "transaction") || strings.Contains(varName, "Transaction") ||
		strings.Contains(varName, "tx") || strings.Contains(varName, "Tx")
}

// ValidateCleanupOrder はdefer文の順序が適切かを検証する
// Goのdeferはスタック（LIFO）なので、依存関係の逆順で呼び出される
func (da *DeferAnalyzer) ValidateCleanupOrder(block *ast.BlockStmt) bool {
	if block == nil {
		return true
	}

	defers := da.FindDeferStatements(block)
	if len(defers) <= 1 {
		return true // 単一または0個のdeferは常に正しい
	}

	// defer文の順序を検証
	// RowIterator.Stop() → Transaction.Close() → Client.Close() の逆順
	// つまり defer client.Close(), defer txn.Close(), defer iter.Stop() の順
	var deferTypes []string
	for _, deferStmt := range defers {
		resourceType := da.identifyResourceTypeFromDefer(deferStmt)
		deferTypes = append(deferTypes, resourceType)
	}

	// 期待される順序（deferの実行とは逆順）
	return da.isValidDeferOrder(deferTypes)
}

// analyzeFunction は関数内のリソース生成を解析する

// collectDeferStatements は文を再帰的に走査してdefer文を収集する
func (da *DeferAnalyzer) collectDeferStatements(stmt ast.Stmt, defers *[]*ast.DeferStmt) {
	if stmt == nil {
		return
	}

	switch s := stmt.(type) {
	case *ast.DeferStmt:
		*defers = append(*defers, s)
	case *ast.BlockStmt:
		da.collectDefersFromBlockStmt(s, defers)
	case *ast.IfStmt:
		da.collectDefersFromIfStmt(s, defers)
	case *ast.ForStmt:
		da.collectDefersFromForStmt(s, defers)
	case *ast.RangeStmt:
		da.collectDefersFromRangeStmt(s, defers)
	case *ast.SwitchStmt:
		da.collectDefersFromSwitchStmt(s, defers)
	case *ast.TypeSwitchStmt:
		da.collectDefersFromTypeSwitchStmt(s, defers)
	case *ast.SelectStmt:
		da.collectDefersFromSelectStmt(s, defers)
	case *ast.CaseClause:
		da.collectDefersFromCaseClause(s, defers)
	case *ast.CommClause:
		da.collectDefersFromCommClause(s, defers)
	case *ast.ExprStmt:
		da.collectDeferFromExpression(s.X, defers)
	case *ast.AssignStmt:
		da.collectDefersFromAssignStmt(s, defers)
	}
}

func (da *DeferAnalyzer) collectDefersFromBlockStmt(s *ast.BlockStmt, defers *[]*ast.DeferStmt) {
	for _, blockStmt := range s.List {
		da.collectDeferStatements(blockStmt, defers)
	}
}

func (da *DeferAnalyzer) collectDefersFromIfStmt(s *ast.IfStmt, defers *[]*ast.DeferStmt) {
	if s.Body != nil {
		da.collectDeferStatements(s.Body, defers)
	}
	if s.Else != nil {
		da.collectDeferStatements(s.Else, defers)
	}
}

func (da *DeferAnalyzer) collectDefersFromForStmt(s *ast.ForStmt, defers *[]*ast.DeferStmt) {
	if s.Body != nil {
		da.collectDeferStatements(s.Body, defers)
	}
}

func (da *DeferAnalyzer) collectDefersFromRangeStmt(s *ast.RangeStmt, defers *[]*ast.DeferStmt) {
	if s.Body != nil {
		da.collectDeferStatements(s.Body, defers)
	}
}

func (da *DeferAnalyzer) collectDefersFromSwitchStmt(s *ast.SwitchStmt, defers *[]*ast.DeferStmt) {
	if s.Body != nil {
		da.collectDeferStatements(s.Body, defers)
	}
}

func (da *DeferAnalyzer) collectDefersFromTypeSwitchStmt(s *ast.TypeSwitchStmt, defers *[]*ast.DeferStmt) {
	if s.Body != nil {
		da.collectDeferStatements(s.Body, defers)
	}
}

func (da *DeferAnalyzer) collectDefersFromSelectStmt(s *ast.SelectStmt, defers *[]*ast.DeferStmt) {
	if s.Body != nil {
		da.collectDeferStatements(s.Body, defers)
	}
}

func (da *DeferAnalyzer) collectDefersFromCaseClause(s *ast.CaseClause, defers *[]*ast.DeferStmt) {
	for _, caseStmt := range s.Body {
		da.collectDeferStatements(caseStmt, defers)
	}
}

func (da *DeferAnalyzer) collectDefersFromCommClause(s *ast.CommClause, defers *[]*ast.DeferStmt) {
	for _, commStmt := range s.Body {
		da.collectDeferStatements(commStmt, defers)
	}
}

func (da *DeferAnalyzer) collectDefersFromAssignStmt(s *ast.AssignStmt, defers *[]*ast.DeferStmt) {
	for _, rhs := range s.Rhs {
		da.collectDeferFromExpression(rhs, defers)
	}
}

// collectDeferFromExpression は式の中のクロージャからdefer文を収集する
func (da *DeferAnalyzer) collectDeferFromExpression(expr ast.Expr, defers *[]*ast.DeferStmt) {
	switch e := expr.(type) {
	case *ast.CallExpr:
		// 関数呼び出しの引数にクロージャがある場合
		for _, arg := range e.Args {
			if funcLit, ok := arg.(*ast.FuncLit); ok {
				if funcLit.Body != nil {
					da.collectDeferStatements(funcLit.Body, defers)
				}
			}
		}
	}
}

// identifyResourceTypeFromDefer はdefer文からリソースタイプを特定する
func (da *DeferAnalyzer) identifyResourceTypeFromDefer(deferStmt *ast.DeferStmt) string {
	if deferStmt == nil || deferStmt.Call == nil {
		return ""
	}

	// defer文の呼び出しからリソースタイプを推定
	call := deferStmt.Call
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		methodName := sel.Sel.Name

		// メソッド名からリソースタイプを推定
		switch methodName {
		case "Close":
			return "client" // 汎用的なクライアント
		case "Stop":
			return "iterator" // RowIteratorなど
		case "Cleanup":
			return "resource" // 汎用的なリソース
		}
	}

	return "unknown"
}

// extractMethodFromDefer はdefer文からメソッド名を抽出する
func (da *DeferAnalyzer) extractMethodFromDefer(deferStmt *ast.DeferStmt) string {
	if deferStmt == nil || deferStmt.Call == nil {
		return ""
	}

	call := deferStmt.Call
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name
	}

	return ""
}

// isValidDeferOrder はdefer文の順序が適切かを判定する
func (da *DeferAnalyzer) isValidDeferOrder(_ []string) bool {
	// 簡単な実装：現在は常にtrueを返す
	// 実際の実装では、依存関係を考慮した順序チェックを行う

	// 例：Client -> Transaction -> Iterator の逆順（defer実行順序）
	// defer client.Close() -> defer txn.Close() -> defer iter.Stop()

	return true // 現在は順序チェックをスキップ
}

// generateDiagnosticMessage はリソースに対する診断メッセージを生成する
func (da *DeferAnalyzer) generateDiagnosticMessage(resource ResourceInfo) string {
	varName := resource.VariableName
	if varName == "" {
		varName = "リソース"
	}
	method := resource.CleanupMethod

	return "GCP リソース '" + varName + "' の解放処理 (" + method + ") が見つかりません"
}

// DeferInfo はdefer文に関する情報を保持する
type DeferInfo struct {
	DeferStmt    *ast.DeferStmt
	ResourceType string
	Method       string
	ScopeDepth   int  // スコープの深さ
	IsValid      bool // defer文が有効かどうか
}

// AnalyzeDefersPrecision は改良されたdefer文の精密解析を実行する
func (da *DeferAnalyzer) AnalyzeDefersPrecision(block *ast.BlockStmt) []DeferInfo {
	if block == nil {
		return nil
	}

	var deferInfos []DeferInfo
	da.analyzeDeferStatementsWithScope(block, 0, &deferInfos)
	return deferInfos
}

// analyzeDeferStatementsWithScope はスコープを考慮したdefer文解析
func (da *DeferAnalyzer) analyzeDeferStatementsWithScope(block *ast.BlockStmt, scopeDepth int, deferInfos *[]DeferInfo) {
	if block == nil {
		return
	}

	// ブロック内の各ステートメントを解析
	for _, stmt := range block.List {
		da.processStatementForDeferPrecision(stmt, scopeDepth, deferInfos)
	}
}

// processStatementForDeferPrecision はステートメントを精密にdefer解析
func (da *DeferAnalyzer) processStatementForDeferPrecision(stmt ast.Stmt, scopeDepth int, deferInfos *[]DeferInfo) {
	switch s := stmt.(type) {
	case *ast.DeferStmt:
		// defer文を解析
		deferInfo := DeferInfo{
			DeferStmt:    s,
			ResourceType: da.identifyResourceTypeFromDefer(s),
			Method:       da.extractMethodFromDefer(s),
			ScopeDepth:   scopeDepth,
			IsValid:      da.validateDeferInScope(s, scopeDepth),
		}
		*deferInfos = append(*deferInfos, deferInfo)

	case *ast.BlockStmt:
		// ネストしたブロック
		da.analyzeDeferStatementsWithScope(s, scopeDepth+1, deferInfos)

	case *ast.IfStmt:
		if s.Body != nil {
			da.analyzeDeferStatementsWithScope(s.Body, scopeDepth+1, deferInfos)
		}
		if s.Else != nil {
			da.processStatementForDeferPrecision(s.Else, scopeDepth+1, deferInfos)
		}

	case *ast.ForStmt:
		if s.Body != nil {
			da.analyzeDeferStatementsWithScope(s.Body, scopeDepth+1, deferInfos)
		}

	case *ast.RangeStmt:
		if s.Body != nil {
			da.analyzeDeferStatementsWithScope(s.Body, scopeDepth+1, deferInfos)
		}

	case *ast.GoStmt:
		// goroutine内のdeferも解析
		if call := s.Call; call != nil {
			if funLit, ok := call.Fun.(*ast.FuncLit); ok && funLit.Body != nil {
				da.analyzeDeferStatementsWithScope(funLit.Body, scopeDepth+1, deferInfos)
			}
		}
	}
}

// validateDeferInScope はスコープ内でのdefer文の妥当性を検証
func (da *DeferAnalyzer) validateDeferInScope(deferStmt *ast.DeferStmt, scopeDepth int) bool {
	if deferStmt == nil || deferStmt.Call == nil {
		return false
	}

	// defer文のコール先が有効かチェック
	call := deferStmt.Call

	// 関数呼び出しの場合（cancel()など）
	if ident, ok := call.Fun.(*ast.Ident); ok {
		// 変数名の妥当性をチェック（簡易版）
		varName := ident.Name
		return da.isValidCancelVariableName(varName)
	}

	// メソッド呼び出しの場合（client.Close()など）
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		methodName := sel.Sel.Name
		return da.isValidCleanupMethodName(methodName)
	}

	return true
}

// isValidCancelVariableName はcancel変数名が妥当かチェック
func (da *DeferAnalyzer) isValidCancelVariableName(varName string) bool {
	validCancelNames := []string{
		"cancel", "Cancel",
		"timeoutCancel", "deadlineCancel",
		"ctx1Cancel", "ctx2Cancel", "ctx3Cancel", "ctx4Cancel",
		"cancel1", "cancel2", "cancel3", "cancel4",
	}

	for _, validName := range validCancelNames {
		if varName == validName {
			return true
		}
	}

	// パターンマッチング（*cancel, *Cancel）
	if strings.Contains(varName, "cancel") || strings.Contains(varName, "Cancel") {
		return true
	}

	return false
}

// isValidCleanupMethodName はクリーンアップメソッド名が妥当かチェック
func (da *DeferAnalyzer) isValidCleanupMethodName(methodName string) bool {
	validMethods := []string{"Close", "Stop", "Shutdown", "Cleanup"}

	for _, validMethod := range validMethods {
		if methodName == validMethod {
			return true
		}
	}

	return false
}

// ValidateDeferScope はdefer文のスコープ妥当性を検証
func (da *DeferAnalyzer) ValidateDeferScope(block *ast.BlockStmt) bool {
	if block == nil {
		return true
	}

	// 精密解析を実行
	deferInfos := da.AnalyzeDefersPrecision(block)

	// 全てのdefer文が有効かチェック
	for _, deferInfo := range deferInfos {
		if !deferInfo.IsValid {
			return false
		}
	}

	return true
}
