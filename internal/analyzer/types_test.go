package analyzer

import (
	"go/token"
	"go/types"
	"testing"
)

func TestResourceInfo(t *testing.T) {
	// ResourceInfo 構造体のテスト
	variable := types.NewVar(token.NoPos, nil, "client", nil)
	scope := types.NewScope(nil, token.NoPos, token.NoPos, "test")

	resource := ResourceInfo{
		Variable:      variable,
		CreationPos:   token.Pos(100),
		ServiceType:   "spanner",
		CleanupMethod: "Close",
		IsRequired:    true,
		Scope:         scope,
	}

	if resource.Variable != variable {
		t.Errorf("Variable が期待値と異なります")
	}
	if resource.ServiceType != "spanner" {
		t.Errorf("ServiceType が期待値と異なります: %s", resource.ServiceType)
	}
	if !resource.IsRequired {
		t.Errorf("IsRequired が期待値と異なります")
	}
}

func TestResourceInfoConstructor(t *testing.T) {
	// NewResourceInfo コンストラクタのテスト
	variable := types.NewVar(token.NoPos, nil, "client", nil)
	scope := types.NewScope(nil, token.NoPos, token.NoPos, "test")

	resource := NewResourceInfo(variable, token.Pos(100), "storage", "NewClient", "Close", true, scope)

	if resource == nil {
		t.Fatal("NewResourceInfo が nil を返しました")
	}
	if resource.ServiceType != "storage" {
		t.Errorf("ServiceType が期待値と異なります: %s", resource.ServiceType)
	}
	if resource.CleanupMethod != "Close" {
		t.Errorf("CleanupMethod が期待値と異なります: %s", resource.CleanupMethod)
	}
}

func TestContextInfo(t *testing.T) {
	// ContextInfo 構造体のテスト
	variable := types.NewVar(token.NoPos, nil, "ctx", nil)
	cancelFunc := types.NewVar(token.NoPos, nil, "cancel", nil)

	contextInfo := ContextInfo{
		Variable:    variable,
		CancelFunc:  cancelFunc,
		CreationPos: token.Pos(200),
		IsDeferred:  false,
	}

	if contextInfo.Variable != variable {
		t.Errorf("Variable が期待値と異なります")
	}
	if contextInfo.CancelFunc != cancelFunc {
		t.Errorf("CancelFunc が期待値と異なります")
	}
	if contextInfo.IsDeferred {
		t.Errorf("IsDeferred が期待値と異なります")
	}
}

func TestServiceRule(t *testing.T) {
	// ServiceRule 構造体のテスト
	cleanupMethods := []CleanupMethod{
		{Method: "Close", Required: true, Description: "クライアント接続のクローズ"},
		{Method: "Stop", Required: true, Description: "RowIterator の停止"},
	}

	rule := ServiceRule{
		ServiceName:    "spanner",
		PackagePath:    "cloud.google.com/go/spanner",
		CreationFuncs:  []string{"NewClient", "ReadOnlyTransaction"},
		CleanupMethods: cleanupMethods,
	}

	if rule.ServiceName != "spanner" {
		t.Errorf("ServiceName が期待値と異なります: %s", rule.ServiceName)
	}
	if len(rule.CreationFuncs) != 2 {
		t.Errorf("CreationFuncs の長さが期待値と異なります: %d", len(rule.CreationFuncs))
	}
	if len(rule.CleanupMethods) != 2 {
		t.Errorf("CleanupMethods の長さが期待値と異なります: %d", len(rule.CleanupMethods))
	}
}

