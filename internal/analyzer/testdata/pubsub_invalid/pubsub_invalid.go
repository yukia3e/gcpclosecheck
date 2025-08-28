package pubsub_invalid

import (
	"context"

	"cloud.google.com/go/pubsub"
)

// PubSubクライアントのClose不足
func missingClientClose(ctx context.Context) error {
	client, err := pubsub.NewClient(ctx, "test-project") // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	return nil
}

// context.WithCancelのcancel不足
func missingContextCancel(ctx context.Context, sub *pubsub.Subscription) error {
	ctx, cancel := context.WithCancel(ctx) // want "context.WithCancel のキャンセル関数 'cancel' の呼び出しが見つかりません"
	// defer cancel() が不足

	err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		m.Ack()
	})

	return err
}

// 複数リソースで一部のClose/Cancel不足
func partialResourceClose(ctx context.Context) error {
	client, err := pubsub.NewClient(ctx, "test-project") // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // これは正しい

	return nil
}

// ネストした関数でのClose不足
func nestedFunctionMissingClose(ctx context.Context) error {
	func() {
		client, _ := pubsub.NewClient(ctx, "test-project") // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
		// defer client.Close() が不足
		_ = client
	}()

	return nil
}

// エラーハンドリング後のClose不足
func errorHandlingMissingClose(ctx context.Context) error {
	client, err := pubsub.NewClient(ctx, "test-project") // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err // エラー時にCloseが呼ばれない
	}
	// defer client.Close() が不足

	return nil
}
