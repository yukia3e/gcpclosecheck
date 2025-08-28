# 品質チェックツール使用ガイド

gcpclosecheckプロジェクトの品質とパフォーマンスを包括的に分析・改善するためのツール群の使用方法とベストプラクティスを説明します。

## 📋 概要

このツール群は以下の機能を提供します：

- **テスト分析**: カバレッジ分析、テスト実行結果の詳細評価
- **コード品質検証**: 静的解析、セキュリティスキャン、ビルド検証
- **パフォーマンス測定**: ベンチマーク実行、プロファイリング
- **自動修正**: 検出された問題の自動修正とコードフォーマット
- **レポート生成**: 統合レポートと詳細分析結果
- **継続改善支援**: 品質指標履歴管理とトレンド分析

## 🚀 クイックスタート

### 前提条件

- Go 1.25.0以上
- golangci-lint 2.4.0以上（推奨）
- gosec、govulncheck（オプション）

### 基本的な使用方法

```bash
# 全分析の実行
scripts/quality-check.sh

# テスト分析のみ
scripts/quality-check.sh --test

# 品質検証のみ
scripts/quality-check.sh --quality

# パフォーマンス測定のみ
scripts/quality-check.sh --perf
```

## 📚 各スクリプトの詳細な使用方法

### quality-check.sh - メインオーケストレーションスクリプト

プロジェクトの包括的な品質とパフォーマンス分析を実行します。

**使用方法:**
```bash
scripts/quality-check.sh [オプション]
```

**オプション:**
- `--test`: テスト分析のみ実行（カバレッジ、テスト結果）
- `--quality`: 品質検証のみ実行（静的解析、セキュリティ）  
- `--perf`: パフォーマンス測定のみ実行（ベンチマーク、プロファイル）
- `--all`: 全ての分析を実行（デフォルト）
- `--help`: ヘルプを表示

**出力ファイル:**
- `reports/performance_summary.json`: 実行時間とパフォーマンス情報

### test-analysis.sh - テスト分析専用スクリプト

テストカバレッジの詳細分析とテスト実行結果の評価を行います。

**主な機能:**
- go test -cover を使用したカバレッジ測定
- テスト実行結果の詳細分析
- HTMLカバレッジレポート生成
- 失敗テストの詳細レポート作成

**出力ファイル:**
- `reports/test_results.json`: 構造化されたテスト結果
- `reports/test_summary.txt`: テスト結果サマリー
- `reports/coverage.html`: HTMLカバレッジレポート
- `reports/failed_tests_detail.txt`: 失敗テストの詳細

### code-quality.sh - コード品質検証スクリプト

静的解析、セキュリティスキャン、ビルド検証を実行します。

**主な機能:**
- golangci-lint による静的解析
- gosec によるセキュリティスキャン
- govulncheck による脆弱性チェック
- go build と go vet によるビルド検証

**出力ファイル:**
- `reports/lint_results.json`: 静的解析結果
- `reports/security_results.json`: セキュリティスキャン結果
- `reports/build_results.json`: ビルド検証結果

### performance-check.sh - パフォーマンス測定スクリプト

ベンチマーク実行とプロファイリング分析を行います。

**主な機能:**
- go test -bench によるベンチマーク実行
- CPU・メモリプロファイリング
- パフォーマンス指標の算出

**出力ファイル:**
- `reports/benchmark_results.json`: ベンチマーク結果
- `reports/profile_results.json`: プロファイリング結果

### fix-issues.sh - 自動修正スクリプト

検出された問題の自動修正と優先度付けを行います。

**主な機能:**
- go fmt による自動フォーマット
- goimports による import 整理
- golangci-lint --fix による自動修正
- 問題の優先度付けと修正推奨順序の提案

**出力ファイル:**
- `reports/fix_results.json`: 修正実行結果
- `reports/priority_results.json`: 問題優先度分析

### generate-report.sh - レポート生成スクリプト

統合レポートと詳細分析レポートを生成します。

**主な機能:**
- 全分析結果の統合レポート生成
- 詳細な技術分析レポート作成
- 経営層向けサマリーレポート生成

**出力ファイル:**
- `reports/integrated_report.md`: 統合レポート
- `reports/detailed_report.md`: 詳細技術レポート
- `reports/executive_summary.md`: 経営層向けサマリー

### track-progress.sh - 継続改善支援スクリプト

品質指標の履歴管理とトレンド分析を行います。

**使用方法:**
```bash
scripts/track-progress.sh [オプション]
```

**オプション:**
- `--track`: 品質指標追跡を実行
- `--trend`: トレンド分析のみ実行
- `--compare`: 比較分析のみ実行

**出力ファイル:**
- `reports/history/`: 品質指標履歴データ
- `reports/trend_analysis.md`: トレンド分析レポート
- `reports/progress_report.md`: 進捗レポート

## 🎯 ベストプラクティス