func TestServiceRuleMethods(t *testing.T) {
	// ServiceRule のメソッドテスト
	cleanupMethods := []CleanupMethod{
		{Method: "Close", Required: true, Description: "クライアント接続のクローズ"},
		{Method: "Flush", Required: false, Description: "バッファフラッシュ"},
		{Method: "Stop", Required: true, Description: "RowIterator の停止"},
	}

	rule := ServiceRule{
		ServiceName:    "spanner",
		PackagePath:    "cloud.google.com/go/spanner",
		CreationFuncs:  []string{"NewClient", "ReadOnlyTransaction"},
		CleanupMethods: cleanupMethods,
	}

	// HasCreationFunc のテスト
	if !rule.HasCreationFunc("NewClient") {
		t.Errorf("NewClient が生成関数として認識されません")
	}
	if rule.HasCreationFunc("NonExistentFunc") {
		t.Errorf("存在しない関数が生成関数として認識されました")
	}

	// GetRequiredCleanupMethods のテスト
	required := rule.GetRequiredCleanupMethods()
	if len(required) != 2 {
		t.Errorf("必須解放メソッド数が期待値と異なります: %d", len(required))
	}
	expectedMethods := map[string]bool{"Close": false, "Stop": false}
	for _, method := range required {
		if _, exists := expectedMethods[method.Method]; exists {
			expectedMethods[method.Method] = true
		}
	}
	for method, found := range expectedMethods {
		if !found {
			t.Errorf("必須メソッド %s が見つかりません", method)
		}
	}
}

func TestCleanupMethod(t *testing.T) {
	// CleanupMethod 構造体のテスト
	method := CleanupMethod{
		Method:      "Close",
		Required:    true,
		Description: "リソースのクローズ",
	}

	if method.Method != "Close" {
		t.Errorf("Method が期待値と異なります: %s", method.Method)
	}
	if !method.Required {
		t.Errorf("Required が期待値と異なります")
	}
}

func TestEscapeInfo(t *testing.T) {
	// EscapeInfo 構造体のテスト
	escapeInfo := EscapeInfo{
		IsReturned:      true,
		IsFieldAssigned: false,
		EscapeReason:    "関数戻り値として返却",
	}

	if !escapeInfo.IsReturned {
		t.Errorf("IsReturned が期待値と異なります")
	}
	if escapeInfo.IsFieldAssigned {
		t.Errorf("IsFieldAssigned が期待値と異なります")
	}
	if escapeInfo.EscapeReason != "関数戻り値として返却" {
		t.Errorf("EscapeReason が期待値と異なります: %s", escapeInfo.EscapeReason)
	}
}

func TestEscapeInfoHelpers(t *testing.T) {
	// NewEscapeInfo コンストラクタのテスト
	escapeInfo1 := NewEscapeInfo(true, false, "戻り値として返却")
	if !escapeInfo1.HasEscaped() {
		t.Errorf("戻り値として返却される場合、HasEscaped() は true を返すべき")
	}

	escapeInfo2 := NewEscapeInfo(false, true, "フィールドに代入")
	if !escapeInfo2.HasEscaped() {
		t.Errorf("フィールドに代入される場合、HasEscaped() は true を返すべき")
	}

	escapeInfo3 := NewEscapeInfo(false, false, "逃げない")
	if escapeInfo3.HasEscaped() {
		t.Errorf("逃げない場合、HasEscaped() は false を返すべき")
	}
}

func TestResourceInfoValidation(t *testing.T) {
	// ResourceInfo バリデーションのテスト
	variable := types.NewVar(token.NoPos, nil, "client", nil)
	scope := types.NewScope(nil, token.NoPos, token.NoPos, "test")

	// 正常なケース
	resource := NewResourceInfo(variable, token.Pos(100), "spanner", "NewClient", "Close", true, scope)
	if err := resource.Validate(); err != nil {
		t.Errorf("正常なResourceInfoでバリデーションエラー: %v", err)
	}

	// Variable が nil のケース
	invalidResource := NewResourceInfo(nil, token.Pos(100), "spanner", "NewClient", "Close", true, scope)
	if err := invalidResource.Validate(); err == nil {
		t.Errorf("Variable が nil の場合にバリデーションエラーになるべき")
	}

	// ServiceType が空のケース
	invalidResource2 := NewResourceInfo(variable, token.Pos(100), "", "NewClient", "Close", true, scope)
	if err := invalidResource2.Validate(); err == nil {
		t.Errorf("ServiceType が空の場合にバリデーションエラーになるべき")
	}
}

