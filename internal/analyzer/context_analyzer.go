package analyzer

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// ContextAnalyzer はcontext.WithCancel/WithTimeout検出とキャンセレーション検証を行う
type ContextAnalyzer struct {
	contextVars   map[*types.Var]*ContextInfo
	cancelVarNames map[string]*ContextInfo // 変数名 -> ContextInfo のマッピング
	scopeStack    []map[string]*ContextInfo // スコープ境界を跨ぐ変数名解決用
}

// NewContextAnalyzer は新しいContextAnalyzerを作成する
func NewContextAnalyzer() *ContextAnalyzer {
	return &ContextAnalyzer{
		contextVars:    make(map[*types.Var]*ContextInfo),
		cancelVarNames: make(map[string]*ContextInfo),
		scopeStack:     make([]map[string]*ContextInfo, 0),
	}
}

// TrackContextCreation はcontext生成関数を解析してキャンセル関数を追跡する
func (ca *ContextAnalyzer) TrackContextCreation(call *ast.CallExpr, typeInfo *types.Info) error {
	if ca == nil || typeInfo == nil {
		return nil
	}

	// セレクタ式（context.WithCancel等）かどうか確認
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	// contextパッケージの呼び出しかどうか確認
	if ident, ok := sel.X.(*ast.Ident); ok {
		if obj := typeInfo.Uses[ident]; obj != nil {
			if pkg, ok := obj.(*types.PkgName); ok {
				if pkg.Imported().Path() == "context" {
					funcName := sel.Sel.Name
					
					// キャンセル関数を返すcontext関数かどうか確認
					if ca.IsContextWithCancel(funcName) {
						// ContextInfoを作成
						contextInfo := &ContextInfo{
							CreationPos: call.Pos(),
							IsDeferred:  false, // defer状態は後で確認
						}

						// 簡易的にcontextVarsに追加（実際の変数追跡は簡略化）
						dummyVar := &types.Var{}
						ca.contextVars[dummyVar] = contextInfo
						contextInfo.CancelFunc = dummyVar
					}
				}
			}
		}
	}

	return nil
}

// FindMissingCancels はanalysis.Passを使用してキャンセル漏れを検出する
func (ca *ContextAnalyzer) FindMissingCancels(pass *analysis.Pass) []analysis.Diagnostic {
	if pass == nil || len(pass.Files) == 0 {
		return nil
	}

	var diagnostics []analysis.Diagnostic

	// 各ファイルを解析
	for _, file := range pass.Files {
		// 改良された解析を実行
		ca.AnalyzeContextUsage(file, pass.TypesInfo)
		
		// 各contextについてdefer文の存在を確認
		for _, contextInfo := range ca.contextVars {
			if !contextInfo.IsDeferred {
				diag := analysis.Diagnostic{
					Pos:     contextInfo.CreationPos,
					End:     contextInfo.CreationPos,
					Message: "context cancel function should be called with defer",
				}
				diagnostics = append(diagnostics, diag)
			}
		}
		
		// 次のファイル用にリセット
		ca.contextVars = make(map[*types.Var]*ContextInfo)
		ca.cancelVarNames = make(map[string]*ContextInfo)
		ca.scopeStack = make([]map[string]*ContextInfo, 0)
	}

	return diagnostics
}

// IsContextWithCancel は関数名がキャンセル関数を返すcontext関数かどうかを判定する
func (ca *ContextAnalyzer) IsContextWithCancel(funcName string) bool {
	cancelFunctions := []string{
		"WithCancel",
		"WithTimeout",
		"WithDeadline",
	}

	for _, cancelFunc := range cancelFunctions {
		if funcName == cancelFunc {
			return true
		}
	}
	
	return false
}

// GetTrackedContextVars は追跡中のcontext変数一覧を取得する（テスト用）
func (ca *ContextAnalyzer) GetTrackedContextVars() []ContextInfo {
	var contexts []ContextInfo
	for _, info := range ca.contextVars {
		contexts = append(contexts, *info)
	}
	return contexts
}

// analyzeContextCreations はファイル内のcontext生成を解析する
func (ca *ContextAnalyzer) analyzeContextCreations(file *ast.File, typeInfo *types.Info) {
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			ca.TrackContextCreation(call, typeInfo)
		}
		return true
	})
}


// analyzeDeferStatementsWithFileScope はより精密なスコープ解析でdefer文を確認する
func (ca *ContextAnalyzer) analyzeDeferStatementsWithFileScope(file *ast.File) {
	// 関数ごとに解析
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			ca.analyzeFunctionForDefers(fn)
		}
	}
}

