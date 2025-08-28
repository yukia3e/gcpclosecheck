package vision_invalid

import (
	"context"

	vision "cloud.google.com/go/vision/apiv1"
)

// VisionクライアントのClose不足
func missingClientClose(ctx context.Context) error {
	client, err := vision.NewImageAnnotatorClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	return nil
}

// ProductSearchクライアントのClose不足
func missingProductSearchClose(ctx context.Context) error {
	client, err := vision.NewProductSearchClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	return nil
}

// 複数リソースで一部のClose不足
func partialResourceClose(ctx context.Context) error {
	imageClient, err := vision.NewImageAnnotatorClient(ctx) // want "GCP リソース 'imageClient' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer imageClient.Close() が不足

	productClient, err := vision.NewProductSearchClient(ctx)
	if err != nil {
		return err
	}
	defer productClient.Close() // これは正しい

	return nil
}

// ネストした関数でのClose不足
func nestedFunctionMissingClose(ctx context.Context) error {
	func() {
		client, _ := vision.NewImageAnnotatorClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
		// defer client.Close() が不足
		_ = client
	}()

	return nil
}

// エラーハンドリング後のClose不足
func errorHandlingMissingClose(ctx context.Context) error {
	client, err := vision.NewImageAnnotatorClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err // エラー時にCloseが呼ばれない
	}
	// defer client.Close() が不足

	return nil
}