// TestSpannerEscapeInfo - Spannerエスケープ解析のテスト（RED: 失敗テスト）
func TestSpannerEscapeInfo(t *testing.T) {
	// SpannerEscapeInfo 構造体の基本テスト
	spannerEscape := SpannerEscapeInfo{
		IsAutoManaged:        true,
		TransactionType:      "ReadWriteTransaction",
		IsClosureManaged:     true,
		ClosureDetected:      true,
		AutoManagementReason: "フレームワーク自動管理",
	}

	if !spannerEscape.IsAutoManaged {
		t.Errorf("IsAutoManaged が期待値と異なります")
	}
	if spannerEscape.TransactionType != "ReadWriteTransaction" {
		t.Errorf("TransactionType が期待値と異なります: %s", spannerEscape.TransactionType)
	}
	if !spannerEscape.IsClosureManaged {
		t.Errorf("IsClosureManaged が期待値と異なります")
	}
}

func TestSpannerEscapeInfoConstructor(t *testing.T) {
	// NewSpannerEscapeInfo コンストラクタのテスト
	spannerEscape := NewSpannerEscapeInfo("ReadOnlyTransaction", true, "クロージャ内自動管理")

	if spannerEscape == nil {
		t.Fatal("NewSpannerEscapeInfo が nil を返しました")
	}
	if spannerEscape.TransactionType != "ReadOnlyTransaction" {
		t.Errorf("TransactionType が期待値と異なります: %s", spannerEscape.TransactionType)
	}
	if !spannerEscape.IsAutoManaged {
		t.Errorf("IsAutoManaged が期待値と異なります")
	}
	if spannerEscape.AutoManagementReason != "クロージャ内自動管理" {
		t.Errorf("AutoManagementReason が期待値と異なります: %s", spannerEscape.AutoManagementReason)
	}
}

func TestSpannerTransactionConstants(t *testing.T) {
	// Spannerトランザクション定数のテスト
	if ReadWriteTransactionType != "ReadWriteTransaction" {
		t.Errorf("ReadWriteTransactionType 定数が期待値と異なります: %s", ReadWriteTransactionType)
	}
	if ReadOnlyTransactionType != "ReadOnlyTransaction" {
		t.Errorf("ReadOnlyTransactionType 定数が期待値と異なります: %s", ReadOnlyTransactionType)
	}
}

func TestResourceInfoWithSpannerEscape(t *testing.T) {
	// ResourceInfo に SpannerEscape 情報が連携されるテスト
	variable := types.NewVar(token.NoPos, nil, "txn", nil)
	scope := types.NewScope(nil, token.NoPos, token.NoPos, "test")

	resource := NewResourceInfo(variable, token.Pos(100), "spanner", "ReadWriteTransaction", "Close", true, scope)
	spannerEscape := NewSpannerEscapeInfo("ReadWriteTransaction", true, "クロージャ管理")

	// SpannerEscape 情報を設定
	resource.SetSpannerEscape(spannerEscape)

	retrievedEscape := resource.GetSpannerEscape()
	if retrievedEscape == nil {
		t.Fatal("SpannerEscape 情報が取得できません")
	}
	if !retrievedEscape.IsAutoManaged {
		t.Errorf("SpannerEscape の IsAutoManaged が期待値と異なります")
	}

	// HasSpannerEscape のテスト
	if !resource.HasSpannerEscape() {
		t.Errorf("HasSpannerEscape() が false を返しましたが、SpannerEscape が設定されています")
	}

	// ShouldSkipSpannerCleanup のテスト
	if !resource.ShouldSkipSpannerCleanup() {
		t.Errorf("自動管理リソースの場合、ShouldSkipSpannerCleanup() は true を返すべきです")
	}
}

func TestSpannerEscapeInfoValidation(t *testing.T) {
	// SpannerEscapeInfo バリデーションのテスト

	// 正常なケース
	validEscape := NewSpannerEscapeInfo(ReadWriteTransactionType, true, "フレームワーク自動管理")
	if err := validEscape.Validate(); err != nil {
		t.Errorf("正常なSpannerEscapeInfoでバリデーションエラー: %v", err)
	}

	// 不正なTransactionTypeのケース
	invalidEscape := &SpannerEscapeInfo{
		IsAutoManaged:        true,
		TransactionType:      "InvalidTransaction",
		AutoManagementReason: "テスト",
	}
	if err := invalidEscape.Validate(); err == nil {
		t.Errorf("不正なTransactionTypeの場合にバリデーションエラーになるべき")
	}

	// 自動管理なのにReasonが空のケース
	invalidEscape2 := &SpannerEscapeInfo{
		IsAutoManaged:        true,
		TransactionType:      ReadWriteTransactionType,
		AutoManagementReason: "",
	}
	if err := invalidEscape2.Validate(); err == nil {
		t.Errorf("自動管理でReason空の場合にバリデーションエラーになるべき")
	}
}

