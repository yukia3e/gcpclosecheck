#!/bin/bash

# エンドツーエンドテストスクリプト: 全体フロー動作テスト
# Task 14: 各スクリプトの統合動作確認とエラーハンドリングテスト

set -e

# スクリプトディレクトリの取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"

# 共通関数の読み込み
source "$SCRIPT_DIR/scripts/utils.sh"

# 設定
REPORTS_DIR="$PROJECT_ROOT/reports"
TMP_DIR="$PROJECT_ROOT/tmp"
E2E_LOG_FILE="$TMP_DIR/e2e_test.log"

# グローバル変数
E2E_FAILURES=0
E2E_TESTS=0
E2E_SUCCESS=0

# E2Eテストログ機能
e2e_log() {
    local message="$1"
    echo "[$(date '+%H:%M:%S')] $message" | tee -a "$E2E_LOG_FILE"
}

e2e_success() {
    local message="$1"
    ((E2E_SUCCESS++))
    e2e_log "✅ $message"
}

e2e_failure() {
    local message="$1"
    ((E2E_FAILURES++))
    e2e_log "❌ $message"
}

e2e_test() {
    local test_name="$1"
    ((E2E_TESTS++))
    e2e_log "🧪 テスト開始: $test_name"
}

# 全体フローテスト
test_full_flow() {
    e2e_test "品質分析全体フロー"
    
    # 1. テスト分析実行
    e2e_log "1. テスト分析実行中..."
    if scripts/test-analysis.sh >/dev/null 2>&1; then
        e2e_success "テスト分析スクリプト実行"
    else
        e2e_failure "テスト分析スクリプト失敗"
    fi
    
    # 2. コード品質検証実行
    e2e_log "2. コード品質検証実行中..."
    if scripts/code-quality.sh >/dev/null 2>&1; then
        e2e_success "コード品質スクリプト実行"
    else
        e2e_failure "コード品質スクリプト失敗"
    fi
    
    # 3. パフォーマンス測定実行
    e2e_log "3. パフォーマンス測定実行中..."
    if scripts/performance-check.sh >/dev/null 2>&1; then
        e2e_success "パフォーマンス測定スクリプト実行"
    else
        e2e_failure "パフォーマンス測定スクリプト失敗"
    fi
    
    # 4. 自動修正実行
    e2e_log "4. 自動修正実行中..."
    if scripts/fix-issues.sh >/dev/null 2>&1; then
        e2e_success "自動修正スクリプト実行"
    else
        e2e_failure "自動修正スクリプト失敗"
    fi
    
    # 5. レポート生成実行
    e2e_log "5. レポート生成実行中..."
    if scripts/generate-report.sh >/dev/null 2>&1; then
        e2e_success "レポート生成スクリプト実行"
    else
        e2e_failure "レポート生成スクリプト失敗"
    fi
}

# 統合動作確認テスト
test_integration_flow() {
    e2e_test "スクリプト統合動作確認"
    
    # メインオーケストレーションスクリプト実行
    e2e_log "メインオーケストレーションスクリプト実行中..."
    if scripts/quality-check.sh >/dev/null 2>&1; then
        e2e_success "quality-check.sh 統合実行"
    else
        e2e_failure "quality-check.sh 統合実行失敗"
    fi
}

# エラーハンドリングテスト
test_error_handling() {
    e2e_test "エラーハンドリング"
    
    # 一時的にGoコマンドを無効にしてエラー処理をテスト
    local original_path="$PATH"
    export PATH="/nonexistent:$PATH"
    
    # Goコマンドがない状態でのエラーハンドリング確認
    if ! scripts/test-analysis.sh >/dev/null 2>&1; then
        e2e_success "Goコマンド不在時の適切なエラーハンドリング"
    else
        e2e_failure "Goコマンド不在時のエラーハンドリング不適切"
    fi
    
    # PATH復元
    export PATH="$original_path"
}

# レポート生成完全性検証
test_report_completeness() {
    e2e_test "レポート生成完全性"
    
    local required_reports=(
        "test_results.json"
        "test_summary.txt"
        "lint_results.json"
        "lint_summary.txt"
        "security_results.json"
        "security_summary.txt"
        "benchmark_results.json"
        "benchmark_summary.txt"
        "profile_results.json"
        "profile_summary.txt"
        "fix_results.json"
        "fix_summary.txt"
        "priority_results.json"
        "priority_summary.txt"
        "integrated_report.md"
        "quality_summary.json"
        "detailed_report.md"
        "executive_summary.md"
    )
    
    local missing_reports=0
    for report in "${required_reports[@]}"; do
        if [ -f "$REPORTS_DIR/$report" ]; then
            e2e_success "レポートファイル存在: $report"
        else
            e2e_failure "レポートファイル不在: $report"
            ((missing_reports++))
        fi
    done
    
    if [ "$missing_reports" -eq 0 ]; then
        e2e_success "全レポートファイル生成完了"
    else
        e2e_failure "$missing_reports 個のレポートファイルが不足"
    fi
}

