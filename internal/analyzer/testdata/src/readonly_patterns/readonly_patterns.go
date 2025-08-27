package readonly_patterns

import (
	"context"
	"log"

	"cloud.google.com/go/spanner"
)

// ReadOnlyTransactionの適切なClose処理 - 検出されるべき
func queryWithReadOnlyTransactionProper(ctx context.Context, client *spanner.Client) {
	txn := client.ReadOnlyTransaction()
	defer txn.Close() // 適切なClose処理
	
	stmt := spanner.NewStatement("SELECT id, name FROM table WHERE active = true")
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		
		var id int64
		var name string
		if err := row.Columns(&id, &name); err != nil {
			log.Printf("Failed to parse row: %v", err)
			continue
		}
		
		log.Printf("ID: %d, Name: %s", id, name)
	}
}

// ReadOnlyTransactionのクロージャパターン - 自動管理されるため検出されるべきではない
func queryWithReadOnlyTransactionClosure(ctx context.Context, client *spanner.Client) {
	err := client.ReadOnlyTransaction().Query(ctx, spanner.NewStatement("SELECT COUNT(*) FROM table")).Do(func(row *spanner.Row) error {
		// このパターンではReadOnlyTransactionは自動管理される
		var count int64
		if err := row.Columns(&count); err != nil {
			return err
		}
		log.Printf("Count: %d", count)
		return nil
	})
	
	if err != nil {
		log.Printf("Query failed: %v", err)
	}
}

// Single.ReadOnlyTransaction - Spannerクライアント経由での自動管理パターン
func queryWithSingleReadOnlyTransaction(ctx context.Context, client *spanner.Client) {
	stmt := spanner.NewStatement("SELECT MAX(created_at) FROM events")
	
	row, err := client.Single().ReadRow(ctx, "table", spanner.Key{"key1"}, []string{"column1", "column2"})
	if err != nil {
		log.Printf("ReadRow failed: %v", err)
		return
	}
	
	var col1, col2 string
	if err := row.Columns(&col1, &col2); err != nil {
		log.Printf("Failed to parse columns: %v", err)
		return
	}
	
	log.Printf("Column1: %s, Column2: %s", col1, col2)
}

// ReadOnlyTransactionのスナップショット読み取り
func queryWithReadOnlyTransactionSnapshot(ctx context.Context, client *spanner.Client) {
	txn := client.ReadOnlyTransaction()
	defer txn.Close() // 適切なClose処理
	
	// 複数のクエリを同一スナップショットで実行
	stmt1 := spanner.NewStatement("SELECT COUNT(*) FROM orders WHERE status = @status")
	stmt1.Params["status"] = "pending"
	
	iter1 := txn.Query(ctx, stmt1)
	defer iter1.Stop()
	
	row1, err := iter1.Next()
	if err != nil {
		log.Printf("First query failed: %v", err)
		return
	}
	
	var pendingCount int64
	if err := row1.Columns(&pendingCount); err != nil {
		log.Printf("Failed to parse pending count: %v", err)
		return
	}
	
	stmt2 := spanner.NewStatement("SELECT COUNT(*) FROM orders WHERE status = @status")
	stmt2.Params["status"] = "completed"
	
	iter2 := txn.Query(ctx, stmt2)
	defer iter2.Stop()
	
	row2, err := iter2.Next()
	if err != nil {
		log.Printf("Second query failed: %v", err)
		return
	}
	
	var completedCount int64
	if err := row2.Columns(&completedCount); err != nil {
		log.Printf("Failed to parse completed count: %v", err)
		return
	}
	
	log.Printf("Pending: %d, Completed: %d", pendingCount, completedCount)
}

// ReadOnlyTransactionでの大量データ処理
func processBatchWithReadOnlyTransaction(ctx context.Context, client *spanner.Client) {
	txn := client.ReadOnlyTransaction()
	defer txn.Close() // 適切なClose処理
	
	stmt := spanner.NewStatement("SELECT id, data FROM large_table ORDER BY id")
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	
	batchSize := 0
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		
		var id int64
		var data string
		if err := row.Columns(&id, &data); err != nil {
			log.Printf("Failed to parse row: %v", err)
			continue
		}
		
		// バッチ処理ロジック
		processSingleRecord(id, data)
		batchSize++
		
		if batchSize >= 1000 {
			log.Printf("Processed batch of %d records", batchSize)
			batchSize = 0
		}
	}
	
	if batchSize > 0 {
		log.Printf("Processed final batch of %d records", batchSize)
	}
}

func processSingleRecord(id int64, data string) {
	// レコード処理ロジック
	log.Printf("Processing record ID: %d", id)
}