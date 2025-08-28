package complex_valid

import (
	"context"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
)

// 複数のGCPサービスを組み合わせた正しいパターン
func correctMultiServicePattern(ctx context.Context) error {
	// Storage
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer storageClient.Close()

	// PubSub
	pubsubClient, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	defer pubsubClient.Close()

	// Spanner
	spannerClient, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		return err
	}
	defer spannerClient.Close()

	// Context with cancel
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return nil
}

// エラーハンドリングを含む正しいパターン
func correctErrorHandling(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // エラー時でも呼ばれる

	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return err // client.Close()は呼ばれる
	}
	defer reader.Close() // エラー時でも呼ばれる

	return nil
}

// インターフェース経由での正しい使用パターン
type ResourceManager interface {
	Close() error
}

func correctInterfaceUsage(ctx context.Context) error {
	var resource ResourceManager

	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	resource = client
	defer resource.Close() // インターフェース経由でも正しい

	return nil
}

// 関数の引数として渡される場合の正しいパターン
func useStorageClient(client *storage.Client) error {
	// 引数として受け取ったクライアントは呼び出し元が責任を持つ
	bucket := client.Bucket("test-bucket")
	_ = bucket
	return nil
}

func correctParameterPattern(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // 呼び出し元が責任を持つ

	return useStorageClient(client)
}

// 条件分岐を含む正しいパターン
func correctConditionalPattern(ctx context.Context, useSpanner bool) error {
	// 共通リソース
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer storageClient.Close()

	// 条件付きリソース
	if useSpanner {
		spannerClient, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
		if err != nil {
			return err
		}
		defer spannerClient.Close() // 条件内でも正しい
	}

	return nil
}
