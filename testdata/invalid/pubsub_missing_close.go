package testdata

import (
	"context"

	"cloud.google.com/go/pubsub"
)

// Pub/Subクライアントのクローズが漏れている例
func PubSubMissingClose(ctx context.Context) error { // want `pubsub client not properly closed`
	client, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	// defer client.Close() が漏れている！

	_ = client
	return nil
}

// 複数のPub/Subクライアントでクローズ漏れ
func PubSubMultipleMissingClose(ctx context.Context) error { // want multiple errors
	client1, err := pubsub.NewClient(ctx, "test-project-1")
	if err != nil {
		return err
	}
	// defer client1.Close() が漏れている！

	client2, err := pubsub.NewClient(ctx, "test-project-2")
	if err != nil {
		return err
	}
	// defer client2.Close() が漏れている！

	_, _ = client1, client2
	return nil
}

// メッセージ処理でのクローズ漏れ
func PubSubMessageProcessingMissingClose(ctx context.Context) error { // want `pubsub client not properly closed`
	client, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	// defer client.Close() が漏れている！

	topic := client.Topic("test-topic")
	result := topic.Publish(ctx, &pubsub.Message{
		Data: []byte("test message"),
	})

	_, err = result.Get(ctx)
	return err
}
