package recaptcha_invalid

import (
	"context"
	recaptcha "cloud.google.com/go/recaptchaenterprise/v2/apiv1"
)

// reCAPTCHAクライアントのClose不足
func missingClientClose(ctx context.Context) error {
	client, err := recaptcha.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	return nil
}

// 複数のreCAPTCHAクライアントでのClose不足
func multipleClientsMissingClose(ctx context.Context) error {
	client1, err := recaptcha.NewClient(ctx) // want "GCP リソース 'client1' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client1.Close() が不足

	client2, err := recaptcha.NewClient(ctx) // want "GCP リソース 'client2' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client2.Close() が不足

	return nil
}

// 一部のクライアントのみClose不足
func partialClientClose(ctx context.Context) error {
	client1, err := recaptcha.NewClient(ctx) // want "GCP リソース 'client1' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client1.Close() が不足

	client2, err := recaptcha.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client2.Close() // これは正しい

	return nil
}

// ネストした関数でのClose不足
func nestedFunctionMissingClose(ctx context.Context) error {
	func() {
		client, _ := recaptcha.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
		// defer client.Close() が不足
		_ = client
	}()

	return nil
}

// エラーハンドリング後のClose不足
func errorHandlingMissingClose(ctx context.Context) error {
	client, err := recaptcha.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err // エラー時にCloseが呼ばれない
	}
	// defer client.Close() が不足

	return nil
}