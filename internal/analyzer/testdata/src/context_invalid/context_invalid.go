package context_invalid

import (
	"context"
	"time"
)

// context.WithCancelのcancel不足
func missingCancel(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx) // want "context.WithCancel のキャンセル関数 'cancel' の呼び出しが見つかりません"
	// defer cancel() が不足

	time.Sleep(100 * time.Millisecond)
	_ = ctx

	return nil
}

// context.WithTimeoutのcancel不足
func missingTimeoutCancel(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second) // want "context.WithTimeout のキャンセル関数 'cancel' の呼び出しが見つかりません"
	// defer cancel() が不足

	time.Sleep(100 * time.Millisecond)
	_ = ctx

	return nil
}

// context.WithDeadlineのcancel不足
func missingDeadlineCancel(ctx context.Context) error {
	deadline := time.Now().Add(10 * time.Second)
	ctx, cancel := context.WithDeadline(ctx, deadline) // want "context.WithDeadline のキャンセル関数 'cancel' の呼び出しが見つかりません"
	// defer cancel() が不足

	time.Sleep(100 * time.Millisecond)
	_ = ctx

	return nil
}

// 複数のcontextで一部のcancel不足
func partialContextCancel(ctx context.Context) error {
	// 外側のcontext（cancel不足）
	outerCtx, outerCancel := context.WithCancel(ctx) // want "context.WithCancel のキャンセル関数 'outerCancel' の呼び出しが見つかりません"
	// defer outerCancel() が不足

	// 内側のcontext（正しい）
	innerCtx, innerCancel := context.WithTimeout(outerCtx, 5*time.Second)
	defer innerCancel()

	time.Sleep(100 * time.Millisecond)
	_ = innerCtx

	return nil
}

// ネストした関数でのcancel不足
func nestedFunctionMissingCancel(ctx context.Context) error {
	func() {
		_, cancel := context.WithCancel(ctx) // want "context.WithCancel のキャンセル関数 'cancel' の呼び出しが見つかりません"
		// defer cancel() が不足
		time.Sleep(100 * time.Millisecond)
	}()

	return nil
}

// エラーハンドリング後のcancel不足
func errorHandlingMissingCancel(ctx context.Context) error {
	_, cancel := context.WithCancel(ctx) // want "context.WithCancel のキャンセル関数 'cancel' の呼び出しが見つかりません"
	
	// エラーが発生した場合のcancel不足
	if time.Now().Unix()%2 == 0 {
		return nil // エラー時にcancelが呼ばれない
	}
	
	// defer cancel() が不足
	return nil
}

// 変数名が異なるパターン
func differentVariableName(ctx context.Context) error {
	newCtx, cancelFunc := context.WithCancel(ctx) // want "context.WithCancel のキャンセル関数 'cancelFunc' の呼び出しが見つかりません"
	// defer cancelFunc() が不足

	time.Sleep(100 * time.Millisecond)
	_ = newCtx

	return nil
}