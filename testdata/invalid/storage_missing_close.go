package testdata

import (
	"context"

	"cloud.google.com/go/storage"
)

// Storageクライアントのクローズが漏れている例
func StorageMissingClose(ctx context.Context) error { // want `storage client not properly closed`
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	// defer client.Close() が漏れている！

	_ = client
	return nil
}

// Readerのクローズが漏れている例
func StorageReaderMissingClose(ctx context.Context) error { // want `storage reader not properly closed`
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // クライアントはクローズされている

	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return err
	}
	// defer reader.Close() が漏れている！

	_ = reader
	return nil
}

// Writerのクローズが漏れている例
func StorageWriterMissingClose(ctx context.Context) error { // want `storage writer not properly closed`
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	bucket := client.Bucket("test-bucket")
	obj := bucket.Object("test-object")

	writer := obj.NewWriter(ctx)
	// defer writer.Close() が漏れている！

	_ = writer
	return nil
}

// 複数のStorageリソースでクローズ漏れ
func StorageMultipleMissingClose(ctx context.Context) error { // want multiple errors
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	// defer client.Close() が漏れている！

	bucket := client.Bucket("test-bucket")

	reader1, err := bucket.Object("object1").NewReader(ctx)
	if err != nil {
		return err
	}
	// defer reader1.Close() が漏れている！

	reader2, err := bucket.Object("object2").NewReader(ctx)
	if err != nil {
		return err
	}
	// defer reader2.Close() が漏れている！

	_, _ = reader1, reader2
	return nil
}
