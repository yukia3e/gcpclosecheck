package memory_intensive

import (
	"context"

	"cloud.google.com/go/storage"
)

// メモリ集約的なテストパターン
func memoryIntensiveFunction(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// メモリ使用量テスト用の処理
	for i := 0; i < 1000; i++ {
		bucket := client.Bucket("test-bucket")
		_ = bucket
	}

	return nil
}
