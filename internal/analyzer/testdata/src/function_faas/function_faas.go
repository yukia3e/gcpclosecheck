package function

import (
	"context"
	"fmt"
	"net/http"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
)

// Cloud Functions HTTP トリガー関数
// function パッケージパスなので例外対象
func HandleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Spanner Client - Cloud Functions例外により診断除外される
	spannerClient, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		http.Error(w, "Spanner client creation failed", http.StatusInternalServerError)
		return
	}
	// defer spannerClient.Close() 不要（例外により診断されない）

	// Storage Client - Cloud Functions例外により診断除外される
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		http.Error(w, "Storage client creation failed", http.StatusInternalServerError)
		return
	}
	// defer storageClient.Close() 不要（例外により診断されない）

	// Firestore Client - Cloud Functions例外により診断除外される
	firestoreClient, err := firestore.NewClient(ctx, "test-project")
	if err != nil {
		http.Error(w, "Firestore client creation failed", http.StatusInternalServerError)
		return
	}
	// defer firestoreClient.Close() 不要（例外により診断されない）

	// 処理を実行
	result := processCloudFunctionRequest(ctx, spannerClient, storageClient, firestoreClient)

	fmt.Fprintf(w, "Processing result: %s", result)
	// Cloud Functions の実行終了時にランタイムがリソースを管理
}

// Cloud Functions Pub/Sub トリガー関数
func HandlePubSubMessage(ctx context.Context, msg PubSubMessage) error {
	// Pub/Sub Client - Cloud Functions例外により診断除外される
	pubsubClient, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return fmt.Errorf("Pub/Sub client creation failed: %v", err)
	}
	// defer pubsubClient.Close() 不要（例外により診断されない）

	// Spanner Client for background processing
	spannerClient, err := spanner.NewClient(ctx, "projects/test-project/instances/test-instance/databases/test-db")
	if err != nil {
		return fmt.Errorf("Spanner client creation failed: %v", err)
	}
	// defer spannerClient.Close() 不要（例外により診断されない）

	// バックグラウンド処理を実行
	return processBackgroundTask(ctx, pubsubClient, spannerClient, msg)
}

// PubSubMessage represents a Pub/Sub message
type PubSubMessage struct {
	Data       []byte            `json:"data"`
	Attributes map[string]string `json:"attributes"`
}

func processCloudFunctionRequest(ctx context.Context, spannerClient *spanner.Client,
	storageClient *storage.Client, firestoreClient *firestore.Client) string {
	// Cloud Functions内での処理をシミュレート
	return "Cloud Functions processing completed"
}

func processBackgroundTask(ctx context.Context, pubsubClient *pubsub.Client,
	spannerClient *spanner.Client, msg PubSubMessage) error {
	// バックグラウンドタスクの処理をシミュレート
	return nil
}
