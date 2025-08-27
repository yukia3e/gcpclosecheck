package recaptcha_valid

import (
	"context"
	recaptcha "cloud.google.com/go/recaptchaenterprise/v2/apiv1"
)

// reCAPTCHAクライアントの正しい使用パターン
func correctRecaptchaClient(ctx context.Context) error {
	client, err := recaptcha.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // 正しいパターン

	return nil
}

// reCAPTCHA Beta クライアントの正しい使用パターン
func correctRecaptchaBetaClient(ctx context.Context) error {
	client, err := recaptcha.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // 正しいパターン

	return nil
}

// 複合パターン
func correctComplexRecaptcha(ctx context.Context) error {
	client, err := recaptcha.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// reCAPTCHA操作の実行
	// req := &recaptchapb.CreateAssessmentRequest{...}
	// _, err = client.CreateAssessment(ctx, req)
	
	return nil
}

// 戻り値として返される場合
func createRecaptchaClient(ctx context.Context) (*recaptcha.Client, error) {
	client, err := recaptcha.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil // 戻り値として返されるのでdefer不要
}