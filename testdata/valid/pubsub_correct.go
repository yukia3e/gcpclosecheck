package testdata

import (
	"context"

	"cloud.google.com/go/pubsub"
)

// 正常なPub/Subクライアントの使用例
func PubSubCorrectUsage(ctx context.Context) error {
	// Pub/Subクライアントを作成
	client, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	defer client.Close() // 正しくクローズ処理

	// トピックを取得
	topic := client.Topic("test-topic")

	// メッセージを発行
	result := topic.Publish(ctx, &pubsub.Message{
		Data: []byte("test message"),
	})

	// 結果を待機
	_, err = result.Get(ctx)
	return err
}

// サブスクリプションの使用例
func PubSubSubscriptionCorrectUsage(ctx context.Context) error {
	client, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	defer client.Close()

	// サブスクリプションを取得
	sub := client.Subscription("test-subscription")

	// メッセージを受信
	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// メッセージ処理
		msg.Ack()
	})

	return err
}

// 複数のPub/Subリソース
func PubSubMultipleResources(ctx context.Context) error {
	client1, err := pubsub.NewClient(ctx, "test-project-1")
	if err != nil {
		return err
	}
	defer client1.Close()

	client2, err := pubsub.NewClient(ctx, "test-project-2")
	if err != nil {
		return err
	}
	defer client2.Close()

	return nil
}
