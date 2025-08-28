#!/bin/bash

# テスト分析スクリプト: カバレッジ分析とテスト結果解析
# go testの実行結果とカバレッジデータの包括的な分析を提供

set -e

# スクリプトディレクトリの取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 共通関数の読み込み
source "$SCRIPT_DIR/utils.sh"

# 設定
REPORTS_DIR="$PROJECT_ROOT/reports"
TMP_DIR="$PROJECT_ROOT/tmp"
COVERAGE_THRESHOLD=80

# カバレッジ分析実行
run_coverage_analysis() {
    log_info "カバレッジ分析を開始..."
    
    # カバレッジファイルの生成
    local coverage_file="$TMP_DIR/coverage.out"
    if time_command "カバレッジ測定" go test -coverprofile="$coverage_file" ./...; then
        log_success "カバレッジデータ取得完了"
    else
        log_warning "一部テストが失敗しましたがカバレッジ分析を継続します"
        # カバレッジファイルが存在しない場合は空ファイルを作成
        if [ ! -f "$coverage_file" ]; then
            touch "$coverage_file"
            log_warning "カバレッジファイルが生成されませんでした"
            return 1
        fi
    fi
    
    # カバレッジファイルが空の場合は処理をスキップ
    if [ ! -s "$coverage_file" ]; then
        log_warning "カバレッジデータが空です。テスト分析をスキップします。"
        return 1
    fi
    
    # HTMLカバレッジレポート生成
    local html_report="$REPORTS_DIR/coverage.html"
    if run_with_error_handling "HTMLカバレッジレポート生成" go tool cover -html="$coverage_file" -o "$html_report"; then
        log_success "HTMLレポート: $html_report"
    fi
    
    # カバレッジサマリー生成
    local summary_file="$REPORTS_DIR/coverage_summary.txt"
    if run_with_error_handling "カバレッジサマリー生成" go tool cover -func="$coverage_file"; then
        go tool cover -func="$coverage_file" > "$summary_file"
        log_success "カバレッジサマリー: $summary_file"
        
        # カバレッジ統計の解析
        analyze_coverage_stats "$summary_file"
    fi
}

# カバレッジ統計解析
analyze_coverage_stats() {
    local summary_file="$1"
    
    log_info "カバレッジ統計を解析中..."
    
    # 全体カバレッジ率の取得
    local total_coverage
    if [ -f "$summary_file" ]; then
        total_coverage=$(tail -n 1 "$summary_file" | awk '{print $3}' | sed 's/%//')
        log_info "全体カバレッジ率: ${total_coverage}%"
        
        # カバレッジ閾値チェック
        if [ "$(echo "$total_coverage >= $COVERAGE_THRESHOLD" | bc -l 2>/dev/null || echo "0")" -eq 1 ]; then
            log_success "カバレッジ率が閾値(${COVERAGE_THRESHOLD}%)を満たしています"
        else
            log_warning "カバレッジ率が閾値(${COVERAGE_THRESHOLD}%)を下回っています: ${total_coverage}%"
        fi
        
        # 低カバレッジ関数の特定
        identify_low_coverage_functions "$summary_file"
    else
        log_error "カバレッジサマリーファイルが見つかりません"
        return 1
    fi
}

# 低カバレッジ関数の特定
identify_low_coverage_functions() {
    local summary_file="$1"
    local low_coverage_file="$REPORTS_DIR/low_coverage_functions.txt"
    
    log_info "低カバレッジ関数を特定中..."
    
    # カバレッジ50%未満の関数を抽出
    if head -n -1 "$summary_file" | awk -v threshold=50 '
    BEGIN { print "低カバレッジ関数一覧 (50%未満):" }
    {
        coverage = $3
        gsub(/%/, "", coverage)
        if (coverage < threshold && coverage > 0) {
            printf "- %s: %s%%\n", $2, $3
            count++
        }
    }
    END { 
        if (count == 0) print "すべての関数が50%以上のカバレッジを達成しています"
        else printf "\n低カバレッジ関数数: %d\n", count
    }' > "$low_coverage_file"; then
        log_success "低カバレッジ関数リスト: $low_coverage_file"
    fi
}

# Task 4: テスト実行結果分析機能
analyze_test_execution_results() {
    log_info "テスト実行結果分析を開始..."
    
    local test_output_file="$TMP_DIR/test_verbose_output.txt"
    local test_results_json="$REPORTS_DIR/test_results.json"
    local test_summary_file="$REPORTS_DIR/test_summary.txt"
    
    # 詳細テスト実行（時間測定付き）
    log_info "詳細テストを実行中..."
    local start_time=$(date +%s)
    
    # go test -v でテスト実行結果を詳細取得
    if go test -v ./... > "$test_output_file" 2>&1; then
        local exit_code=0
        log_success "テスト実行完了"
    else
        local exit_code=$?
        log_warning "テスト実行で一部失敗がありました（分析継続）"
    fi
    
    local end_time=$(date +%s)
    local execution_time=$((end_time - start_time))
    
    # テスト結果解析
    parse_test_results "$test_output_file" "$test_results_json" "$test_summary_file" "$execution_time"
}

