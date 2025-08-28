package spanner

import (
	"context"
)

// Client はSpannerクライアントのモック
type Client struct{}

// ReadWriteTransaction はReadWriteTransactionのモック
type ReadWriteTransaction struct{}

// ReadOnlyTransaction はReadOnlyTransactionのモック
type ReadOnlyTransaction struct{}

// Statement はクエリステートメントのモック
type Statement struct {
	Params map[string]interface{}
}

// Key はSpannerキーのモック
type Key []interface{}

// Row は結果行のモック
type Row struct{}

// RowIterator は結果イテレータのモック
type RowIterator struct{}

// NewClient creates a new Spanner client (mock)
func NewClient(ctx context.Context, project string) (*Client, error) {
	return &Client{}, nil
}

// NewStatement creates a new statement (mock)
func NewStatement(sql string) Statement {
	return Statement{Params: make(map[string]interface{})}
}

// ReadWriteTransaction executes a read-write transaction (mock)
func (c *Client) ReadWriteTransaction(ctx context.Context, fn func(context.Context, *ReadWriteTransaction) error) (interface{}, error) {
	return nil, fn(ctx, &ReadWriteTransaction{})
}

// BatchReadWriteTransaction executes a batch read-write transaction (mock)
func (c *Client) BatchReadWriteTransaction(ctx context.Context, fn func(context.Context, *ReadWriteTransaction) error) (interface{}, error) {
	return nil, fn(ctx, &ReadWriteTransaction{})
}

// ReadOnlyTransaction creates a read-only transaction (mock)
func (c *Client) ReadOnlyTransaction() *ReadOnlyTransaction {
	return &ReadOnlyTransaction{}
}

// Single creates a single-use read-only transaction (mock)
func (c *Client) Single() *ReadOnlyTransaction {
	return &ReadOnlyTransaction{}
}

// Close closes the client (mock)
func (c *Client) Close() error {
	return nil
}

// Update executes an update statement (mock)
func (txn *ReadWriteTransaction) Update(ctx context.Context, stmt Statement) (int64, error) {
	return 0, nil
}

// Query executes a query (mock)
func (txn *ReadOnlyTransaction) Query(ctx context.Context, stmt Statement) *RowIterator {
	return &RowIterator{}
}

// ReadRow reads a single row (mock)
func (txn *ReadOnlyTransaction) ReadRow(ctx context.Context, table string, key Key, columns []string) (*Row, error) {
	return &Row{}, nil
}

// Do executes a query with a callback (mock)
func (txn *ReadOnlyTransaction) Do(fn func(*Row) error) error {
	return fn(&Row{})
}

// Close closes the read-only transaction (mock)
func (txn *ReadOnlyTransaction) Close() {
}

// Next returns the next row (mock)
func (iter *RowIterator) Next() (*Row, error) {
	return nil, context.Canceled // Simulate end of iteration
}

// Stop stops the iterator (mock)
func (iter *RowIterator) Stop() {
}

// Columns extracts column values (mock)
func (row *Row) Columns(dest ...interface{}) error {
	return nil
}
