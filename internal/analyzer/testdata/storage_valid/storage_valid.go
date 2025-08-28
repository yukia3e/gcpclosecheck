package storage_valid

import (
	"context"
	"cloud.google.com/go/storage"
)

// Storageクライアントの正しい使用パターン
func correctStorageClient(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // 正しいパターン

	return nil
}

// Readerの正しい使用パターン
func correctStorageReader(ctx context.Context, client *storage.Client) error {
	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")
	
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return err
	}
	defer reader.Close() // 正しいパターン

	return nil
}

// Writerの正しい使用パターン
func correctStorageWriter(ctx context.Context, client *storage.Client) error {
	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")
	
	writer := obj.NewWriter(ctx)
	defer writer.Close() // 正しいパターン

	return nil
}

// 複合パターン
func correctComplexStorage(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")

	// Read
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Write
	writer := obj.NewWriter(ctx)
	defer writer.Close()

	return nil
}

// バケットハンドル（戻り値として返される）
func getBucket(client *storage.Client) *storage.BucketHandle {
	bucket := client.Bucket("test-bucket")
	return bucket // 戻り値として返されるのでdefer不要
}

// オブジェクトハンドル（戻り値として返される）
func getObject(bucket *storage.BucketHandle) *storage.ObjectHandle {
	obj := bucket.Object("test-object")
	return obj // 戻り値として返されるのでdefer不要
}