# テスト結果パース機能
parse_test_results() {
    local test_output="$1"
    local json_output="$2"
    local summary_output="$3"
    local exec_time="$4"
    
    log_info "テスト結果をパース中..."
    
    # テスト結果統計の計算
    local passed_count=0
    local failed_count=0
    local skipped_count=0
    
    if [ -f "$test_output" ]; then
        passed_count=$(grep -c "PASS:" "$test_output" 2>/dev/null || echo 0)
        failed_count=$(grep -c "FAIL:" "$test_output" 2>/dev/null || echo 0)
        skipped_count=$(grep -c "SKIP:" "$test_output" 2>/dev/null || echo 0)
    fi
    
    # JSON形式でテスト結果を出力 - 簡略版で問題解決
    cat > "$json_output" << 'EOF'
{
    "timestamp": "timestamp_placeholder",
    "execution_time": "exec_time_placeholder",
    "test_summary": {
        "passed": passed_placeholder,
        "failed": failed_placeholder,  
        "skipped": skipped_placeholder,
        "total": total_placeholder
    },
    "individual_tests": []
}
EOF
    
    # プレースホルダーの置換
    sed -i '' \
        -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
        -e "s/exec_time_placeholder/${exec_time}s/" \
        -e "s/passed_placeholder/$passed_count/" \
        -e "s/failed_placeholder/$failed_count/" \
        -e "s/skipped_placeholder/$skipped_count/" \
        -e "s/total_placeholder/$((passed_count + failed_count + skipped_count))/" \
        "$json_output" 2>/dev/null || {
        # macOS以外でsedを使用
        sed -i \
            -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
            -e "s/exec_time_placeholder/${exec_time}s/" \
            -e "s/passed_placeholder/$passed_count/" \
            -e "s/failed_placeholder/$failed_count/" \
            -e "s/skipped_placeholder/$skipped_count/" \
            -e "s/total_placeholder/$((passed_count + failed_count + skipped_count))/" \
            "$json_output"
    }
    
    # テキスト形式サマリー生成
    cat > "$summary_output" << EOF
テスト実行結果サマリー
========================
実行時間: ${exec_time}秒
成功テスト数: $passed_count
失敗テスト数: $failed_count
スキップテスト数: $skipped_count
合計テスト数: $((passed_count + failed_count + skipped_count))

実行時刻: $(date)
EOF
    
    log_success "テスト結果JSON: $json_output"
    log_success "テスト結果サマリー: $summary_output"
    
    # 失敗テストの詳細分析（失敗がある場合のみ）
    if [ $failed_count -gt 0 ]; then
        analyze_failed_tests "$test_output"
    fi
}

# 失敗テスト詳細分析
analyze_failed_tests() {
    local test_output="$1"
    local failed_detail_file="$REPORTS_DIR/failed_tests_detail.txt"
    
    log_info "失敗テスト詳細を分析中..."
    
    cat > "$failed_detail_file" << EOF
失敗テスト詳細分析
==================

EOF
    
    # 失敗テストの詳細を抽出
    if [ -f "$test_output" ]; then
        grep -A 10 "FAIL:" "$test_output" >> "$failed_detail_file" 2>/dev/null || echo "詳細情報の抽出に失敗しました" >> "$failed_detail_file"
    fi
    
    log_success "失敗テスト詳細: $failed_detail_file"
}

# メイン実行
main() {
    log_info "🧪 テスト分析開始"
    
    # 前提条件チェック
    check_prerequisites
    
    # 必要なディレクトリ作成
    ensure_directories "$REPORTS_DIR" "$TMP_DIR"
    
    # プロジェクトルートに移動
    cd "$PROJECT_ROOT"
    
    # Task 4: テスト実行結果分析機能を実行
    if analyze_test_execution_results; then
        log_success "テスト実行結果分析完了"
    else
        log_warning "テスト実行結果分析で問題が発生しました（処理継続）"
    fi
    
    # カバレッジ分析実行
    if run_coverage_analysis; then
        log_success "カバレッジ分析完了"
    else
        log_warning "カバレッジ分析で問題が発生しました（処理継続）"
    fi
    
    log_success "テスト分析完了"
}

main "$@"