package testdata

import (
	"context"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
)

// 構造体のフィールドに格納される場合（追跡対象外）
type ServiceClient struct {
	SpannerClient *spanner.Client
	StorageClient *storage.Client
}

func NewServiceClient(ctx context.Context) (*ServiceClient, error) {
	spannerClient, err := spanner.NewClient(ctx, "projects/test")
	if err != nil {
		return nil, err
	}
	// 構造体フィールドに格納されるため、ここでCloseする必要はない

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		spannerClient.Close() // エラー時のクリーンアップ
		return nil, err
	}
	// 構造体フィールドに格納されるため、ここでCloseする必要はない

	return &ServiceClient{
		SpannerClient: spannerClient,
		StorageClient: storageClient,
	}, nil
}

func (sc *ServiceClient) Close() {
	if sc.SpannerClient != nil {
		sc.SpannerClient.Close()
	}
	if sc.StorageClient != nil {
		sc.StorageClient.Close()
	}
}

// 戻り値として返される場合（追跡対象外）
func CreateSpannerClient(ctx context.Context) (*spanner.Client, error) {
	client, err := spanner.NewClient(ctx, "projects/test")
	if err != nil {
		return nil, err
	}
	// 戻り値として返されるため、呼び出し元でCloseする責任がある
	return client, nil
}

// グローバル変数に格納される場合（追跡対象外）
var globalSpannerClient *spanner.Client

func InitializeGlobalClient(ctx context.Context) error {
	client, err := spanner.NewClient(ctx, "projects/test")
	if err != nil {
		return err
	}
	globalSpannerClient = client // グローバル変数に格納
	return nil
}

// チャネルを通して返される場合（追跡対象外）
func CreateClientAsync(ctx context.Context) <-chan *spanner.Client {
	ch := make(chan *spanner.Client, 1)
	go func() {
		client, err := spanner.NewClient(ctx, "projects/test")
		if err != nil {
			close(ch)
			return
		}
		ch <- client // チャネルを通して返される
	}()
	return ch
}

// 関数内で条件分岐がある場合の正しい例
func ConditionalClientCreation(ctx context.Context, createClient bool) error {
	if !createClient {
		return nil
	}

	client, err := spanner.NewClient(ctx, "projects/test")
	if err != nil {
		return err
	}
	defer client.Close() // 条件付きで作成されても、作成されたら必ずクローズ

	return nil
}