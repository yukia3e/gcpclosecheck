package analyzer

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// ResourceTracker はGCPリソースの生成を追跡する
type ResourceTracker struct {
	typeInfo   *types.Info
	ruleEngine *ServiceRuleEngine
	variables  map[*types.Var]*ResourceInfo
}

// NewResourceTracker は新しいResourceTrackerを作成する
func NewResourceTracker(typeInfo *types.Info, ruleEngine *ServiceRuleEngine) *ResourceTracker {
	return &ResourceTracker{
		typeInfo:   typeInfo,
		ruleEngine: ruleEngine,
		variables:  make(map[*types.Var]*ResourceInfo),
	}
}

// TrackCall は関数呼び出しを解析してGCPリソース生成を追跡する
func (rt *ResourceTracker) TrackCall(call *ast.CallExpr) error {
	if rt == nil || rt.typeInfo == nil || rt.ruleEngine == nil {
		return nil // 初期化されていない場合はスキップ
	}

	// 呼び出し式の型情報を取得
	if rt.typeInfo.Types == nil {
		return nil // 型情報がない場合はスキップ
	}

	// 関数呼び出しの識別子を取得
	funcIdent := rt.extractFunctionIdent(call)
	if funcIdent == nil {
		return nil
	}

	// パッケージ情報を取得
	packagePath := rt.extractPackagePath(call, funcIdent)
	if packagePath == "" {
		return nil
	}

	// GCPパッケージかどうか確認
	isGCP, serviceName := rt.GetPackageInfo(packagePath)
	if !isGCP {
		return nil
	}

	// サービスルールを取得
	serviceRule := rt.ruleEngine.GetServiceRule(serviceName)
	if serviceRule == nil {
		return nil
	}

	// 生成関数かどうか確認
	funcName := funcIdent.Name
	if !rt.isCreationFunction(serviceRule, funcName) {
		return nil
	}

	// リソース情報を作成・記録
	resourceInfo := rt.createResourceInfo(call, serviceName, serviceRule)
	if resourceInfo != nil {
		// 変数への代入を追跡（簡易実装）
		rt.trackVariableAssignment(call, resourceInfo)
	}

	return nil
}

// FindResourceCreation はanalysis.Passを使用してリソース生成を検出する
func (rt *ResourceTracker) FindResourceCreation(pass *analysis.Pass) []ResourceInfo {
	if pass == nil || len(pass.Files) == 0 {
		return nil
	}

	var resources []ResourceInfo

	// 各ファイルを走査
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			// 代入文を検索してリソース生成を検出
			if assignStmt, ok := n.(*ast.AssignStmt); ok {
				rt.trackAssignmentStatement(assignStmt, pass)
			}
			return true
		})
	}

	// 追跡されたリソースを返す
	for _, info := range rt.variables {
		resources = append(resources, *info)
	}

	return resources
}

// IsResourceType は型がGCPリソース型かどうかを判定する
func (rt *ResourceTracker) IsResourceType(typ types.Type) (bool, string) {
	if typ == nil {
		return false, ""
	}

	// 型名を文字列として取得
	typeName := typ.String()
	return rt.IsResourceTypeByName(typeName)
}

// IsResourceTypeByName は型名からGCPリソース型かどうかを判定する
func (rt *ResourceTracker) IsResourceTypeByName(typeName string) (bool, string) {
	// GCPパッケージのパターンを確認
	gcpPackages := []struct {
		pattern string
		service string
	}{
		{"*spanner.", "spanner"},
		{"*storage.", "storage"},
		{"*pubsub.", "pubsub"},
		{"*bigquery.", "bigquery"},
		{"*firestore.", "firestore"},
		{"*vision.", "vision"},
	}

	for _, pkg := range gcpPackages {
		if strings.Contains(typeName, pkg.pattern) {
			return true, pkg.service
		}
	}

	return false, ""
}

