package context_valid

import (
	"context"
	"time"
)

// context.WithCancelの正しい使用パターン
func correctWithCancel(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // 正しいパターン

	// 何らかの処理
	time.Sleep(100 * time.Millisecond)

	return nil
}

// context.WithTimeoutの正しい使用パターン
func correctWithTimeout(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel() // 正しいパターン

	// 何らかの処理
	time.Sleep(100 * time.Millisecond)

	return nil
}

// context.WithDeadlineの正しい使用パターン
func correctWithDeadline(ctx context.Context) error {
	deadline := time.Now().Add(10 * time.Second)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel() // 正しいパターン

	// 何らかの処理
	time.Sleep(100 * time.Millisecond)

	return nil
}

// 複合パターン
func correctComplexContext(ctx context.Context) error {
	// 外側のcontext
	outerCtx, outerCancel := context.WithCancel(ctx)
	defer outerCancel()

	// 内側のcontext
	innerCtx, innerCancel := context.WithTimeout(outerCtx, 5*time.Second)
	defer innerCancel()

	// 処理
	time.Sleep(100 * time.Millisecond)
	_ = innerCtx

	return nil
}

// 戻り値として返される場合
func createContextWithCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx) // 戻り値として返されるのでdefer不要
}

// チャネルでのキャンセル通知
func correctChannelPattern(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan bool)
	go func() {
		defer close(done)
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			done <- true
		}
	}()

	<-done
	return nil
}