func TestSpannerEscapeInfoShouldSkipCleanup(t *testing.T) {
	// ShouldSkipCleanup のテスト

	// 自動管理の場合
	autoManaged := NewSpannerEscapeInfo(ReadWriteTransactionType, true, "自動管理")
	if !autoManaged.ShouldSkipCleanup() {
		t.Errorf("自動管理の場合、ShouldSkipCleanup() は true を返すべき")
	}

	// 手動管理の場合
	manualManaged := NewSpannerEscapeInfo(ReadWriteTransactionType, false, "")
	if manualManaged.ShouldSkipCleanup() {
		t.Errorf("手動管理の場合、ShouldSkipCleanup() は false を返すべき")
	}
}

func TestNewSpannerEscapeInfoValidation(t *testing.T) {
	// NewSpannerEscapeInfo のバリデーション動作テスト

	// 不正なTransactionTypeを指定した場合はデフォルト値に修正
	spannerEscape := NewSpannerEscapeInfo("InvalidType", true, "テスト")
	if spannerEscape.TransactionType != ReadWriteTransactionType {
		t.Errorf("不正なTransactionTypeはデフォルト値に修正されるべき: %s", spannerEscape.TransactionType)
	}

	// 正常なTransactionTypeの場合はそのまま
	spannerEscape2 := NewSpannerEscapeInfo(ReadOnlyTransactionType, false, "")
	if spannerEscape2.TransactionType != ReadOnlyTransactionType {
		t.Errorf("正常なTransactionTypeがそのまま設定されるべき: %s", spannerEscape2.TransactionType)
	}
}

// タスク13: Context cancel情報の型定義追加テスト
func TestDeferCancelInfo(t *testing.T) {
	tests := []struct {
		name            string
		cancelVarName   string
		deferPos        token.Pos
		scopeDepth      int
		isValid         bool
		expectedVarName string
	}{
		{
			name:            "基本的なcancel変数",
			cancelVarName:   "cancel",
			deferPos:        token.Pos(100),
			scopeDepth:      1,
			isValid:         true,
			expectedVarName: "cancel",
		},
		{
			name:            "複雑な命名のcancel変数",
			cancelVarName:   "timeoutCancel",
			deferPos:        token.Pos(200),
			scopeDepth:      2,
			isValid:         true,
			expectedVarName: "timeoutCancel",
		},
		{
			name:            "無効なcancel変数",
			cancelVarName:   "",
			deferPos:        token.Pos(300),
			scopeDepth:      0,
			isValid:         false,
			expectedVarName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// DeferCancelInfoを作成（まだ実装されていない）
			deferInfo := NewDeferCancelInfo(tt.cancelVarName, tt.deferPos, tt.scopeDepth, tt.isValid)

			if deferInfo == nil {
				t.Fatal("NewDeferCancelInfo should not return nil")
			}

			if deferInfo.CancelVarName != tt.expectedVarName {
				t.Errorf("CancelVarName = %v, want %v", deferInfo.CancelVarName, tt.expectedVarName)
			}

			if deferInfo.DeferPos != tt.deferPos {
				t.Errorf("DeferPos = %v, want %v", deferInfo.DeferPos, tt.deferPos)
			}

			if deferInfo.ScopeDepth != tt.scopeDepth {
				t.Errorf("ScopeDepth = %v, want %v", deferInfo.ScopeDepth, tt.scopeDepth)
			}

			if deferInfo.IsValid != tt.isValid {
				t.Errorf("IsValid = %v, want %v", deferInfo.IsValid, tt.isValid)
			}

			// バリデーションテスト
			err := deferInfo.Validate()
			if tt.isValid && err != nil {
				t.Errorf("Valid defer cancel info should not return error: %v", err)
			}
			if !tt.isValid && err == nil {
				t.Error("Invalid defer cancel info should return error")
			}
		})
	}
}

