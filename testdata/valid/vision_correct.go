package testdata

import (
	"context"

	vision "cloud.google.com/go/vision/apiv1"
)

// 正常なVision APIクライアントの使用例
func VisionCorrectUsage(ctx context.Context) error {
	// Vision APIクライアントを作成
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // 正しくクローズ処理

	// 画像を解析（実際の処理は省略）
	return nil
}

// 複数のVision APIクライアント
func VisionMultipleClients(ctx context.Context) error {
	client1, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	defer client1.Close()

	client2, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	defer client2.Close()

	return nil
}

// 関数で返されるVisionクライアント（追跡対象外）
func GetVisionClient(ctx context.Context) (*vision.ImageAnnotatorClient, error) {
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, err
	}
	// 戻り値として返されるため、この関数内でCloseする必要はない
	return client, nil
}