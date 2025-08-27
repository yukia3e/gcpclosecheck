package testdata

import (
	"context"

	"cloud.google.com/go/spanner"
)

// 正常なSpannerクライアントの使用例
func SpannerCorrectUsage(ctx context.Context) error {
	// Spannerクライアントを作成
	client, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		return err
	}
	defer client.Close() // 正しくクローズ処理

	// ReadOnlyTransactionの使用
	txn := client.ReadOnlyTransaction()
	defer txn.Close() // 正しくクローズ処理

	// クエリ実行
	iter := txn.Query(ctx, spanner.NewStatement("SELECT * FROM test_table"))
	defer iter.Stop() // 正しく停止処理

	// 結果を処理
	return iter.Do(func(row *spanner.Row) error {
		// 行の処理
		return nil
	})
}

// 複数のSpannerリソースを使用する例
func SpannerMultipleResources(ctx context.Context) error {
	client1, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db1")
	if err != nil {
		return err
	}
	defer client1.Close()

	client2, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db2")
	if err != nil {
		return err
	}
	defer client2.Close()

	return nil
}

// 関数で返されるリソース（追跡対象外）
func GetSpannerClient(ctx context.Context) (*spanner.Client, error) {
	client, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		return nil, err
	}
	// 戻り値として返されるため、この関数内でCloseする必要はない
	return client, nil
}