// GetPackageInfo はパッケージパスからGCP情報を取得する
func (rt *ResourceTracker) GetPackageInfo(packagePath string) (bool, string) {
	if packagePath == "" {
		return false, ""
	}

	// GCPパッケージのパターン
	gcpPatterns := map[string]string{
		"cloud.google.com/go/spanner":                   "spanner",
		"cloud.google.com/go/storage":                   "storage",
		"cloud.google.com/go/pubsub":                    "pubsub",
		"cloud.google.com/go/bigquery":                  "bigquery",
		"cloud.google.com/go/firestore":                 "firestore",
		"cloud.google.com/go/vision/apiv1":              "vision",
		"cloud.google.com/go/iam/admin/apiv1":           "admin",
		"cloud.google.com/go/recaptchaenterprise/apiv1": "recaptcha",
		"cloud.google.com/go/functions/apiv1":           "functions",
	}

	if service, exists := gcpPatterns[packagePath]; exists {
		return true, service
	}

	// プレフィックスマッチも試行
	for path, service := range gcpPatterns {
		if strings.HasPrefix(packagePath, path) {
			return true, service
		}
	}

	return false, ""
}

// GetTrackedResources は追跡中のリソース一覧を取得する
func (rt *ResourceTracker) GetTrackedResources() []ResourceInfo {
	var resources []ResourceInfo
	for _, info := range rt.variables {
		resources = append(resources, *info)
	}
	return resources
}

// ClearTrackedResources は追跡中のリソースをクリアする
func (rt *ResourceTracker) ClearTrackedResources() {
	rt.variables = make(map[*types.Var]*ResourceInfo)
}

// extractFunctionIdent は関数呼び出しから関数の識別子を抽出する
func (rt *ResourceTracker) extractFunctionIdent(call *ast.CallExpr) *ast.Ident {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun
	case *ast.SelectorExpr:
		return fun.Sel
	default:
		return nil
	}
}

// extractPackagePath は関数呼び出しからパッケージパスを抽出する
func (rt *ResourceTracker) extractPackagePath(call *ast.CallExpr, _ *ast.Ident) string {
	// セレクタ式の場合（pkg.Function または obj.Method）
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		// パッケージ関数の場合（pkg.Function）
		if pkgIdent, ok := sel.X.(*ast.Ident); ok {
			// 型情報からパッケージパスを取得
			if rt.typeInfo != nil && rt.typeInfo.Uses != nil {
				if obj := rt.typeInfo.Uses[pkgIdent]; obj != nil {
					if pkg, ok := obj.(*types.PkgName); ok {
						return pkg.Imported().Path()
					}
				}
			}
		}

		// メソッド呼び出しの場合（obj.Method）
		// sel.Xの型情報を取得してパッケージパスを推定
		if rt.typeInfo != nil && rt.typeInfo.Types != nil {
			if typeAndValue, exists := rt.typeInfo.Types[sel.X]; exists {
				if typeAndValue.Type != nil {
					typeName := typeAndValue.Type.String()
					// 型名からパッケージパスを推定
					if strings.Contains(typeName, "spanner") {
						return "cloud.google.com/go/spanner"
					}
					if strings.Contains(typeName, "storage") {
						return "cloud.google.com/go/storage"
					}
					if strings.Contains(typeName, "pubsub") {
						return "cloud.google.com/go/pubsub"
					}
					if strings.Contains(typeName, "vision") {
						return "cloud.google.com/go/vision"
					}
				}
			}
		}
	}

	return ""
}

// isCreationFunction は関数名がリソース生成関数かどうかを確認する
func (rt *ResourceTracker) isCreationFunction(serviceRule *ServiceRule, funcName string) bool {
	if serviceRule == nil {
		return false
	}

	for _, creationFunc := range serviceRule.CreationFuncs {
		if creationFunc == funcName {
			return true
		}
	}

	return false
}