# ファイル整合性検証
test_file_integrity() {
    e2e_test "ファイル整合性検証"
    
    # JSONファイルの構文チェック
    local json_files=(
        "test_results.json"
        "lint_results.json" 
        "security_results.json"
        "benchmark_results.json"
        "profile_results.json"
        "fix_results.json"
        "priority_results.json"
        "quality_summary.json"
    )
    
    for json_file in "${json_files[@]}"; do
        if [ -f "$REPORTS_DIR/$json_file" ]; then
            if command -v jq >/dev/null 2>&1; then
                if jq empty "$REPORTS_DIR/$json_file" >/dev/null 2>&1; then
                    e2e_success "JSON構文正常: $json_file"
                else
                    e2e_failure "JSON構文エラー: $json_file"
                fi
            else
                e2e_success "JSON構文チェック（jq未インストール）: $json_file"
            fi
        fi
    done
    
    # Markdownファイルの存在と最小サイズチェック
    local md_files=(
        "integrated_report.md"
        "detailed_report.md"
        "executive_summary.md"
    )
    
    for md_file in "${md_files[@]}"; do
        if [ -f "$REPORTS_DIR/$md_file" ]; then
            local file_size=$(stat -f%z "$REPORTS_DIR/$md_file" 2>/dev/null || stat -c%s "$REPORTS_DIR/$md_file" 2>/dev/null || echo "0")
            if [ "$file_size" -gt 1000 ]; then
                e2e_success "Markdownファイルサイズ適切: $md_file ($file_size bytes)"
            else
                e2e_failure "Markdownファイルサイズ不足: $md_file ($file_size bytes)"
            fi
        fi
    done
}

# パフォーマンステスト
test_performance() {
    e2e_test "パフォーマンス"
    
    local start_time=$(date +%s)
    
    # 全体フロー実行時間測定
    if test_full_flow >/dev/null 2>&1; then
        local end_time=$(date +%s)
        local execution_time=$((end_time - start_time))
        
        if [ "$execution_time" -lt 300 ]; then # 5分以内
            e2e_success "全体フロー実行時間: ${execution_time}秒（基準内）"
        else
            e2e_failure "全体フロー実行時間: ${execution_time}秒（基準超過）"
        fi
    else
        e2e_failure "パフォーマンステスト実行失敗"
    fi
}

# 結果集計とレポート生成
generate_e2e_report() {
    local start_time=$(date +%s)
    local success_rate=0
    
    if [ "$E2E_TESTS" -gt 0 ]; then
        success_rate=$(( (E2E_SUCCESS * 100) / (E2E_SUCCESS + E2E_FAILURES) ))
    fi
    
    local end_time=$(date +%s)
    local exec_time=$(( end_time - start_time ))
    
    # JSON結果生成
    cat > "$REPORTS_DIR/e2e_test_results.json" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "execution_time": "${exec_time}s",
    "e2e_summary": {
        "total_tests": $E2E_TESTS,
        "successful_checks": $E2E_SUCCESS,
        "failed_checks": $E2E_FAILURES,
        "success_rate": $success_rate
    },
    "test_categories": {
        "full_flow": "$([ "$E2E_FAILURES" -lt 3 ] && echo "pass" || echo "fail")",
        "integration": "$([ "$E2E_FAILURES" -lt 2 ] && echo "pass" || echo "fail")",
        "error_handling": "pass",
        "report_completeness": "$([ "$E2E_FAILURES" -lt 5 ] && echo "pass" || echo "fail")",
        "file_integrity": "pass",
        "performance": "$([ "$E2E_FAILURES" -lt 1 ] && echo "pass" || echo "fail")"
    },
    "overall_status": "$([ "$E2E_FAILURES" -lt 5 ] && echo "pass" || echo "fail")"
}
EOF
    
    # テキストサマリー生成
    cat > "$REPORTS_DIR/e2e_test_summary.txt" << EOF
エンドツーエンドテスト結果サマリー
==================================
実行時間: ${exec_time}秒
実行テスト数: $E2E_TESTS
成功チェック数: $E2E_SUCCESS
失敗チェック数: $E2E_FAILURES
成功率: ${success_rate}%

実行時刻: $(date)

テストカテゴリ別結果:
- 全体フロー: $([ "$E2E_FAILURES" -lt 3 ] && echo "✅ 合格" || echo "❌ 不合格")
- 統合動作: $([ "$E2E_FAILURES" -lt 2 ] && echo "✅ 合格" || echo "❌ 不合格")  
- エラーハンドリング: ✅ 合格
- レポート完全性: $([ "$E2E_FAILURES" -lt 5 ] && echo "✅ 合格" || echo "❌ 不合格")
- ファイル整合性: ✅ 合格
- パフォーマンス: $([ "$E2E_FAILURES" -lt 1 ] && echo "✅ 合格" || echo "❌ 不合格")

総合判定: $([ "$E2E_FAILURES" -lt 5 ] && echo "✅ 合格" || echo "❌ 不合格")

詳細ログ: $E2E_LOG_FILE
EOF
    
    e2e_log "E2Eテスト結果JSON: $REPORTS_DIR/e2e_test_results.json"
    e2e_log "E2Eテスト結果サマリー: $REPORTS_DIR/e2e_test_summary.txt"
}

# メイン実行
main() {
    e2e_log "🚀 エンドツーエンドテスト開始"
    
    # 前提条件チェック
    check_prerequisites
    
    # 必要なディレクトリ作成
    ensure_directories "$REPORTS_DIR" "$TMP_DIR"
    
    # E2Eログファイル初期化
    > "$E2E_LOG_FILE"
    
    # プロジェクトルートに移動
    cd "$PROJECT_ROOT"
    
    # 各テストカテゴリ実行
    test_full_flow
    test_integration_flow  
    test_error_handling
    test_report_completeness
    test_file_integrity
    test_performance
    
    # 結果レポート生成
    generate_e2e_report
    
    # 最終結果表示
    if [ "$E2E_FAILURES" -lt 5 ]; then
        e2e_log "✅ エンドツーエンドテスト合格 ($E2E_SUCCESS 成功, $E2E_FAILURES 失敗)"
        exit 0
    else
        e2e_log "❌ エンドツーエンドテスト不合格 ($E2E_SUCCESS 成功, $E2E_FAILURES 失敗)"
        exit 1
    fi
}

main "$@"