module spanner_transaction_valid

go 1.21

require (
	cloud.google.com/go/spanner v1.45.0
)

// 依存関係は実際のテスト実行では使用されない（AST解析のみ）
replace cloud.google.com/go/spanner => ./fake_spanner