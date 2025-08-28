# gcpclosecheck

[![Go Report Card](https://goreportcard.com/badge/github.com/yukia3e/gcpclosecheck)](https://goreportcard.com/report/github.com/yukia3e/gcpclosecheck)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

GCP リソースの解放処理 (`Close`, `Stop`, `Cancel`) 漏れを検出する Go 静的解析ツール

## 🔍 概要

`gcpclosecheck` は Google Cloud Platform (GCP) の Go SDK を使用するコードにおいて、適切にリソースが解放されていない箇所を自動検出する静的解析ツールです。

### 検出対象

- **GCPクライアント**: `defer client.Close()` の不足
- **Spanner**: Client, Transaction, RowIterator の解放漏れ
- **Cloud Storage**: Client, Reader, Writer の解放漏れ  
- **Pub/Sub**: Client の解放漏れ
- **Vision API**: Client の解放漏れ
- **Firebase Admin SDK**: Database, Firestore クライアントの解放漏れ
- **reCAPTCHA**: Client の解放漏れ
- **Context**: `context.WithCancel`, `WithTimeout`, `WithDeadline` の `cancel()` 漏れ

## ⚡ 特徴

- **高速**: 軽量なAST解析による高速処理
- **正確**: 偽陽性・偽陰性を最小化するエスケープ解析
- **包括的**: 6つの GCP サービス + Context 対応
- **拡張可能**: YAML 設定でカスタムルール追加
- **go vet 統合**: `-vettool` オプションで既存ワークフローに組込み
- **自動修正**: SuggestedFix による自動 `defer` 文追加

## 🚀 インストール

```bash
go install github.com/yukia3e/gcpclosecheck/cmd/gcpclosecheck@latest
```

## 📖 使用方法

### 基本実行

```bash
# 単一ファイルの解析
gcpclosecheck main.go

# パッケージ全体の解析  
gcpclosecheck ./...

# 特定ディレクトリの解析
gcpclosecheck ./internal/...
```

### go vet との統合

```bash
go vet -vettool=$(which gcpclosecheck) ./...
```

### オプション

```bash
gcpclosecheck [options] [packages]

Options:
  -V, --version          バージョン表示
  -fix                   自動修正を適用  
  -json                  JSON 形式で出力
  -gcpdebug              デバッグモード有効
  -gcpconfig string      設定ファイルパス指定
```

## 💡 使用例

### ❌ 問題のあるコード

```go
package main

import (
    "context"
    "cloud.google.com/go/spanner"
)

func badExample(ctx context.Context) error {
    client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
    if err != nil {
        return err
    }
    // ❌ defer client.Close() が不足

    ctx, cancel := context.WithCancel(ctx)  
    // ❌ defer cancel() が不足
    
    return nil
}
```

### ✅ 修正後のコード

```go
package main

import (
    "context"
    "cloud.google.com/go/spanner"  
)

func goodExample(ctx context.Context) error {
    client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
    if err != nil {
        return err
    }
    defer client.Close() // ✅ 正しい

    ctx, cancel := context.WithCancel(ctx)
    defer cancel() // ✅ 正しい
    
    return nil  
}
```

### 🔧 実行結果

```bash
$ gcpclosecheck ./examples/bad.go
./examples/bad.go:12:2: GCP リソース 'client' の解放処理 (Close) が見つかりません
./examples/bad.go:15:17: Context cancel function should be called with defer
```

## ⚙️ 設定

### カスタム設定ファイル

```yaml
# .gcpclosecheck.yaml
services:
  myservice:
    packages:
      - "github.com/myorg/myservice"
    resource_types:
      MyClient:
        creation_functions: ["NewMyClient"]
        cleanup_method: "Close"
        cleanup_required: true
```

## 🏗️ 開発・ビルド

### 前提条件

- Go 1.21+
- Git

### ビルド

```bash
git clone https://github.com/yukia3e/gcpclosecheck.git
cd gcpclosecheck
make build
```

### テスト実行

```bash
# 全テスト実行
make test

# E2E テスト
make test-e2e  

# ベンチマーク
make bench

# カバレッジ
make test-coverage
```

### 品質チェック

```bash
# 静的解析 + テスト + カバレッジ
make quality-gate

# CI パイプライン
make ci
```

## 🎯 設計哲学

- **Test-Driven Development**: RED → GREEN → REFACTOR
- **高精度**: エスケープ解析による偽陽性最小化
- **高性能**: AST キャッシュとルールキャッシュの効率化
- **拡張性**: プラガブルなルールエンジン
- **統合性**: 既存ツールチェーンとの親和性

## 🏛️ アーキテクチャ

```
├── cmd/gcpclosecheck/          # CLI エントリポイント
├── internal/
│   ├── analyzer/               # 解析エンジン
│   │   ├── analyzer.go         # メイン解析器
│   │   ├── resource_tracker.go # リソース追跡
│   │   ├── defer_analyzer.go   # defer 文解析
│   │   ├── context_analyzer.go # context 解析
│   │   └── escape_analyzer.go  # エスケープ解析
│   └── config/                 # 設定管理
├── testdata/                   # E2E テストデータ
└── rules/                      # デフォルトルール
```

## 🤝 コントリビューション

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

### 開発ガイドライン

- TDD でテスト駆動開発
- golangci-lint による品質チェック
- 80%+ テストカバレッジ維持
- パフォーマンス回帰防止

## 📄 ライセンス

MIT License - 詳細は [LICENSE](LICENSE) ファイルを参照してください。

## 🙋 サポート

- **Issues**: [GitHub Issues](https://github.com/yukia3e/gcpclosecheck/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yukia3e/gcpclosecheck/discussions)
