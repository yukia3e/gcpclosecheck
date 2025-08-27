package complex_invalid

import (
	"context"
	"cloud.google.com/go/storage"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
)

// 複数のGCPサービスで一部のClose不足
func multiServicePartialClose(ctx context.Context) error {
	// Storage（Close不足）
	storageClient, err := storage.NewClient(ctx) // want "GCP リソース 'storageClient' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer storageClient.Close() が不足

	// PubSub（正しい）
	pubsubClient, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	defer pubsubClient.Close()

	// Spanner（Close不足）
	spannerClient, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test") // want "GCP リソース 'spannerClient' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer spannerClient.Close() が不足

	return nil
}

// エラーハンドリング内でのClose不足
func errorHandlingMissingClose(ctx context.Context) error {
	client, err := storage.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")

	reader, err := obj.NewReader(ctx) // want "GCP リソース 'reader' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err // clientもreaderもCloseされない
	}
	// defer reader.Close() が不足

	return nil
}

// 条件分岐内でのClose不足
func conditionalMissingClose(ctx context.Context, useSpanner bool) error {
	// 共通リソース（Close不足）
	storageClient, err := storage.NewClient(ctx) // want "GCP リソース 'storageClient' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer storageClient.Close() が不足

	// 条件付きリソース（Close不足）
	if useSpanner {
		spannerClient, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test") // want "GCP リソース 'spannerClient' の解放処理 \\(Close\\) が見つかりません"
		if err != nil {
			return err
		}
		// defer spannerClient.Close() が不足
	}

	return nil
}

// ループ内でのClose不足
func loopMissingClose(ctx context.Context) error {
	databases := []string{"db1", "db2", "db3"}
	
	for _, db := range databases {
		client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/"+db) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
		if err != nil {
			continue
		}
		// defer client.Close() が不足（ループ内で累積してしまう）
		_ = client
	}

	return nil
}

// goroutine内でのClose不足
func goroutineMissingClose(ctx context.Context) error {
	go func() {
		client, err := storage.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
		if err != nil {
			return
		}
		// defer client.Close() が不足
		_ = client
	}()

	return nil
}

// 複雑なネストでのClose不足
func complexNestedMissingClose(ctx context.Context) error {
	func() {
		if true {
			for i := 0; i < 1; i++ {
				client, _ := storage.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
				// defer client.Close() が不足
				_ = client
			}
		}
	}()

	return nil
}

// インターフェース経由でのClose不足
type ResourceManager interface {
	Close() error
}

func interfaceMissingClose(ctx context.Context) error {
	var resource ResourceManager
	
	client, err := storage.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	resource = client
	// defer resource.Close() が不足

	_ = resource
	return nil
}