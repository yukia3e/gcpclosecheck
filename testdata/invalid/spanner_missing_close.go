package testdata

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Spannerクライアントのクローズが漏れている例
func SpannerMissingClose(ctx context.Context) error { // want `spanner client not properly closed`
	// Spannerクライアントを作成
	client, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		return err
	}
	// defer client.Close() が漏れている！

	// 何らかの処理
	_ = client
	return nil
}

// ReadOnlyTransactionのクローズが漏れている例
func SpannerTransactionMissingClose(ctx context.Context) error { // want `spanner transaction not properly closed`
	client, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		return err
	}
	defer client.Close() // クライアントはクローズされている

	// ReadOnlyTransactionを作成
	txn := client.ReadOnlyTransaction()
	// defer txn.Close() が漏れている！

	_ = txn
	return nil
}

// RowIteratorのStopが漏れている例
func SpannerIteratorMissingStop(ctx context.Context) error { // want `spanner iterator not properly stopped`
	client, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		return err
	}
	defer client.Close()

	txn := client.ReadOnlyTransaction()
	defer txn.Close()

	// クエリ実行
	iter := txn.Query(ctx, spanner.NewStatement("SELECT * FROM test_table"))
	// defer iter.Stop() が漏れている！

	_ = iter
	return nil
}

// 複数のクローズ漏れ
func SpannerMultipleMissingClose(ctx context.Context) error { // want multiple errors
	client1, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db1")
	if err != nil {
		return err
	}
	// defer client1.Close() が漏れている！

	client2, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db2")
	if err != nil {
		return err
	}
	// defer client2.Close() が漏れている！

	_, _ = client1, client2
	return nil
}
