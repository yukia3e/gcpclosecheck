package admin_valid

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
)

// Firebase Appの正しい使用パターン
func correctFirebaseApp(ctx context.Context) error {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return err
	}
	// Firebase Appは特別なClose処理が不要
	_ = app

	return nil
}

// Firebase Database Clientの正しい使用パターン
func correctDatabaseClient(ctx context.Context, app *firebase.App) error {
	client, err := app.Database(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // 正しいパターン

	return nil
}

// Firebase Auth Clientの正しい使用パターン
func correctAuthClient(ctx context.Context, app *firebase.App) error {
	client, err := app.Auth(ctx)
	if err != nil {
		return err
	}
	// Auth clientは特別なClose処理が不要
	_ = client

	return nil
}

// Firebase Firestoreの正しい使用パターン
func correctFirestoreClient(ctx context.Context, app *firebase.App) error {
	client, err := app.Firestore(ctx)
	if err != nil {
		return err
	}
	defer client.Close() // 正しいパターン

	return nil
}

// 複合パターン
func correctComplexFirebase(ctx context.Context) error {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return err
	}

	// Database
	dbClient, err := app.Database(ctx)
	if err != nil {
		return err
	}
	defer dbClient.Close()

	// Firestore
	fsClient, err := app.Firestore(ctx)
	if err != nil {
		return err
	}
	defer fsClient.Close()

	return nil
}

// Real-time Database参照の正しい使用パターン
func correctDatabaseRef(dbClient *db.Client) *db.Ref {
	ref := dbClient.NewRef("test")
	return ref // 戻り値として返されるのでdefer不要
}