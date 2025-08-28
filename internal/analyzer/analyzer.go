package analyzer

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer は GCP リソースの解放漏れを検出する静的解析ツール
var Analyzer = &analysis.Analyzer{
	Name: "gcpclosecheck",
	Doc:  "detect missing Close/Stop/Cancel calls for GCP resources",
	Run:  run,
}

// run は解析のメイン実行関数
func run(pass *analysis.Pass) (interface{}, error) {
	// 型チェックエラーの確認
	if len(pass.TypeErrors) > 0 {
		// 型エラーがある場合は警告を出力して解析をスキップ
		foundDependencyError := false
		for _, err := range pass.TypeErrors {
			if isUndefinedIdentifierError(err.Error()) {
				foundDependencyError = true
				break
			}
		}

		if foundDependencyError {
			// 依存関係の問題を示唆する診断メッセージを出力
			pass.Report(analysis.Diagnostic{
				Pos:     pass.Files[0].Pos(), // ファイルの先頭位置を使用
				Message: "依存関係の問題でファイルを解析できません。パッケージ単位での解析を推奨します（例: ./internal/infrastructure/spanner/ 形式）。",
			})
			return nil, nil // 解析を中断
		}
	}

	// 各コンポーネントを初期化
	serviceRuleEngine := NewServiceRuleEngine()
	if err := serviceRuleEngine.LoadDefaultRules(); err != nil {
		return nil, err
	}

	// パッケージ例外判定を実行
	packagePath := getPackagePath(pass)
	shouldExempt, exemptReason := serviceRuleEngine.ShouldExemptPackage(packagePath)

	// パッケージが例外対象でない場合、ファイルパスベースの例外判定も実行
	if !shouldExempt {
		shouldExempt, exemptReason = checkFileBasedExemptions(pass, serviceRuleEngine)
	}

	// パッケージまたはファイルが例外対象の場合は診断を生成せずに終了
	if shouldExempt {
		// デバッグログ出力（将来的にログレベル制御可能にする）
		_ = exemptReason // 例外理由を記録（後でログ出力に使用）
		return nil, nil
	}

	resourceTracker := NewResourceTracker(pass.TypesInfo, serviceRuleEngine)
	deferAnalyzer := NewDeferAnalyzer(resourceTracker)
	contextAnalyzer := NewContextAnalyzer()
	escapeAnalyzer := NewEscapeAnalyzer()

	// ResourceTracker でリソース生成を検出
	resources := resourceTracker.FindResourceCreation(pass)

	// ContextAnalyzer でコンテキストキャンセレーション問題を検出
	contextDiagnostics := contextAnalyzer.FindMissingCancels(pass)

	// 診断レポート
	for _, diagnostic := range contextDiagnostics {
		pass.Report(diagnostic)
	}

	// 各ファイルを解析
	for _, file := range pass.Files {
		// 各関数を解析
		ast.Inspect(file, func(n ast.Node) bool {
			if fn, ok := n.(*ast.FuncDecl); ok {
				if fn.Body != nil {
					// 関数内のリソースを収集・フィルタリング
					functionResources := collectAndFilterFunctionResources(
						resources, fn, pass, escapeAnalyzer)

					// 自動管理リソースの最終フィルタリング
					functionResources = applyAutoManagedResourceFiltering(
						functionResources, resourceTracker)

					// DeferAnalyzer で関数全体を検証（リソース情報を渡す）
					if len(functionResources) > 0 {
						diagnostics := deferAnalyzer.AnalyzeDefers(fn, functionResources)
						for _, diagnostic := range diagnostics {
							pass.Report(diagnostic)
						}
					}
				}
			}
			return true
		})
	}

	return nil, nil
}

// isResourceInFunction は指定されたリソースが関数内で生成されたかどうかを判定する
func isResourceInFunction(resource ResourceInfo, fn *ast.FuncDecl, pass *analysis.Pass) bool {
	if fn.Body == nil {
		return false
	}

	// 関数内でリソースが生成される位置が関数の範囲内かどうかを確認
	fnStart := fn.Body.Lbrace
	fnEnd := fn.Body.Rbrace

	return resource.CreationPos >= fnStart && resource.CreationPos <= fnEnd
}

// integrateSpannerEscapeAnalysis はSpannerエスケープ解析をリソースに統合する
func integrateSpannerEscapeAnalysis(resource ResourceInfo, escapeAnalyzer *EscapeAnalyzer, fn *ast.FuncDecl) ResourceInfo {
	// Spanner以外のサービスはそのまま返す
	if resource.ServiceType != "spanner" {
		return resource
	}

	// 変数情報がない場合はそのまま返す
	if resource.Variable == nil {
		return resource
	}

	// Spannerエスケープ解析を実行
	spannerEscape := escapeAnalyzer.DetectSpannerAutoManagement(resource.Variable, fn)
	if spannerEscape != nil {
		resource.SpannerEscape = spannerEscape
	}

	return resource
}

