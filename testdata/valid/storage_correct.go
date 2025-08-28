package testdata

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
)

// 正常なStorageクライアントの使用例
func StorageCorrectUsage(ctx context.Context) error {
	// Storageクライアントを作成
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // 正しくクローズ処理

	// バケットからオブジェクトを読み取り
	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return err
	}
	defer reader.Close() // 正しくクローズ処理

	// データを読み取り
	_, err = io.ReadAll(reader)
	return err
}

// オブジェクトの書き込み例
func StorageWriteCorrectUsage(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")

	writer := obj.NewWriter(ctx)
	defer writer.Close() // 正しくクローズ処理

	// データを書き込み
	_, err = writer.Write([]byte("test data"))
	return err
}

// 複数のStorageリーダー/ライター
func StorageMultipleResources(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	bucket := client.Bucket("test-bucket")

	// 複数のリーダー
	reader1, err := bucket.Object("object1").NewReader(ctx)
	if err != nil {
		return err
	}
	defer reader1.Close()

	reader2, err := bucket.Object("object2").NewReader(ctx)
	if err != nil {
		return err
	}
	defer reader2.Close()

	return nil
}
