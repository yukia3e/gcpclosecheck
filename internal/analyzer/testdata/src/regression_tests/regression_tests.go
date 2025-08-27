package regression_tests

import (
	"context"
	"cloud.google.com/go/storage"
)

// 回帰テスト - 以前に発見された問題パターン
func regressionTest1(ctx context.Context) error {
	client, err := storage.NewClient(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// 以前のバージョンで見逃されていたパターン
	return nil
}

func regressionTest2(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	// 正しいパターン
	return nil
}