// shouldSkipResourceWithSpannerIntegration はSpanner統合を考慮したスキップ判定を行う
func shouldSkipResourceWithSpannerIntegration(resource ResourceInfo, escapeInfo EscapeInfo, escapeAnalyzer *EscapeAnalyzer) (bool, string) {
	// 1. 既存のエスケープ分析判定（戻り値、フィールド代入など）
	shouldSkip, reason := escapeAnalyzer.ShouldSkipResource(resource, escapeInfo)
	if shouldSkip {
		return true, reason
	}

	// 2. Spannerの自動管理判定
	if isSpannerAutoManagedResource(resource) {
		return true, resource.SpannerEscape.AutoManagementReason
	}

	// 3. 他のサービス固有の判定も将来的に追加可能

	return false, ""
}

// isSpannerAutoManagedResource はSpannerリソースが自動管理されているかチェック
func isSpannerAutoManagedResource(resource ResourceInfo) bool {
	return resource.ServiceType == "spanner" &&
		resource.SpannerEscape != nil &&
		resource.SpannerEscape.IsAutoManaged
}

// convertToPointerSlice は[]ResourceInfoを[]*ResourceInfoに変換する
func convertToPointerSlice(resources []ResourceInfo) []*ResourceInfo {
	result := make([]*ResourceInfo, len(resources))
	for i := range resources {
		result[i] = &resources[i]
	}
	return result
}

// convertFromPointerSlice は[]*ResourceInfoを[]ResourceInfoに変換する
func convertFromPointerSlice(resources []*ResourceInfo) []ResourceInfo {
	result := make([]ResourceInfo, len(resources))
	for i, r := range resources {
		if r != nil {
			result[i] = *r
		}
	}
	return result
}

// collectAndFilterFunctionResources は関数内のリソースを収集しフィルタリングする
func collectAndFilterFunctionResources(
	resources []ResourceInfo,
	fn *ast.FuncDecl,
	pass *analysis.Pass,
	escapeAnalyzer *EscapeAnalyzer) []ResourceInfo {

	var functionResources []ResourceInfo

	for _, resource := range resources {
		// 関数スコープ内のリソースのみを対象とする
		if !isResourceInFunction(resource, fn, pass) {
			continue
		}

		// Spannerエスケープ解析統合
		resource = integrateSpannerEscapeAnalysis(resource, escapeAnalyzer, fn)

		// エスケープ分析
		escapeInfo := escapeAnalyzer.AnalyzeEscape(resource.Variable, fn)

		// スキップ判定（Spanner自動管理判定を含む）
		shouldSkip, _ := shouldSkipResourceWithSpannerIntegration(resource, escapeInfo, escapeAnalyzer)
		if !shouldSkip {
			functionResources = append(functionResources, resource)
		}
	}

	return functionResources
}

// applyAutoManagedResourceFiltering は自動管理リソースのフィルタリングを適用する
func applyAutoManagedResourceFiltering(
	resources []ResourceInfo,
	resourceTracker *ResourceTracker) []ResourceInfo {

	// ポインタスライスに変換
	resourcePtrs := convertToPointerSlice(resources)

	// ResourceTrackerでフィルタリング
	filteredResourcePtrs := resourceTracker.FilterAutoManagedResources(resourcePtrs)

	// 元の形式に戻す
	return convertFromPointerSlice(filteredResourcePtrs)
}

// getPackagePath はanalysis.Passからパッケージパスを取得する
func getPackagePath(pass *analysis.Pass) string {
	if pass.Pkg == nil {
		return ""
	}
	return pass.Pkg.Path()
}

// isUndefinedIdentifierError は未定義識別子エラーかどうかを判定する
func isUndefinedIdentifierError(errMsg string) bool {
	return strings.Contains(errMsg, "undefined:") ||
		strings.Contains(errMsg, "undeclared name:") ||
		strings.Contains(errMsg, "not declared") ||
		strings.Contains(errMsg, "could not import")
}

// checkFileBasedExemptions はファイルパスベースの例外判定を行う
func checkFileBasedExemptions(pass *analysis.Pass, serviceRuleEngine *ServiceRuleEngine) (bool, string) {
	// 各ファイルのパスをチェックして例外対象か判定
	for _, file := range pass.Files {
		if pass.Fset == nil {
			continue
		}

		// ファイル位置からファイルパスを取得
		position := pass.Fset.Position(file.Pos())
		filePath := position.Filename

		// ファイルパスベースの例外判定
		if serviceRuleEngine.config != nil {
			isExempt, reason := serviceRuleEngine.config.ShouldExemptFilePath(filePath)
			if isExempt {
				return true, reason
			}
		}
	}

	return false, ""
}
