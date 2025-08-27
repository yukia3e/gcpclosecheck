package storage_invalid

import (
	"context"
	"cloud.google.com/go/storage"
)

// StorageクライアントのClose不足
func missingClientClose(ctx context.Context) error {
	client, err := storage.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	return nil
}

// ReaderのClose不足
func missingReaderClose(ctx context.Context, client *storage.Client) error {
	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")
	
	reader, err := obj.NewReader(ctx) // want "GCP リソース 'reader' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer reader.Close() が不足

	return nil
}

// WriterのClose不足
func missingWriterClose(ctx context.Context, client *storage.Client) error {
	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")
	
	writer := obj.NewWriter(ctx) // want "GCP リソース 'writer' の解放処理 \\(Close\\) が見つかりません"
	// defer writer.Close() が不足

	return nil
}

// 複数リソースで一部のClose不足
func partialResourceClose(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // これは正しい

	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")

	reader, err := obj.NewReader(ctx) // want "GCP リソース 'reader' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer reader.Close() が不足

	return nil
}

// ネストした関数でのClose不足
func nestedFunctionMissingClose(ctx context.Context) error {
	func() {
		client, _ := storage.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
		// defer client.Close() が不足
		_ = client
	}()

	return nil
}

// エラーハンドリング後のClose不足
func errorHandlingMissingClose(ctx context.Context) error {
	client, err := storage.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err // エラー時にCloseが呼ばれない
	}
	// defer client.Close() が不足

	return nil
}