// analyzeFunctionForDefers は関数内のcontext生成とdefer文を対応付ける
func (ca *ContextAnalyzer) analyzeFunctionForDefers(fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	// この関数で生成されたcontextを追跡
	functionContexts := make(map[string]*ContextInfo)

	// 関数内のcontext生成を検索
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			if len(assignStmt.Lhs) >= 2 && len(assignStmt.Rhs) == 1 {
				if call, ok := assignStmt.Rhs[0].(*ast.CallExpr); ok {
					if ca.isContextWithCancelCall(call) {
						// cancel関数の変数名を取得
						if len(assignStmt.Lhs) >= 2 {
							if cancelIdent, ok := assignStmt.Lhs[1].(*ast.Ident); ok {
								cancelVarName := cancelIdent.Name
								// contextVarsから対応するcontextを検索
								for _, contextInfo := range ca.contextVars {
									functionContexts[cancelVarName] = contextInfo
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	// defer文を検索してキャンセル関数の名前と対応付け
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if deferStmt, ok := n.(*ast.DeferStmt); ok {
			if call := deferStmt.Call; call != nil {
				if ident, ok := call.Fun.(*ast.Ident); ok {
					cancelVarName := ident.Name
					if contextInfo, exists := functionContexts[cancelVarName]; exists {
						contextInfo.IsDeferred = true
					}
				}
			}
		}
		return true
	})
}

// isContextWithCancelCall は呼び出しがcontext.WithCancel系の関数かどうかを判定する
func (ca *ContextAnalyzer) isContextWithCancelCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		funcName := sel.Sel.Name
		return ca.IsContextWithCancel(funcName)
	}
	return false
}

// AnalyzeContextUsage は改良されたcontext使用解析を実行する
func (ca *ContextAnalyzer) AnalyzeContextUsage(file *ast.File, typeInfo *types.Info) error {
	if file == nil || typeInfo == nil {
		return nil
	}

	// ファイル全体を解析してcontext生成とdefer文を精密に追跡
	return ca.analyzeWithImprovedTracking(file, typeInfo)
}

// analyzeWithImprovedTracking は改良された追跡システムで解析を実行
func (ca *ContextAnalyzer) analyzeWithImprovedTracking(file *ast.File, typeInfo *types.Info) error {
	// ファイルレベルのスコープを初期化
	ca.pushScope()
	defer ca.popScope()

	// 関数ごとに解析
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			ca.analyzeFunctionWithImprovedTracking(fn, typeInfo)
		}
	}

	return nil
}

// analyzeFunctionWithImprovedTracking は関数を改良された追跡機能で解析
func (ca *ContextAnalyzer) analyzeFunctionWithImprovedTracking(fn *ast.FuncDecl, typeInfo *types.Info) {
	if fn.Body == nil {
		return
	}

	// 関数スコープを開始
	ca.pushScope()
	defer ca.popScope()

	// 関数内を解析
	ca.analyzeFunctionBodyWithTracking(fn.Body, typeInfo)
}

// analyzeFunctionBodyWithTracking は関数本体を詳細に追跡して解析
func (ca *ContextAnalyzer) analyzeFunctionBodyWithTracking(body *ast.BlockStmt, typeInfo *types.Info) {
	// 関数本体のステートメントを順次処理
	for _, stmt := range body.List {
		ca.processStatementWithTracking(stmt, typeInfo)
	}
}

// processStatementWithTracking は個々のステートメントを処理
func (ca *ContextAnalyzer) processStatementWithTracking(stmt ast.Stmt, typeInfo *types.Info) {
	switch node := stmt.(type) {
	case *ast.AssignStmt:
		ca.handleImprovedAssignment(node, typeInfo)
	case *ast.DeferStmt:
		ca.handleImprovedDefer(node)
	case *ast.IfStmt:
		ca.processIfStatementWithTracking(node, typeInfo)
	case *ast.BlockStmt:
		ca.processBlockStatementWithTracking(node, typeInfo)
	case *ast.GoStmt:
		ca.processGoStatementWithTracking(node, typeInfo)
	default:
		// その他のステートメント内にネストした構造があるかチェック
		ast.Inspect(stmt, func(n ast.Node) bool {
			switch nested := n.(type) {
			case *ast.FuncLit:
				ca.analyzeFunctionWithImprovedTracking(&ast.FuncDecl{Body: nested.Body}, typeInfo)
				return false
			case *ast.AssignStmt:
				ca.handleImprovedAssignment(nested, typeInfo)
			case *ast.DeferStmt:
				ca.handleImprovedDefer(nested)
			}
			return true
		})
	}
}