### 1. 定期実行の推奨

```bash
# 週次での品質チェック実行を推奨
# crontabに以下を追加:
# 0 2 * * 1 /path/to/gcpclosecheck/scripts/quality-check.sh
```

### 2. 品質改善ワークフロー

1. **分析実行**: `scripts/quality-check.sh` で現状把握
2. **問題特定**: `reports/` ディレクトリのレポートを確認
3. **自動修正**: `scripts/fix-issues.sh` で修正可能な問題を解決
4. **手動修正**: 残りの問題を優先度順に対応
5. **効果測定**: `scripts/track-progress.sh` で改善効果を確認

### 3. CI/CDパイプラインとの統合

```yaml
# GitHub Actions例
- name: Quality Check
  run: |
    chmod +x scripts/quality-check.sh
    scripts/quality-check.sh --test
```

### 4. レポートの活用方法

- **developers**: `reports/detailed_report.md` で技術的な詳細を確認
- **managers**: `reports/executive_summary.md` で全体状況を把握
- **継続改善**: `reports/trend_analysis.md` で改善傾向を追跡

## 🔧 トラブルシューティング

### よくある問題と解決方法

#### 1. golangci-lint設定エラー

**エラー例:**
```
Error: can't load config: unsupported version of the configuration
```

**解決方法:**
```bash
# .golangci.yml の更新
golangci-lint --version  # バージョン確認
# 最新の設定形式に更新
```

#### 2. テスト失敗の大量発生

**対処方法:**
1. `reports/failed_tests_detail.txt` で失敗原因を確認
2. 依存関係の問題か確認: `go mod tidy`
3. 段階的にテスト修正を実行

#### 3. メモリ不足エラー

**対処方法:**
```bash
# 大規模プロジェクトの場合、並列数を削減
# scripts/quality-check.sh内のMAX_PARALLEL_JOBSを調整
```

#### 4. カバレッジが取得できない

**確認事項:**
- テストファイルが存在するか
- go.mod の設定が正しいか
- テストが実際に実行されているか

### 既知の問題

#### 1. GCPライブラリの依存関係エラー

一部のテストでGCPクライアントライブラリのインポートエラーが発生する場合があります。

**回避策:**
```bash
# 必要なGCPライブラリをインストール
go mod download cloud.google.com/go/...
```

#### 2. macOSでのdate コマンド互換性

macOSとLinuxでdateコマンドの動作が異なる場合があります。

**対処済み:**
スクリプト内で自動的に環境を検出し、適切なコマンドを使用します。

## 📊 運用ガイド

### 継続的品質改善のための運用プロセス

#### 1. 日次モニタリング

```bash
# 開発チーム向け: 毎日の品質チェック
scripts/quality-check.sh --test
```

#### 2. 週次品質レビュー

```bash
# 全分析と履歴比較
scripts/quality-check.sh
scripts/track-progress.sh
```

#### 3. 月次品質評価

1. 詳細レポートの作成と共有
2. 品質指標のトレンド分析
3. 改善計画の見直し

### 品質ゲート基準

以下の基準を満たすことを推奨します：

- **テストカバレッジ**: 80%以上
- **テスト失敗**: 0件
- **セキュリティ問題**: 0件
- **クリティカルなLint問題**: 0件

### メンテナンス

#### 定期メンテナンス作業

```bash
# 週次クリーンアップ
scripts/cleanup.sh

# 月次セットアップ確認
scripts/setup.sh --verify
```

#### ログとレポートの管理

- レポートは自動的にローテーションされます
- 30日以上古いレポートは自動削除されます
- 履歴データは`reports/history/`に永続保存されます

## 🔄 アップデートガイド

### ツールのアップデート

```bash
# Goツールチェーンの更新
mise install go@latest

# golangci-lintの更新
golangci-lint --version
# 最新版をインストール
```

### 設定の更新

- `.golangci.yml`: 静的解析ルールの調整
- `scripts/utils.sh`: 共通設定の変更
- `scripts/quality-check.sh`: 実行パラメーターの調整

## 📈 品質指標の理解

### 総合品質スコア算出方法

```
総合品質スコア = テストカバレッジ(%) - (失敗テスト数 × 2) - (Lint問題数 ÷ 2) - (セキュリティ問題数 × 10)
```

### 推奨改善順序

1. **セキュリティ問題** (最優先): 即座に修正
2. **テスト失敗**: リリース前に修正
3. **カバレッジ不足**: 継続的改善
4. **Lint問題**: 次回リリースで修正

## 🤝 サポート

### 問題報告

品質チェックツールに関する問題や改善提案は、プロジェクトのissueとして報告してください。

### 拡張方法

新しい品質チェック機能の追加方法：

1. `scripts/`ディレクトリに新しいスクリプトを作成
2. `scripts/quality-check.sh`に統合
3. `scripts/tests/test-scripts.sh`にテストを追加
4. このドキュメントを更新
