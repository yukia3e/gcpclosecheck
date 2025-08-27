#!/bin/bash
# go vet統合のためのインストールスクリプト

set -e

echo "gcpclosecheck を go vet ツールとしてインストールしています..."

# バイナリのビルド
go build -o gcpclosecheck ./cmd/gcpclosecheck/

# GOPATH/binにインストール
GOBIN=$(go env GOPATH)/bin
cp gcpclosecheck "$GOBIN/"

echo "インストールが完了しました。"
echo ""
echo "使用方法:"
echo "  go vet -vettool=gcpclosecheck ./..."
echo "  go vet -vettool=\$(which gcpclosecheck) ./internal/service/"
echo ""
echo "これにより依存関係の問題を回避できます。"