// processIfStatementWithTracking はif文を処理
func (ca *ContextAnalyzer) processIfStatementWithTracking(ifStmt *ast.IfStmt, typeInfo *types.Info) {
	if ifStmt.Init != nil {
		ca.processStatementWithTracking(ifStmt.Init, typeInfo)
	}
	if ifStmt.Body != nil {
		ca.processBlockStatementWithTracking(ifStmt.Body, typeInfo)
	}
	if ifStmt.Else != nil {
		ca.processStatementWithTracking(ifStmt.Else, typeInfo)
	}
}

// processBlockStatementWithTracking はブロック文を処理
func (ca *ContextAnalyzer) processBlockStatementWithTracking(block *ast.BlockStmt, typeInfo *types.Info) {
	ca.pushScope()
	defer ca.popScope()
	ca.analyzeFunctionBodyWithTracking(block, typeInfo)
}

// processGoStatementWithTracking はgoroutine文を処理
func (ca *ContextAnalyzer) processGoStatementWithTracking(goStmt *ast.GoStmt, typeInfo *types.Info) {
	if call := goStmt.Call; call != nil {
		if funLit, ok := call.Fun.(*ast.FuncLit); ok {
			ca.analyzeFunctionWithImprovedTracking(&ast.FuncDecl{Body: funLit.Body}, typeInfo)
		}
	}
}

// handleImprovedAssignment は改良された代入文解析
func (ca *ContextAnalyzer) handleImprovedAssignment(assign *ast.AssignStmt, typeInfo *types.Info) {
	_ = typeInfo // 未使用警告回避
	
	if len(assign.Rhs) != 1 {
		return
	}

	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}

	// context.WithCancel系の呼び出しかチェック（簡易実装）
	if !ca.isSimpleContextCall(call) {
		return
	}

	// 複数戻り値代入での変数名追跡精度向上
	if len(assign.Lhs) >= 2 {
		// cancel関数（第2戻り値）
		if cancelIdent, ok := assign.Lhs[1].(*ast.Ident); ok {
			cancelVarName := cancelIdent.Name
			
			// ContextInfoを作成
			contextInfo := &ContextInfo{
				CreationPos: call.Pos(),
				IsDeferred:  false,
			}

			// 現在のスコープに変数名を登録
			ca.registerCancelVar(cancelVarName, contextInfo)

			// 既存の仕組みにも追加
			dummyVar := &types.Var{}
			ca.contextVars[dummyVar] = contextInfo
			contextInfo.CancelFunc = dummyVar
		}
	}
}

// isSimpleContextCall は簡易版のcontext関数判定
func (ca *ContextAnalyzer) isSimpleContextCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == "context" && ca.IsContextWithCancel(sel.Sel.Name)
		}
	}
	return false
}

// handleImprovedDefer は改良されたdefer文解析
func (ca *ContextAnalyzer) handleImprovedDefer(defer_stmt *ast.DeferStmt) {
	if defer_stmt.Call == nil {
		return
	}

	// defer call() パターンの識別
	if ident, ok := defer_stmt.Call.Fun.(*ast.Ident); ok {
		cancelVarName := ident.Name
		
		// スコープ境界を跨ぐ変数名解決
		if contextInfo := ca.resolveCancelVar(cancelVarName); contextInfo != nil {
			contextInfo.IsDeferred = true
		}
	}
}

// pushScope は新しいスコープを開始する
func (ca *ContextAnalyzer) pushScope() {
	newScope := make(map[string]*ContextInfo)
	ca.scopeStack = append(ca.scopeStack, newScope)
}

// popScope は現在のスコープを終了する
func (ca *ContextAnalyzer) popScope() {
	if len(ca.scopeStack) > 0 {
		ca.scopeStack = ca.scopeStack[:len(ca.scopeStack)-1]
	}
}

// registerCancelVar は変数名をcurrentスコープに登録する
func (ca *ContextAnalyzer) registerCancelVar(name string, contextInfo *ContextInfo) {
	if len(ca.scopeStack) > 0 {
		currentScope := ca.scopeStack[len(ca.scopeStack)-1]
		currentScope[name] = contextInfo
	}
	// グローバルマップにも追加
	ca.cancelVarNames[name] = contextInfo
}

// resolveCancelVar はスコープ境界を跨いで変数名を解決する
func (ca *ContextAnalyzer) resolveCancelVar(name string) *ContextInfo {
	// 現在のスコープから上位スコープへ向かって検索
	for i := len(ca.scopeStack) - 1; i >= 0; i-- {
		if contextInfo, exists := ca.scopeStack[i][name]; exists {
			return contextInfo
		}
	}
	
	// グローバルマップからも検索
	return ca.cancelVarNames[name]
}

