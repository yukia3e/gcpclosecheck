package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/yukia3e/gcpclosecheck/internal/messages"
)

// DiagnosticGenerator は診断レポートを生成する
type DiagnosticGenerator struct {
	fset *token.FileSet
}

// NewDiagnosticGenerator は新しいDiagnosticGeneratorを作成する
func NewDiagnosticGenerator(fset *token.FileSet) *DiagnosticGenerator {
	return &DiagnosticGenerator{
		fset: fset,
	}
}

// ReportMissingDefer はdefer文が不足しているリソースの診断を生成する
func (dg *DiagnosticGenerator) ReportMissingDefer(resource ResourceInfo) analysis.Diagnostic {
	message := fmt.Sprintf(messages.MissingResourceCleanup,
		resource.Variable.Name(), resource.CleanupMethod)

	suggestedFix := dg.CreateSuggestedFix(
		resource.Variable.Name(),
		resource.CleanupMethod,
		resource.CreationPos,
	)

	return analysis.Diagnostic{
		Pos:            resource.CreationPos,
		End:            resource.CreationPos,
		Category:       "resource-leak",
		Message:        message,
		SuggestedFixes: []analysis.SuggestedFix{suggestedFix},
	}
}

// ReportMissingContextCancel はcontext.WithCancelのキャンセル関数が不足している診断を生成する
func (dg *DiagnosticGenerator) ReportMissingContextCancel(contextInfo ContextInfo) analysis.Diagnostic {
	message := fmt.Sprintf(messages.MissingContextCancel,
		contextInfo.CancelFunc.Name())

	suggestedFix := dg.CreateSuggestedFix(
		contextInfo.CancelFunc.Name(),
		"",
		contextInfo.CreationPos,
	)

	return analysis.Diagnostic{
		Pos:            contextInfo.CreationPos,
		End:            contextInfo.CreationPos,
		Category:       "context-leak",
		Message:        message,
		SuggestedFixes: []analysis.SuggestedFix{suggestedFix},
	}
}

// CreateSuggestedFix はdefer文追加の修正提案を作成する
func (dg *DiagnosticGenerator) CreateSuggestedFix(variableName, method string, creationPos token.Pos) analysis.SuggestedFix {
	var message string
	var deferStatement string

	if method == "" {
		// Context cancel function
		message = fmt.Sprintf(messages.AddDeferStatement, variableName)
		deferStatement = fmt.Sprintf("defer %s()", variableName)
	} else {
		// Resource cleanup method
		message = fmt.Sprintf(messages.AddDeferMethodCall, variableName, method)
		deferStatement = fmt.Sprintf("defer %s.%s()", variableName, method)
	}

	// TextEdit を作成 - リソース作成後の次の行にdefer文を挿入
	textEdit := analysis.TextEdit{
		Pos:     creationPos,
		End:     creationPos,
		NewText: []byte("\n\t" + deferStatement),
	}

	return analysis.SuggestedFix{
		Message:   message,
		TextEdits: []analysis.TextEdit{textEdit},
	}
}

// ShouldIgnoreNolint はnolintディレクティブをチェックし、診断を抑制すべきかどうかを判定する
func (dg *DiagnosticGenerator) ShouldIgnoreNolint(file *ast.File, pos token.Pos) bool {
	// pos の行番号を取得
	position := dg.fset.Position(pos)
	targetLine := position.Line

	// ファイル内のコメントを検索
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			commentPos := dg.fset.Position(comment.Pos())

			// 同じ行または直前の行のコメントをチェック
			if commentPos.Line == targetLine || commentPos.Line == targetLine-1 {
				if dg.isNolintComment(comment.Text) {
					return true
				}
			}
		}
	}

	return false
}

// isNolintComment はコメントがnolintディレクティブかどうかを判定する
func (dg *DiagnosticGenerator) isNolintComment(commentText string) bool {
	// //nolint:gcpclosecheck または //nolint:all パターンをチェック
	if strings.Contains(commentText, "nolint:gcpclosecheck") {
		return true
	}
	if strings.Contains(commentText, "nolint:all") {
		return true
	}
	return false
}

// GenerateLocationInfo はファイル位置情報を含む詳細な診断情報を生成する
func (dg *DiagnosticGenerator) GenerateLocationInfo(pos token.Pos) string {
	position := dg.fset.Position(pos)
	return fmt.Sprintf("%s:%d:%d", position.Filename, position.Line, position.Column)
}
