package nested_patterns

import (
	"context"
	"log"
	"sync"

	"cloud.google.com/go/spanner"
)

// 複雑なネストしたトランザクション処理パターン
type TransactionProcessor struct {
	client *spanner.Client
	mu     sync.Mutex
}

// ReadWriteTransactionとReadOnlyTransactionの混在パターン
func (tp *TransactionProcessor) processComplexTransaction(ctx context.Context, orderID int64) error {
	// ReadWriteTransactionで注文を更新 - 自動管理
	_, err := tp.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 注文ステータスを更新
		updateStmt := spanner.NewStatement("UPDATE orders SET status = @status, updated_at = CURRENT_TIMESTAMP() WHERE id = @id")
		updateStmt.Params["status"] = "processing"
		updateStmt.Params["id"] = orderID

		_, err := txn.Update(ctx, updateStmt)
		if err != nil {
			return err
		}

		// 在庫を減らす
		inventoryStmt := spanner.NewStatement("UPDATE inventory SET quantity = quantity - @amount WHERE product_id = (SELECT product_id FROM orders WHERE id = @order_id)")
		inventoryStmt.Params["amount"] = 1
		inventoryStmt.Params["order_id"] = orderID

		_, err = txn.Update(ctx, inventoryStmt)
		if err != nil {
			return err
		}

		// 内部でReadOnlyTransactionを使用してログを取得
		logTxn := tp.client.ReadOnlyTransaction()
		defer logTxn.Close() // ReadOnlyTransactionは明示的にClose必要

		logStmt := spanner.NewStatement("SELECT action, timestamp FROM order_logs WHERE order_id = @order_id ORDER BY timestamp DESC LIMIT 10")
		logStmt.Params["order_id"] = orderID

		iter := logTxn.Query(ctx, logStmt)
		defer iter.Stop()

		for {
			row, err := iter.Next()
			if err != nil {
				break
			}

			var action string
			var timestamp string
			if err := row.Columns(&action, &timestamp); err != nil {
				log.Printf("Failed to parse log: %v", err)
				continue
			}

			log.Printf("Order %d log: %s at %s", orderID, action, timestamp)
		}

		return nil
	})

	return err
}

// 並列処理でのトランザクション管理
func (tp *TransactionProcessor) processBatchOrders(ctx context.Context, orderIDs []int64) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(orderIDs))

	for _, orderID := range orderIDs {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()

			// 各ゴルーチンでReadWriteTransaction - 自動管理
			_, err := tp.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				stmt := spanner.NewStatement("UPDATE orders SET processed = true WHERE id = @id")
				stmt.Params["id"] = id

				_, err := txn.Update(ctx, stmt)
				return err
			})

			if err != nil {
				errChan <- err
			}
		}(orderID)
	}

	wg.Wait()
	close(errChan)

	// エラー集約
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// 条件分岐でのトランザクション使い分け
func (tp *TransactionProcessor) processConditionalTransaction(ctx context.Context, userID int64, amount float64) error {
	// まずReadOnlyTransactionでユーザー情報を確認
	readTxn := tp.client.ReadOnlyTransaction()
	defer readTxn.Close() // ReadOnlyTransactionは明示的にClose必要

	userStmt := spanner.NewStatement("SELECT balance, account_type FROM accounts WHERE user_id = @user_id")
	userStmt.Params["user_id"] = userID

	row, err := readTxn.ReadRow(ctx, "accounts", spanner.Key{userID}, []string{"balance", "account_type"})
	if err != nil {
		return err
	}

	var balance float64
	var accountType string
	if err := row.Columns(&balance, &accountType); err != nil {
		return err
	}

	// 条件に応じてReadWriteTransactionを実行
	if accountType == "premium" && balance >= amount {
		// プレミアムアカウントの場合 - ReadWriteTransaction使用、自動管理
		_, err = tp.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// 残高を更新
			updateStmt := spanner.NewStatement("UPDATE accounts SET balance = balance - @amount WHERE user_id = @user_id")
			updateStmt.Params["amount"] = amount
			updateStmt.Params["user_id"] = userID

			_, err := txn.Update(ctx, updateStmt)
			if err != nil {
				return err
			}

			// 取引履歴を追加
			historyStmt := spanner.NewStatement("INSERT INTO transaction_history (user_id, amount, type, timestamp) VALUES (@user_id, @amount, @type, CURRENT_TIMESTAMP())")
			historyStmt.Params["user_id"] = userID
			historyStmt.Params["amount"] = amount
			historyStmt.Params["type"] = "withdrawal"

			_, err = txn.Update(ctx, historyStmt)
			return err
		})

		return err
	}

	// 通常アカウントの場合は別の処理
	log.Printf("Transaction denied for user %d: insufficient privileges or balance", userID)
	return nil
}

// ヘルパー関数でのトランザクション抽象化
func (tp *TransactionProcessor) executeInTransaction(ctx context.Context, operations []string) error {
	// ReadWriteTransactionをヘルパー関数で抽象化 - 自動管理
	_, err := tp.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		for i, operation := range operations {
			stmt := spanner.NewStatement(operation)
			stmt.Params["batch_id"] = i

			_, err := txn.Update(ctx, stmt)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

// リトライ機能付きトランザクション
func (tp *TransactionProcessor) processWithRetry(ctx context.Context, data string, maxRetries int) error {
	var lastErr error

	for retry := 0; retry < maxRetries; retry++ {
		// ReadWriteTransactionは内部でSpannerのリトライロジックを処理 - 自動管理
		_, err := tp.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			stmt := spanner.NewStatement("INSERT INTO retry_data (data, attempt, timestamp) VALUES (@data, @attempt, CURRENT_TIMESTAMP())")
			stmt.Params["data"] = data
			stmt.Params["attempt"] = retry + 1

			_, err := txn.Update(ctx, stmt)
			return err
		})

		if err == nil {
			log.Printf("Transaction succeeded on attempt %d", retry+1)
			return nil
		}

		lastErr = err
		log.Printf("Transaction failed on attempt %d: %v", retry+1, err)
	}

	return lastErr
}
