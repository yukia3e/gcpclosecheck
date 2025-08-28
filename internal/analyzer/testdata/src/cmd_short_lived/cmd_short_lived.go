package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
)

// main関数でのクライアント生成 - 短命プログラムなので例外対象
func main() {
	ctx := context.Background()

	// Spanner Client - パッケージ例外により診断除外される
	spannerClient, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		log.Fatal("Spanner client creation failed:", err)
	}
	// defer spannerClient.Close() 不要（例外により診断されない）

	// Storage Client - パッケージ例外により診断除外される
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal("Storage client creation failed:", err)
	}
	// defer storageClient.Close() 不要（例外により診断されない）

	// BigQuery Client - パッケージ例外により診断除外される
	bqClient, err := bigquery.NewClient(ctx, "test-project")
	if err != nil {
		log.Fatal("BigQuery client creation failed:", err)
	}
	// defer bqClient.Close() 不要（例外により診断されない）

	// Pub/Sub Client - パッケージ例外により診断除外される
	pubsubClient, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		log.Fatal("Pub/Sub client creation failed:", err)
	}
	// defer pubsubClient.Close() 不要（例外により診断されない）

	// 短時間の処理を実行
	processData(spannerClient, storageClient, bqClient, pubsubClient)

	// プログラム終了時に自動的にリソースが解放される
	os.Exit(0)
}

func processData(spannerClient *spanner.Client, storageClient *storage.Client,
	bqClient *bigquery.Client, pubsubClient *pubsub.Client) {
	// 実際の処理をシミュレート
	log.Println("Processing data with clients...")

	// 短時間の処理後、プログラム終了
	// リソースは OS により自動的に解放される
}