func TestContextInfoWithDeferInfo(t *testing.T) {
	// ContextInfoとDeferCancelInfoの連携テスト
	variable := types.NewVar(token.NoPos, nil, "ctx", nil)
	cancelFunc := types.NewVar(token.NoPos, nil, "cancel", nil)

	contextInfo := NewContextInfo(variable, cancelFunc, token.Pos(100), false)

	// DeferCancelInfoを設定（まだ実装されていない）
	deferInfo := NewDeferCancelInfo("cancel", token.Pos(200), 1, true)
	contextInfo.SetDeferInfo(deferInfo)

	// DeferInfo取得のテスト
	retrievedDeferInfo := contextInfo.GetDeferInfo()
	if retrievedDeferInfo == nil {
		t.Error("DeferInfo should not be nil after setting")
	}

	if retrievedDeferInfo.CancelVarName != deferInfo.CancelVarName ||
		retrievedDeferInfo.DeferPos != deferInfo.DeferPos ||
		retrievedDeferInfo.ScopeDepth != deferInfo.ScopeDepth ||
		retrievedDeferInfo.IsValid != deferInfo.IsValid {
		t.Error("Retrieved DeferInfo should match the set DeferInfo")
	}

	// HasDeferInfoのテスト
	if !contextInfo.HasDeferInfo() {
		t.Error("HasDeferInfo should return true after setting DeferInfo")
	}
}

func TestDeferCancelInfoValidation(t *testing.T) {
	tests := []struct {
		name      string
		deferInfo *DeferCancelInfo
		wantErr   bool
	}{
		{
			name: "有効なDeferCancelInfo",
			deferInfo: &DeferCancelInfo{
				CancelVarName: "cancel",
				DeferPos:      token.Pos(100),
				ScopeDepth:    1,
				IsValid:       true,
			},
			wantErr: false,
		},
		{
			name: "変数名が空のDeferCancelInfo",
			deferInfo: &DeferCancelInfo{
				CancelVarName: "",
				DeferPos:      token.Pos(100),
				ScopeDepth:    1,
				IsValid:       false,
			},
			wantErr: true,
		},
		{
			name: "不正な位置のDeferCancelInfo",
			deferInfo: &DeferCancelInfo{
				CancelVarName: "cancel",
				DeferPos:      token.NoPos,
				ScopeDepth:    1,
				IsValid:       false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.deferInfo.Validate()

			if tt.wantErr && err == nil {
				t.Error("Expected error, but got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestContextInfoIntegration(t *testing.T) {
	// Context情報管理の統合テスト
	variable := types.NewVar(token.NoPos, nil, "ctx", nil)
	cancelFunc := types.NewVar(token.NoPos, nil, "cancel", nil)

	// ContextInfoを作成
	contextInfo := NewContextInfo(variable, cancelFunc, token.Pos(100), false)

	// DeferCancelInfoを作成して設定
	deferInfo1 := NewDeferCancelInfo("cancel", token.Pos(200), 1, true)
	deferInfo2 := NewDeferCancelInfo("cancel", token.Pos(300), 2, true)

	contextInfo.SetDeferInfo(deferInfo1)

	// 最初のDeferInfoが設定されることを確認
	if !contextInfo.HasDeferInfo() {
		t.Error("ContextInfo should have DeferInfo")
	}

	firstDefer := contextInfo.GetDeferInfo()
	if firstDefer.DeferPos != token.Pos(200) {
		t.Errorf("First defer pos = %v, want %v", firstDefer.DeferPos, token.Pos(200))
	}

	// 複数のDeferInfoを追加（まだ実装されていない）
	contextInfo.AddDeferInfo(deferInfo2)

	// 複数のDeferInfoが管理されることを確認
	allDefers := contextInfo.GetAllDeferInfos()
	if len(allDefers) != 2 {
		t.Errorf("Expected 2 defer infos, got %d", len(allDefers))
	}

	t.Logf("✓ Context cancel information integration test completed")
}
