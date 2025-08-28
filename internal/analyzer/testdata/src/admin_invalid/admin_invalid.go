package admin_invalid

import (
	"context"

	firebase "firebase.google.com/go/v4"
)

// Firebase Database ClientのClose不足
func missingDatabaseClose(ctx context.Context, app *firebase.App) error {
	client, err := app.Database(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	return nil
}

// Firebase Firestore ClientのClose不足
func missingFirestoreClose(ctx context.Context, app *firebase.App) error {
	client, err := app.Firestore(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer client.Close() が不足

	return nil
}

// 複数リソースで一部のClose不足
func partialResourceClose(ctx context.Context) error {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return err
	}

	dbClient, err := app.Database(ctx) // want "GCP リソース 'dbClient' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err
	}
	// defer dbClient.Close() が不足

	fsClient, err := app.Firestore(ctx)
	if err != nil {
		return err
	}
	defer fsClient.Close() // これは正しい

	return nil
}

// ネストした関数でのClose不足
func nestedFunctionMissingClose(ctx context.Context) error {
	func() {
		app, _ := firebase.NewApp(ctx, nil)
		client, _ := app.Database(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
		// defer client.Close() が不足
		_ = client
	}()

	return nil
}

// エラーハンドリング後のClose不足
func errorHandlingMissingClose(ctx context.Context, app *firebase.App) error {
	client, err := app.Database(ctx) // want "GCP リソース 'client' の解放処理 \\(Close\\) が見つかりません"
	if err != nil {
		return err // エラー時にCloseが呼ばれない
	}
	// defer client.Close() が不足

	return nil
}
