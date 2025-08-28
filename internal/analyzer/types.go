package analyzer

import (
	"errors"
	"go/token"
	"go/types"

	"github.com/yukia3e/gcpclosecheck/internal/messages"
)

// ResourceInfo は GCP リソースの生成情報と解放要求を表す
type ResourceInfo struct {
	Variable         *types.Var         // 変数の型情報
	VariableName     string             // 変数名（文字列）
	CreationPos      token.Pos          // 生成位置
	ServiceType      string             // GCP サービスタイプ（spanner, storage, pubsub 等）
	CreationFunction string             // 生成関数名（NewClient, ReadOnlyTransaction 等）
	CleanupMethod    string             // 解放メソッド名（Close, Stop, Cleanup）
	IsRequired       bool               // 解放が必須かどうか
	Scope            *types.Scope       // 変数のスコープ
	SpannerEscape    *SpannerEscapeInfo // Spannerエスケープ情報（Spannerリソースのみ）
}

// NewResourceInfo は ResourceInfo のコンストラクタ
func NewResourceInfo(variable *types.Var, creationPos token.Pos, serviceType, creationFunction, cleanupMethod string, isRequired bool, scope *types.Scope) *ResourceInfo {
	variableName := ""
	if variable != nil {
		variableName = variable.Name()
	}

	return &ResourceInfo{
		Variable:         variable,
		VariableName:     variableName,
		CreationPos:      creationPos,
		ServiceType:      serviceType,
		CreationFunction: creationFunction,
		CleanupMethod:    cleanupMethod,
		IsRequired:       isRequired,
		Scope:            scope,
		SpannerEscape:    nil, // デフォルトではnilに設定
	}
}

// Validate は ResourceInfo の妥当性を検証する
func (r *ResourceInfo) Validate() error {
	if r.Variable == nil {
		return errors.New(messages.VariableCannotBeNil)
	}
	if r.ServiceType == "" {
		return errors.New(messages.ServiceTypeCannotBeEmpty)
	}
	if r.CleanupMethod == "" {
		return errors.New(messages.CleanupMethodCannotBeEmpty)
	}
	return nil
}

// SetSpannerEscape は SpannerEscapeInfo を設定する
func (r *ResourceInfo) SetSpannerEscape(escape *SpannerEscapeInfo) {
	r.SpannerEscape = escape
}

// GetSpannerEscape は SpannerEscapeInfo を取得する
func (r *ResourceInfo) GetSpannerEscape() *SpannerEscapeInfo {
	return r.SpannerEscape
}

// HasSpannerEscape は SpannerEscapeInfo が設定されているかどうかを判定する
func (r *ResourceInfo) HasSpannerEscape() bool {
	return r.SpannerEscape != nil
}

// ShouldSkipSpannerCleanup は、Spannerリソースの解放処理をスキップすべきかどうかを判定する
func (r *ResourceInfo) ShouldSkipSpannerCleanup() bool {
	if r.SpannerEscape == nil {
		return false
	}
	return r.SpannerEscape.ShouldSkipCleanup()
}

// ContextInfo は context.WithCancel/WithTimeout の追跡情報を表す
type ContextInfo struct {
	Variable    *types.Var        // context 変数
	CancelFunc  *types.Var        // cancel 関数
	CreationPos token.Pos         // 生成位置
	IsDeferred  bool              // defer で呼ばれているかどうか
	DeferInfos  []DeferCancelInfo // defer情報のリスト（複数のdeferに対応）
}

// NewContextInfo は ContextInfo のコンストラクタ
func NewContextInfo(variable, cancelFunc *types.Var, creationPos token.Pos, isDeferred bool) *ContextInfo {
	return &ContextInfo{
		Variable:    variable,
		CancelFunc:  cancelFunc,
		CreationPos: creationPos,
		IsDeferred:  isDeferred,
		DeferInfos:  make([]DeferCancelInfo, 0),
	}
}

// Validate は ContextInfo の妥当性を検証する
func (c *ContextInfo) Validate() error {
	if c.Variable == nil {
		return errors.New(messages.VariableCannotBeNil)
	}
	if c.CancelFunc == nil {
		return errors.New(messages.CancelFuncCannotBeNil)
	}
	return nil
}

// SetDeferInfo は単一のdefer情報を設定する（既存の情報を置き換え）
func (c *ContextInfo) SetDeferInfo(deferInfo *DeferCancelInfo) {
	if deferInfo != nil {
		c.DeferInfos = []DeferCancelInfo{*deferInfo}
	}
}

// GetDeferInfo は最初のdefer情報を取得する（後方互換性のため）
func (c *ContextInfo) GetDeferInfo() *DeferCancelInfo {
	if len(c.DeferInfos) > 0 {
		return &c.DeferInfos[0]
	}
	return nil
}

// HasDeferInfo は defer情報が設定されているかどうかを判定する
func (c *ContextInfo) HasDeferInfo() bool {
	return len(c.DeferInfos) > 0
}

// AddDeferInfo は新しいdefer情報を追加する（複数対応）
func (c *ContextInfo) AddDeferInfo(deferInfo *DeferCancelInfo) {
	if deferInfo != nil {
		c.DeferInfos = append(c.DeferInfos, *deferInfo)
	}
}

// GetAllDeferInfos は全てのdefer情報を取得する
func (c *ContextInfo) GetAllDeferInfos() []DeferCancelInfo {
	return c.DeferInfos
}

