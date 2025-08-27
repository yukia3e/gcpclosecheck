package testdata

import (
	"context"

	vision "cloud.google.com/go/vision/apiv1"
)

// Vision APIクライアントのクローズが漏れている例
func VisionMissingClose(ctx context.Context) error { // want `vision client not properly closed`
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	// defer client.Close() が漏れている！

	_ = client
	return nil
}

// 複数のVision APIクライアントでクローズ漏れ
func VisionMultipleMissingClose(ctx context.Context) error { // want multiple errors
	client1, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	// defer client1.Close() が漏れている！

	client2, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	// defer client2.Close() が漏れている！

	_, _ = client1, client2
	return nil
}

// 処理中にクローズ漏れ
func VisionProcessingMissingClose(ctx context.Context) error { // want `vision client not properly closed`
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	// defer client.Close() が漏れている！

	// 実際の画像解析処理（詳細は省略）
	_ = client
	
	return nil
}