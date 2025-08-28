package spanner_invalid

import (
	"context"

	"cloud.google.com/go/spanner"
)

// スパナークライアントのClose不足
func missingClientClose(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test") // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	return nil
}

// スパナートランザクションのClose不足
func missingTransactionClose(ctx context.Context, client *spanner.Client) error {
	txn := client.ReadOnlyTransaction() // want "GCP リソース 'txn' の解放処理 \\(Close\\) が見つかりません"
	// defer txn.Close() が不足

	return nil
}

// スパナーイテレーターのStop不足
func missingIteratorStop(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
	stmt := spanner.NewStatement("SELECT * FROM test")
	iter := txn.Query(ctx, stmt) // want "GCP リソース 'iter' の解放処理 \\(Stop\\) が見つかりません"
	// defer iter.Stop() が不足

	return nil
}

// 複数リソースで一部のClose不足
func partialResourceClose(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test") // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	txn := client.ReadOnlyTransaction()
	defer txn.Close() // これは正しい

	return nil
}

// ネストした関数でのClose不足
func nestedFunctionMissingClose(ctx context.Context) error {
	func() {
		client, _ := spanner.NewClient(ctx, "projects/test/instances/test/databases/test") // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
		// defer client.Close() が不足
		_ = client
	}()

	return nil
}

// エラーハンドリング後のClose不足
func errorHandlingMissingClose(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test") // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err // エラー時にCloseが呼ばれない
	}
	// defer client.Close() が不足

	return nil
}