// createResourceInfo はResourceInfoを作成する
func (rt *ResourceTracker) createResourceInfo(call *ast.CallExpr, serviceName string, serviceRule *ServiceRule) *ResourceInfo {
	if serviceRule == nil || len(serviceRule.CleanupMethods) == 0 {
		return nil
	}

	// 関数名を取得
	funcIdent := rt.extractFunctionIdent(call)
	funcName := ""
	if funcIdent != nil {
		funcName = funcIdent.Name
	}

	// 関数名に基づいてクリーンアップメソッドを決定
	var cleanupMethod string
	var isRequired bool = true

	// 特定の関数に対する特別なクリーンアップメソッド
	switch {
	case funcName == "ReadOnlyTransaction" || funcName == "ReadWriteTransaction" || funcName == "BatchReadOnlyTransaction":
		cleanupMethod = "Close" // Transactionは必ずClose
		isRequired = true
	case funcName == "Query" || funcName == "Read":
		cleanupMethod = "Stop" // IteratorはStop
		isRequired = true
	default:
		// デフォルトのクリーンアップメソッドを取得
		for _, method := range serviceRule.CleanupMethods {
			if method.Required {
				cleanupMethod = method.Method
				isRequired = true
				break
			}
		}

		if cleanupMethod == "" && len(serviceRule.CleanupMethods) > 0 {
			// 必須でなくても最初のメソッドを使用
			cleanupMethod = serviceRule.CleanupMethods[0].Method
			isRequired = serviceRule.CleanupMethods[0].Required
		}
	}

	// ResourceInfoを作成
	resourceInfo := &ResourceInfo{
		Variable:         nil, // 後で設定
		CreationPos:      call.Pos(),
		ServiceType:      serviceName,
		CreationFunction: funcName,
		CleanupMethod:    cleanupMethod,
		IsRequired:       isRequired,
		Scope:            nil, // 後で設定
	}

	// Spannerリソースの場合はエスケープ解析用情報を初期化
	if serviceName == "spanner" {
		resourceInfo.SpannerEscape = rt.initializeSpannerEscapeInfo(funcName)
	}

	return resourceInfo
}

// trackVariableAssignment は変数への代入を追跡する（AST解析改良版）
func (rt *ResourceTracker) trackVariableAssignment(call *ast.CallExpr, resourceInfo *ResourceInfo) {
	if resourceInfo == nil {
		return
	}

	// 呼び出しの親ノードを確認して変数代入を検出
	varName := rt.extractVariableNameFromContext(call)
	if varName != "" {
		resourceInfo.VariableName = varName

		// 型情報から変数を検索
		if rt.typeInfo != nil && rt.typeInfo.Defs != nil {
			for ident, obj := range rt.typeInfo.Defs {
				if obj != nil && ident.Name == varName {
					if varObj, ok := obj.(*types.Var); ok {
						resourceInfo.Variable = varObj
						rt.variables[varObj] = resourceInfo
						return
					}
				}
			}
		}
	}

	// 変数が見つからない場合はダミーの変数を作成
	dummyVar := &types.Var{}
	rt.variables[dummyVar] = resourceInfo
	resourceInfo.Variable = dummyVar
}

// extractVariableNameFromContext は呼び出しの文脈から変数名を抽出する
func (rt *ResourceTracker) extractVariableNameFromContext(call *ast.CallExpr) string {
	// まず実際の変数名を抽出することを試みる
	actualVarName := rt.extractActualVariableName(call)
	if actualVarName != "" {
		return actualVarName
	}

	// 実際の変数名が取得できない場合は推定を使用
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		funcName := sel.Sel.Name
		return rt.inferVariableNameFromFunction(funcName)
	}

	return ""
}

// extractActualVariableName はAST解析により実際の変数名を抽出する
func (rt *ResourceTracker) extractActualVariableName(call *ast.CallExpr) string {
	// クロージャ内の関数パラメータから変数名を抽出
	varName := rt.extractFromClosureParameters(call)
	if varName != "" {
		return varName
	}

	// 通常の代入文から変数名を抽出（将来の拡張用）
	return rt.extractFromAssignmentStatement(call)
}