// DeferCancelInfo は defer cancel() 呼び出しの情報を表す
type DeferCancelInfo struct {
	CancelVarName string    // cancel変数名
	DeferPos      token.Pos // defer文の位置
	ScopeDepth    int       // スコープの深さ
	IsValid       bool      // defer文が有効かどうか
}

// NewDeferCancelInfo は DeferCancelInfo のコンストラクタ
func NewDeferCancelInfo(cancelVarName string, deferPos token.Pos, scopeDepth int, isValid bool) *DeferCancelInfo {
	return &DeferCancelInfo{
		CancelVarName: cancelVarName,
		DeferPos:      deferPos,
		ScopeDepth:    scopeDepth,
		IsValid:       isValid,
	}
}

// Validate は DeferCancelInfo の妥当性を検証する
func (d *DeferCancelInfo) Validate() error {
	if d.CancelVarName == "" {
		return errors.New(messages.CancelVarNameCannotBeEmpty)
	}
	if d.DeferPos == 0 {
		return errors.New(messages.DeferPosInvalid)
	}
	return nil
}

// ServiceRule は GCP サービス固有の解放ルール定義を表す
type ServiceRule struct {
	ServiceName    string          `yaml:"service_name"`       // サービス名
	PackagePath    string          `yaml:"package_path"`       // パッケージパス
	CreationFuncs  []string        `yaml:"creation_functions"` // 生成関数一覧
	CleanupMethods []CleanupMethod `yaml:"cleanup_methods"`    // 解放メソッド一覧
}

// CleanupMethod は解放メソッドの詳細情報を表す
type CleanupMethod struct {
	Method      string `yaml:"method"`      // メソッド名
	Required    bool   `yaml:"required"`    // 必須かどうか
	Description string `yaml:"description"` // 説明
}

// HasCreationFunc は指定された関数名が生成関数に含まれるかチェックする
func (s *ServiceRule) HasCreationFunc(funcName string) bool {
	for _, f := range s.CreationFuncs {
		if f == funcName {
			return true
		}
	}
	return false
}

// GetRequiredCleanupMethods は必須の解放メソッド一覧を返す
func (s *ServiceRule) GetRequiredCleanupMethods() []CleanupMethod {
	var required []CleanupMethod
	for _, method := range s.CleanupMethods {
		if method.Required {
			required = append(required, method)
		}
	}
	return required
}

// EscapeInfo は変数の逃げパス（return/field格納）情報を表す
type EscapeInfo struct {
	IsReturned      bool   // 関数戻り値として返されるか
	IsFieldAssigned bool   // 構造体フィールドに代入されるか
	EscapeReason    string // 逃げる理由の説明
}

// NewEscapeInfo は EscapeInfo のコンストラクタ
func NewEscapeInfo(isReturned, isFieldAssigned bool, reason string) *EscapeInfo {
	return &EscapeInfo{
		IsReturned:      isReturned,
		IsFieldAssigned: isFieldAssigned,
		EscapeReason:    reason,
	}
}

// HasEscaped は変数が逃げているかどうかを判定する
func (e *EscapeInfo) HasEscaped() bool {
	return e.IsReturned || e.IsFieldAssigned
}

// Spannerトランザクション種別定数
const (
	ReadWriteTransactionType = "ReadWriteTransaction"
	ReadOnlyTransactionType  = "ReadOnlyTransaction"
)

// SpannerEscapeInfo は Spannerリソースの自動管理情報を表す
type SpannerEscapeInfo struct {
	IsAutoManaged        bool   // フレームワークによる自動管理かどうか
	TransactionType      string // トランザクション種別（ReadWriteTransaction/ReadOnlyTransaction）
	IsClosureManaged     bool   // クロージャ内で管理されているか
	ClosureDetected      bool   // クロージャパターンが検出されたか
	AutoManagementReason string // 自動管理の理由
}

// NewSpannerEscapeInfo は SpannerEscapeInfo のコンストラクタ
func NewSpannerEscapeInfo(transactionType string, isAutoManaged bool, reason string) *SpannerEscapeInfo {
	// トランザクション種別の検証
	if transactionType != ReadWriteTransactionType && transactionType != ReadOnlyTransactionType {
		transactionType = ReadWriteTransactionType // デフォルト値
	}

	return &SpannerEscapeInfo{
		IsAutoManaged:        isAutoManaged,
		TransactionType:      transactionType,
		IsClosureManaged:     isAutoManaged, // 自動管理の場合はクロージャ管理と見なす
		ClosureDetected:      isAutoManaged, // 自動管理の場合はクロージャ検出済み
		AutoManagementReason: reason,
	}
}

// Validate は SpannerEscapeInfo の妥当性を検証する
func (s *SpannerEscapeInfo) Validate() error {
	if s.TransactionType != ReadWriteTransactionType && s.TransactionType != ReadOnlyTransactionType {
		return errors.New(messages.TransactionTypeMustBeValid)
	}
	if s.IsAutoManaged && s.AutoManagementReason == "" {
		return errors.New(messages.AutoManagementReasonRequired)
	}
	return nil
}

// ShouldSkipCleanup は、このSpannerリソースの解放処理をスキップすべきかどうかを判定する
func (s *SpannerEscapeInfo) ShouldSkipCleanup() bool {
	return s.IsAutoManaged
}
