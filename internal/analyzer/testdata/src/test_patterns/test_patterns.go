package testpatterns

import (
	"context"
	"testing"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
)

// テストコード内でのクライアント利用パターン
// パッケージパス pattern **/*_test.go はデフォルト無効なので診断される

func TestSpannerClientUsage(t *testing.T) {
	ctx := context.Background()

	// Spanner Client - テスト例外が無効なので診断される
	client, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		t.Fatalf("Failed to create Spanner client: %v", err)
	}
	// defer client.Close() が必要 - 診断されるべき

	// テスト用のクエリ実行
	txn := client.ReadOnlyTransaction()
	// defer txn.Close() が必要 - 診断されるべき

	iter := txn.Query(ctx, spanner.NewStatement("SELECT 1"))
	// defer iter.Stop() が必要 - 診断されるべき

	// テスト処理をシミュレート
	_ = iter
	t.Log("Test completed")
}

func TestStorageClientUsage(t *testing.T) {
	ctx := context.Background()

	// Storage Client - テスト例外が無効なので診断される
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create Storage client: %v", err)
	}
	// defer client.Close() が必要 - 診断されるべき

	// テスト用のバケット操作
	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")

	// テスト処理をシミュレート
	_ = obj
	t.Log("Storage test completed")
}

func TestBigQueryClientUsage(t *testing.T) {
	ctx := context.Background()

	// BigQuery Client - テスト例外が無効なので診断される
	client, err := bigquery.NewClient(ctx, "test-project")
	if err != nil {
		t.Fatalf("Failed to create BigQuery client: %v", err)
	}
	// defer client.Close() が必要 - 診断されるべき

	// テスト用のクエリ実行
	query := client.Query("SELECT 1")
	job, err := query.Run(ctx)
	if err != nil {
		t.Fatalf("Failed to run query: %v", err)
	}

	// テスト処理をシミュレート
	_ = job
	t.Log("BigQuery test completed")
}

func TestPubSubClientUsage(t *testing.T) {
	ctx := context.Background()

	// Pub/Sub Client - テスト例外が無効なので診断される
	client, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		t.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	// defer client.Close() が必要 - 診断されるべき

	// テスト用のトピック操作
	topic := client.Topic("test-topic")

	// テスト処理をシミュレート
	_ = topic
	t.Log("Pub/Sub test completed")
}

// ベンチマークテストでのクライアント利用
func BenchmarkSpannerQuery(b *testing.B) {
	ctx := context.Background()

	// Spanner Client - テスト例外が無効なので診断される
	client, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		b.Fatalf("Failed to create Spanner client: %v", err)
	}
	// defer client.Close() が必要 - 診断されるべき

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		txn := client.ReadOnlyTransaction()
		// defer txn.Close() が必要 - 診断されるべき

		// ベンチマーク処理をシミュレート
		_ = txn
	}
}

// ヘルパー関数内でのクライアント作成（テストコード用）
func setupTestClients(t *testing.T) (*spanner.Client, *storage.Client) {
	ctx := context.Background()

	// Spanner Client - テスト例外が無効なので診断される
	spannerClient, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		t.Fatalf("Failed to create Spanner client: %v", err)
	}
	// defer spannerClient.Close() が必要 - 診断されるべき

	// Storage Client - テスト例外が無効なので診断される
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create Storage client: %v", err)
	}
	// defer storageClient.Close() が必要 - 診断されるべき

	return spannerClient, storageClient
}