// extractFromClosureParameters はクロージャパラメータから変数名を抽出する
func (rt *ResourceTracker) extractFromClosureParameters(call *ast.CallExpr) string {
	// ReadWriteTransaction(ctx, func(ctx context.Context, transaction *spanner.ReadWriteTransaction) error { ... })
	// のパターンを検出

	if len(call.Args) < 2 {
		return ""
	}

	// 2番目の引数がクロージャかチェック
	if funcLit, ok := call.Args[1].(*ast.FuncLit); ok {
		if funcLit.Type != nil && funcLit.Type.Params != nil && len(funcLit.Type.Params.List) >= 2 {
			// 2番目のパラメータ（トランザクション）の名前を取得
			secondParam := funcLit.Type.Params.List[1]
			if len(secondParam.Names) > 0 && secondParam.Names[0] != nil {
				return secondParam.Names[0].Name
			}
		}
	}

	return ""
}

// extractFromAssignmentStatement は代入文から変数名を抽出する
func (rt *ResourceTracker) extractFromAssignmentStatement(call *ast.CallExpr) string {
	// AST構造を辿って代入文から実際の変数名を見つける
	actualVarName := rt.findActualAssignmentVariable(call)
	if actualVarName != "" {
		return actualVarName
	}

	// ReadOnlyTransactionの直接呼び出しパターンを特別処理
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "ReadOnlyTransaction" {
			// KultureJPでよく使われる "tx" を推定名として返す
			return "tx"
		}
	}

	return ""
}

// findActualAssignmentVariable は実際の代入文の変数名を探す
func (rt *ResourceTracker) findActualAssignmentVariable(call *ast.CallExpr) string {
	// この実装では AST の親ノードを辿る必要があるため、
	// 一旦、analysis.Pass の Fset と Files を使用して代入文を検索する
	//
	// 簡易実装：通常のパターンをサポート
	// dst := obj.NewWriter(ctx)
	// writer := client.NewWriter(...)
	//
	// より完全な実装は別途追加予定
	return ""
}

// inferVariableNameFromFunction は関数名から一般的な変数名を推定する
func (rt *ResourceTracker) inferVariableNameFromFunction(funcName string) string {
	switch funcName {
	case "NewClient":
		return "client"
	case "NewReader":
		return "reader"
	case "NewWriter":
		return "writer"
	case "ReadOnlyTransaction", "ReadWriteTransaction", "BatchReadOnlyTransaction":
		return "tx"
	case "Query":
		return "iter"
	case "NewImageAnnotatorClient":
		return "client"
	case "NewProductSearchClient":
		return "client"
	default:
		return ""
	}
}

// IntegrateSpannerEscape はSpannerエスケープ解析結果をResourceInfoに統合する
func (rt *ResourceTracker) IntegrateSpannerEscape(resourceInfo *ResourceInfo, escapeAnalyzer *EscapeAnalyzer, funcDecl *ast.FuncDecl) {
	if resourceInfo == nil || escapeAnalyzer == nil {
		return
	}

	// Spanner以外のリソースは対象外
	if resourceInfo.ServiceType != "spanner" {
		return
	}

	// 変数情報がある場合のみエスケープ解析を実行
	if resourceInfo.Variable != nil {
		escapeInfo := escapeAnalyzer.DetectSpannerAutoManagement(resourceInfo.Variable, funcDecl)
		if escapeInfo != nil {
			resourceInfo.SpannerEscape = escapeInfo
		}
	}
}

