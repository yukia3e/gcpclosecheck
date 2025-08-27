package pubsub_valid

import (
	"context"
	"cloud.google.com/go/pubsub"
)

// PubSubクライアントの正しい使用パターン
func correctPubSubClient(ctx context.Context) error {
	client, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	defer client.Close() // 正しいパターン

	return nil
}

// トピックの正しい使用パターン（戻り値として返される）
func createTopic(ctx context.Context, client *pubsub.Client) *pubsub.Topic {
	topic := client.Topic("test-topic")
	return topic // 戻り値として返されるのでdefer不要
}

// サブスクリプションの正しい使用パターン（戻り値として返される）
func createSubscription(ctx context.Context, client *pubsub.Client) *pubsub.Subscription {
	sub := client.Subscription("test-subscription")
	return sub // 戻り値として返されるのでdefer不要
}

// Receiveの正しい使用パターン（contextでキャンセル）
func correctReceiveWithCancel(ctx context.Context, sub *pubsub.Subscription) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // 正しいパターン

	err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		// メッセージ処理
		m.Ack()
	})

	return err
}

// 複合パターン
func correctComplexPubSub(ctx context.Context) error {
	client, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	topic := client.Topic("test-topic")
	sub := client.Subscription("test-subscription")

	// パブリッシュ
	result := topic.Publish(ctx, &pubsub.Message{Data: []byte("test")})
	_, err = result.Get(ctx)
	if err != nil {
		return err
	}

	// サブスクライブ
	err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		m.Ack()
	})

	return err
}