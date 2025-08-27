package vision_valid

import (
	"context"
	vision "cloud.google.com/go/vision/apiv1"
)

// Visionクライアントの正しい使用パターン
func correctVisionClient(ctx context.Context) error {
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // 正しいパターン

	return nil
}

// ProductSearchクライアントの正しい使用パターン
func correctProductSearchClient(ctx context.Context) error {
	client, err := vision.NewProductSearchClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // 正しいパターン

	return nil
}

// 複合パターン
func correctComplexVision(ctx context.Context) error {
	imageClient, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	defer imageClient.Close()

	productClient, err := vision.NewProductSearchClient(ctx)
	if err != nil {
		return err
	}
	defer productClient.Close()

	return nil
}

// 戻り値として返される場合
func createVisionClient(ctx context.Context) (*vision.ImageAnnotatorClient, error) {
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil // 戻り値として返されるのでdefer不要
}