// FilterAutoManagedResources は自動管理されるリソースをフィルタリングして除外する
func (rt *ResourceTracker) FilterAutoManagedResources(resources []*ResourceInfo) []*ResourceInfo {
	if len(resources) == 0 {
		return nil
	}

	var filtered []*ResourceInfo

	for _, resource := range resources {
		if resource == nil {
			continue
		}

		// Spannerの自動管理リソースをスキップ
		if rt.isAutoManagedResource(resource) {
			continue // 自動管理されるので検査不要
		}

		// その他のリソースは保持
		filtered = append(filtered, resource)
	}

	return filtered
}

// isAutoManagedResource はリソースが自動管理されるかどうかを判定する
func (rt *ResourceTracker) isAutoManagedResource(resource *ResourceInfo) bool {
	if resource.ServiceType == "spanner" &&
		resource.SpannerEscape != nil &&
		resource.SpannerEscape.IsAutoManaged {
		return true
	}

	// 他のサービスの自動管理パターンも今後追加可能
	return false
}

// initializeSpannerEscapeInfo はSpannerリソースのエスケープ情報を初期化する
func (rt *ResourceTracker) initializeSpannerEscapeInfo(funcName string) *SpannerEscapeInfo {
	escapeInfo := &SpannerEscapeInfo{
		IsAutoManaged:        false,
		TransactionType:      funcName,
		IsClosureManaged:     false,
		ClosureDetected:      false,
		AutoManagementReason: "",
	}

	// トランザクション型の正規化
	switch funcName {
	case "ReadWriteTransaction":
		escapeInfo.TransactionType = ReadWriteTransactionType
	case "ReadOnlyTransaction":
		escapeInfo.TransactionType = ReadOnlyTransactionType
	}

	return escapeInfo
}

// isWrappedSpannerTransactionCall はラップされたSpannerトランザクションの呼び出しかチェック
func (rt *ResourceTracker) isWrappedSpannerTransactionCall(callExpr *ast.CallExpr) bool {
	// c.ReadWriteTransaction(ctx, func(ctx context.Context) error { ... })
	// c.ReadOnlyTransaction(ctx, func(ctx context.Context) error { ... })
	if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		methodName := sel.Sel.Name
		if methodName == "ReadWriteTransaction" || methodName == "ReadOnlyTransaction" {
			// 引数にクロージャがあるかチェック
			for _, arg := range callExpr.Args {
				if _, ok := arg.(*ast.FuncLit); ok {
					// さらにレシーバが直接的なSpannerClientではないことを確認
					return rt.isNotDirectSpannerClient(sel.X)
				}
			}
		}
	}
	return false
}

// isNotDirectSpannerClient はレシーバが直接的なSpannerClientでないかチェック
func (rt *ResourceTracker) isNotDirectSpannerClient(expr ast.Expr) bool {
	// x.SpannerClient.ReadWriteTransaction の形式は除外しない
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "SpannerClient" {
			return false // 直接的なSpannerClientなので除外しない
		}
	}

	// その他の場合（c.ReadWriteTransaction等）はラップされたものとして除外
	return true
}

// trackAssignmentStatement は代入文からリソース生成を検出する
func (rt *ResourceTracker) trackAssignmentStatement(assignStmt *ast.AssignStmt, pass *analysis.Pass) {
	if len(assignStmt.Rhs) == 0 {
		return
	}

	for i, rhs := range assignStmt.Rhs {
		if call, ok := rhs.(*ast.CallExpr); ok {
			// ラップされたSpannerトランザクションは除外
			if rt.isWrappedSpannerTransactionCall(call) {
				continue
			}

			// リソース生成かチェック
			if rt.isResourceCreationCall(call) {
				// 複数戻り値の場合は、GCPリソースを返す戻り値のみ追跡
				if rt.shouldTrackMultipleReturnValues(call) {
					rt.trackMultipleReturnValues(assignStmt, call, pass)
				} else {
					// 単一戻り値の場合
					varName := rt.extractVariableNameFromAssignment(assignStmt, i)
					if varName != "" {
						rt.trackCallWithVariableName(call, varName, pass)
					}
				}
			}
		}
	}
}

