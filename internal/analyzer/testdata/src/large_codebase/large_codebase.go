package large_codebase

import (
	"context"
	"cloud.google.com/go/storage"
	"cloud.google.com/go/pubsub"
)

// 大規模コードベースのテスト用パフォーマンスファイル
// パフォーマンステスト用に多数の関数を含む

func function1(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	return nil
}

func function2(ctx context.Context) error {
	client, err := storage.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足
	return nil
}

func function3(ctx context.Context) error {
	client, err := pubsub.NewClient(ctx, "project")
	if err != nil {
		return err
	}
	defer client.Close()
	return nil
}

func function4(ctx context.Context) error {
	client, err := pubsub.NewClient(ctx, "project") // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足
	return nil
}