package spanner_valid

import (
	"context"

	"cloud.google.com/go/spanner"
)

// スパナークライアントの正しい使用パターン
func correctSpannerClient(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		return err
	}
	defer client.Close() // 正しいパターン

	return nil
}

// スパナートランザクションの正しい使用パターン
func correctSpannerTransaction(ctx context.Context, client *spanner.Client) error {
	txn := client.ReadOnlyTransaction()
	defer txn.Close() // 正しいパターン

	return nil
}

// スパナーイテレーターの正しい使用パターン
func correctSpannerIterator(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
	stmt := spanner.NewStatement("SELECT * FROM test")
	iter := txn.Query(ctx, stmt)
	defer iter.Stop() // 正しいパターン

	return nil
}

// 複数リソースの正しい使用パターン
func correctMultipleResources(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		return err
	}
	defer client.Close()

	txn := client.ReadOnlyTransaction()
	defer txn.Close()

	stmt := spanner.NewStatement("SELECT * FROM test")
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	return nil
}

// 戻り値として返されるリソース（deferが不要）
func createClient(ctx context.Context) (*spanner.Client, error) {
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		return nil, err
	}
	return client, nil // 戻り値として返されるのでdefer不要
}

// 構造体フィールドに格納されるリソース（deferが不要）
type SpannerService struct {
	client *spanner.Client
}

func (s *SpannerService) init(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		return err
	}
	s.client = client // フィールドに格納されるのでdefer不要
	return nil
}