// extractVariableNameFromAssignment は代入文から変数名を抽出する
func (rt *ResourceTracker) extractVariableNameFromAssignment(assignStmt *ast.AssignStmt, rhsIndex int) string {
	if rhsIndex >= len(assignStmt.Lhs) {
		// 複数戻り値の場合、最初の変数を使用
		if len(assignStmt.Lhs) > 0 {
			if ident, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
				// ブランク識別子は除外
				if ident.Name == "_" {
					return ""
				}
				return ident.Name
			}
		}
		return ""
	}

	if ident, ok := assignStmt.Lhs[rhsIndex].(*ast.Ident); ok {
		// ブランク識別子は除外
		if ident.Name == "_" {
			return ""
		}
		return ident.Name
	}

	return ""
}

// shouldTrackMultipleReturnValues は複数戻り値の関数かどうかを判定
func (rt *ResourceTracker) shouldTrackMultipleReturnValues(call *ast.CallExpr) bool {
	// ReadWriteTransaction系のみ特別扱い（time.Time, errorを返す）
	funcIdent := rt.extractFunctionIdent(call)
	if funcIdent == nil {
		return false
	}

	return strings.Contains(funcIdent.Name, "ReadWriteTransaction")
}

// trackMultipleReturnValues は複数戻り値関数の場合のリソース追跡
func (rt *ResourceTracker) trackMultipleReturnValues(assignStmt *ast.AssignStmt, call *ast.CallExpr, pass *analysis.Pass) {
	// ReadWriteTransactionの場合、最初の戻り値（time.Time）のみGCPリソースではない
	// 実際にはReadWriteTransaction自体はリソースを直接返さないため、何もしない
	// この関数は将来的に他の複数戻り値GCP関数に対応するための拡張ポイント
}

// isResourceCreationCall はリソース生成呼び出しかチェック
func (rt *ResourceTracker) isResourceCreationCall(call *ast.CallExpr) bool {
	// TrackCallの簡略版ロジック
	funcIdent := rt.extractFunctionIdent(call)
	if funcIdent == nil {
		return false
	}

	packagePath := rt.extractPackagePath(call, funcIdent)
	if packagePath == "" {
		return false
	}

	isGCP, serviceName := rt.GetPackageInfo(packagePath)
	if !isGCP {
		return false
	}

	serviceRule := rt.ruleEngine.GetServiceRule(serviceName)
	if serviceRule == nil {
		return false
	}

	return rt.isCreationFunction(serviceRule, funcIdent.Name)
}

// trackCallWithVariableName は実際の変数名でリソース呼び出しを追跡する
func (rt *ResourceTracker) trackCallWithVariableName(call *ast.CallExpr, varName string, pass *analysis.Pass) {
	funcIdent := rt.extractFunctionIdent(call)
	if funcIdent == nil {
		return
	}

	packagePath := rt.extractPackagePath(call, funcIdent)
	isGCP, serviceName := rt.GetPackageInfo(packagePath)
	if !isGCP {
		return
	}

	serviceRule := rt.ruleEngine.GetServiceRule(serviceName)
	if serviceRule == nil {
		return
	}

	// ResourceInfoを作成
	resourceInfo := rt.createResourceInfo(call, serviceName, serviceRule)
	if resourceInfo != nil {
		// 実際の変数名を設定
		resourceInfo.VariableName = varName

		// 型情報から変数を検索
		if rt.typeInfo != nil && pass.Pkg != nil {
			for ident, obj := range rt.typeInfo.Defs {
				if obj != nil && ident.Name == varName {
					if varObj, ok := obj.(*types.Var); ok {
						resourceInfo.Variable = varObj
						rt.variables[varObj] = resourceInfo
						return
					}
				}
			}
		}

		// 変数が見つからない場合はダミーの変数を作成
		dummyVar := &types.Var{}
		rt.variables[dummyVar] = resourceInfo
		resourceInfo.Variable = dummyVar
	}
}
