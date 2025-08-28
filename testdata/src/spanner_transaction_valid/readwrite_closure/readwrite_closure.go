package readwrite_closure

import (
	"context"
	"log"

	"cloud.google.com/go/spanner"
)

// ReadWriteTransactionクロージャパターン - 自動管理されるため検出されるべきではない
func processWithReadWriteTransaction(ctx context.Context, client *spanner.Client) {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// txnはSpannerフレームワークが自動管理 - Close不要
		stmt := spanner.NewStatement("UPDATE table SET column = @value WHERE id = @id")
		stmt.Params["value"] = "new_value"
		stmt.Params["id"] = 123

		_, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}

		// 複数のUpdate操作
		stmt2 := spanner.NewStatement("INSERT INTO another_table (name, value) VALUES (@name, @value)")
		stmt2.Params["name"] = "test"
		stmt2.Params["value"] = 456

		_, err = txn.Update(ctx, stmt2)
		return err
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
	}
}

// BatchReadWriteTransaction - 同様に自動管理
func processBatchWithReadWriteTransaction(ctx context.Context, client *spanner.Client) {
	_, err := client.BatchReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// txnは自動管理される
		stmt := spanner.NewStatement("DELETE FROM table WHERE status = @status")
		stmt.Params["status"] = "inactive"

		_, err := txn.Update(ctx, stmt)
		return err
	})

	if err != nil {
		log.Printf("Batch transaction failed: %v", err)
	}
}

// 変数に格納されたクロージャでのReadWriteTransaction
func processWithVariableTransaction(ctx context.Context, client *spanner.Client) {
	transactionFunc := func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// txnは自動管理される
		stmt := spanner.NewStatement("UPDATE settings SET value = @value WHERE key = @key")
		stmt.Params["value"] = "enabled"
		stmt.Params["key"] = "feature_flag"

		_, err := txn.Update(ctx, stmt)
		return err
	}

	_, err := client.ReadWriteTransaction(ctx, transactionFunc)
	if err != nil {
		log.Printf("Variable transaction failed: %v", err)
	}
}

// ネストしたReadWriteTransactionクロージャ
func processNestedReadWriteTransactions(ctx context.Context, client *spanner.Client) {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, outerTxn *spanner.ReadWriteTransaction) error {
		// outerTxnは自動管理される
		stmt1 := spanner.NewStatement("UPDATE parent_table SET updated_at = CURRENT_TIMESTAMP() WHERE id = @id")
		stmt1.Params["id"] = 1

		_, err := outerTxn.Update(ctx, stmt1)
		if err != nil {
			return err
		}

		// 内部でさらにReadWriteTransactionを使用（実際のコードでは推奨されないが、テストケースとして）
		_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, innerTxn *spanner.ReadWriteTransaction) error {
			// innerTxnも自動管理される
			stmt2 := spanner.NewStatement("INSERT INTO child_table (parent_id, data) VALUES (@parent_id, @data)")
			stmt2.Params["parent_id"] = 1
			stmt2.Params["data"] = "child_data"

			_, err := innerTxn.Update(ctx, stmt2)
			return err
		})

		return err
	})

	if err != nil {
		log.Printf("Nested transaction failed: %v", err)
	}
}
