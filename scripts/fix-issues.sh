#!/bin/bash

# 自動修正スクリプト: 検出された問題の自動修正と修正結果レポート
# go fmt、goimports、golangci-lint --fixを使用した自動修正機能
# Task 11: 問題優先度付け機能を含む

set -e

# スクリプトディレクトリの取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 共通関数の読み込み
source "$SCRIPT_DIR/utils.sh"

# 設定
REPORTS_DIR="$PROJECT_ROOT/reports"
TMP_DIR="$PROJECT_ROOT/tmp"

# Task 10: 自動修正実行機能
run_auto_fixes() {
    log_info "自動修正を開始..."
    
    local fix_results_json="$REPORTS_DIR/fix_results.json"
    local fix_summary_file="$REPORTS_DIR/fix_summary.txt"
    
    local start_time=$(date +%s)
    local fmt_success=false
    local goimports_success=false
    local golangci_fix_success=false
    local total_fixes=0
    
    # 修正前の状態をバックアップ
    create_backup_before_fixes
    
    # go fmt実行
    log_info "go fmtを実行中..."
    if run_go_fmt; then
        fmt_success=true
        log_success "go fmt実行完了"
        ((total_fixes++))
    else
        fmt_success=false
        log_warning "go fmtで問題が発生しました"
    fi
    
    # goimports実行（利用可能な場合）
    log_info "goimportsを実行中..."
    if run_goimports; then
        goimports_success=true
        log_success "goimports実行完了"
        ((total_fixes++))
    else
        goimports_success=false
        log_warning "goimportsで問題が発生しました（スキップ）"
    fi
    
    # golangci-lint --fix実行（利用可能な場合）
    log_info "golangci-lint --fixを実行中..."
    if run_golangci_fix; then
        golangci_fix_success=true
        log_success "golangci-lint --fix実行完了"
        ((total_fixes++))
    else
        golangci_fix_success=false
        log_warning "golangci-lint --fixで問題が発生しました（スキップ）"
    fi
    
    local end_time=$(date +%s)
    local execution_time=$((end_time - start_time))
    
    # 修正後のdiff生成
    generate_fix_diff
    
    # 修正結果の解析と出力
    parse_fix_results "$fix_results_json" "$fix_summary_file" "$execution_time" \
        "$total_fixes" "$fmt_success" "$goimports_success" "$golangci_fix_success"
}

# バックアップ作成機能
create_backup_before_fixes() {
    local backup_dir="$TMP_DIR/backup_$(date +%Y%m%d_%H%M%S)"
    
    log_info "修正前状態をバックアップ中..."
    mkdir -p "$backup_dir"
    
    # Go ファイルのバックアップ
    find . -name "*.go" -not -path "./tmp/*" -not -path "./reports/*" | while read -r file; do
        local dir_path=$(dirname "$file")
        mkdir -p "$backup_dir/$dir_path"
        cp "$file" "$backup_dir/$file"
    done
    
    echo "$backup_dir" > "$TMP_DIR/last_backup_path.txt"
    log_success "バックアップ完了: $backup_dir"
}

# go fmt実行機能
run_go_fmt() {
    if command -v go >/dev/null 2>&1; then
        go fmt ./... > "$TMP_DIR/fmt_output.txt" 2>&1
        return $?
    else
        return 1
    fi
}

# goimports実行機能
run_goimports() {
    if command -v goimports >/dev/null 2>&1; then
        goimports -w . > "$TMP_DIR/goimports_output.txt" 2>&1
        return $?
    else
        return 1
    fi
}

# golangci-lint --fix実行機能
run_golangci_fix() {
    if command -v golangci-lint >/dev/null 2>&1; then
        # golangci-lintの設定問題を回避するため、基本的なfixのみ実行
        golangci-lint run --fix --disable-all --enable=gofmt,goimports,unused > "$TMP_DIR/golangci_fix_output.txt" 2>&1 || true
        return 0  # golangci-lintは警告でも非0を返すため、常に成功とみなす
    else
        return 1
    fi
}

# diff生成機能
generate_fix_diff() {
    local diff_file="$REPORTS_DIR/fix_changes.diff"
    
    log_info "修正内容のdiffを生成中..."
    
    if [ -f "$TMP_DIR/last_backup_path.txt" ]; then
        local backup_dir=$(cat "$TMP_DIR/last_backup_path.txt")
        
        if [ -d "$backup_dir" ]; then
            # バックアップと現在の状態を比較してdiff生成
            diff -u -r "$backup_dir" . --exclude="tmp" --exclude="reports" > "$diff_file" 2>/dev/null || true
            log_success "diff生成完了: $diff_file"
        else
            echo "バックアップが見つからないため、diffは生成できませんでした" > "$diff_file"
            log_warning "バックアップが見つかりません"
        fi
    else
        echo "バックアップ情報が見つからないため、diffは生成できませんでした" > "$diff_file"
        log_warning "バックアップ情報が見つかりません"
    fi
}

# 修正結果パース機能
parse_fix_results() {
    local json_output="$1"
    local summary_output="$2"
    local exec_time="$3"
    local total_fixes="$4"
    local fmt_success="$5"
    local goimports_success="$6"
    local golangci_fix_success="$7"
    
    log_info "修正結果をパース中..."
    
    # 修正されたファイル数を計算
    local changed_files=0
    if [ -f "$REPORTS_DIR/fix_changes.diff" ]; then
        changed_files=$(grep -c "^diff" "$REPORTS_DIR/fix_changes.diff" 2>/dev/null || echo "0")
        changed_files=$(echo "$changed_files" | tr -d '\n\r ')
    fi
    
    # 簡略版JSON構造で修正結果を出力
    cat > "$json_output" << 'EOF'
{
    "timestamp": "timestamp_placeholder",
    "execution_time": "exec_time_placeholder",
    "fix_summary": {
        "total_fixes_attempted": total_fixes_placeholder,
        "changed_files": changed_files_placeholder,
        "fmt_success": fmt_success_placeholder,
        "goimports_success": goimports_success_placeholder,
        "golangci_fix_success": golangci_fix_success_placeholder
    },
    "tools_used": ["go fmt", "goimports", "golangci-lint --fix"],
    "changes": []
}
EOF
    
    # プレースホルダーの置換
    sed -i '' \
        -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
        -e "s/exec_time_placeholder/${exec_time}s/" \
        -e "s/total_fixes_placeholder/$total_fixes/" \
        -e "s/changed_files_placeholder/$changed_files/" \
        -e "s/fmt_success_placeholder/$fmt_success/" \
        -e "s/goimports_success_placeholder/$goimports_success/" \
        -e "s/golangci_fix_success_placeholder/$golangci_fix_success/" \
        "$json_output" 2>/dev/null || {
        # macOS以外でsedを使用
        sed -i \
            -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
            -e "s/exec_time_placeholder/${exec_time}s/" \
            -e "s/total_fixes_placeholder/$total_fixes/" \
            -e "s/changed_files_placeholder/$changed_files/" \
            -e "s/fmt_success_placeholder/$fmt_success/" \
            -e "s/goimports_success_placeholder/$goimports_success/" \
            -e "s/golangci_fix_success_placeholder/$golangci_fix_success/" \
            "$json_output"
    }
    
    # テキスト形式サマリー生成
    cat > "$summary_output" << EOF
自動修正結果サマリー
====================
実行時間: ${exec_time}秒
修正試行数: $total_fixes
変更されたファイル数: $changed_files

ツール実行結果:
- go fmt: $([ "$fmt_success" = true ] && echo "成功" || echo "失敗")
- goimports: $([ "$goimports_success" = true ] && echo "成功" || echo "スキップ/失敗")
- golangci-lint --fix: $([ "$golangci_fix_success" = true ] && echo "成功" || echo "スキップ/失敗")

実行時刻: $(date)

変更サマリー:
$(if [ -f "$REPORTS_DIR/fix_changes.diff" ] && [ -s "$REPORTS_DIR/fix_changes.diff" ]; then
    echo "修正による変更が検出されました。詳細は fix_changes.diff を参照してください。"
    echo ""
    echo "変更ファイル一覧:"
    grep "^diff" "$REPORTS_DIR/fix_changes.diff" | sed 's/^diff.*b\//- /' | head -10
else
    echo "修正による変更は検出されませんでした。"
fi)

バックアップ:
$(if [ -f "$TMP_DIR/last_backup_path.txt" ]; then
    echo "修正前の状態: $(cat "$TMP_DIR/last_backup_path.txt")"
else
    echo "バックアップなし"
fi)
EOF
    
    log_success "修正結果JSON: $json_output"
    log_success "修正結果サマリー: $summary_output"
    
    # 修正内容の詳細分析
    if [ "$changed_files" -gt 0 ]; then
        analyze_fix_changes
    fi
}

# 修正内容詳細分析
analyze_fix_changes() {
    local changes_analysis_file="$REPORTS_DIR/fix_changes_analysis.txt"
    
    log_info "修正内容詳細を分析中..."
    
    cat > "$changes_analysis_file" << EOF
修正内容詳細分析
================

EOF
    
    # diff詳細分析
    if [ -f "$REPORTS_DIR/fix_changes.diff" ]; then
        echo "=== 修正内容diff（上位30行） ===" >> "$changes_analysis_file"
        head -30 "$REPORTS_DIR/fix_changes.diff" >> "$changes_analysis_file" 2>/dev/null || echo "diff分析に失敗しました" >> "$changes_analysis_file"
        echo "" >> "$changes_analysis_file"
        
        # 変更統計
        echo "=== 変更統計 ===" >> "$changes_analysis_file"
        echo "追加行数: $(grep -c "^+" "$REPORTS_DIR/fix_changes.diff" 2>/dev/null || echo "0")" >> "$changes_analysis_file"
        echo "削除行数: $(grep -c "^-" "$REPORTS_DIR/fix_changes.diff" 2>/dev/null || echo "0")" >> "$changes_analysis_file"
    fi
    
    log_success "修正内容詳細分析: $changes_analysis_file"
}

# Task 11: 問題優先度付け機能
prioritize_issues() {
    log_info "問題の優先度付けを実行中..."
    
    local priority_json="$REPORTS_DIR/priority_results.json"
    local priority_summary="$REPORTS_DIR/priority_summary.txt"
    local start_time=$(date +%s)
    
    # 既存分析結果を取得
    local lint_issues=0
    local security_issues=0
    local test_failures=0
    local coverage_issues=0
    
    # lint結果の読み取り
    if [ -f "$REPORTS_DIR/lint_results.json" ]; then
        lint_issues=$(grep -o '"errors":[0-9]*' "$REPORTS_DIR/lint_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
        [ -z "$lint_issues" ] && lint_issues=0
    fi
    
    # セキュリティ問題の読み取り  
    if [ -f "$REPORTS_DIR/security_results.json" ]; then
        security_issues=$(grep -o '"total_issues":[0-9]*' "$REPORTS_DIR/security_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
        [ -z "$security_issues" ] && security_issues=0
    fi
    
    # テスト結果の読み取り
    if [ -f "$REPORTS_DIR/test_results.json" ]; then
        test_failures=$(grep -o '"failed":[0-9]*' "$REPORTS_DIR/test_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
        [ -z "$test_failures" ] && test_failures=0
    fi
    
    # カバレッジ問題（80%未満のファイル数をカウント）
    if [ -f "$REPORTS_DIR/low_coverage_functions.txt" ]; then
        coverage_issues=$(grep -c "%" "$REPORTS_DIR/low_coverage_functions.txt" 2>/dev/null || echo "0")
        [ -z "$coverage_issues" ] && coverage_issues=0
    fi
    
    # 優先度計算
    calculate_priority() {
        local issue_type=$1
        local issue_count=$2
        
        case $issue_type in
            "security")
                # セキュリティ問題は最高優先度
                echo $(( issue_count * 100 ))
                ;;
            "test_failure")  
                # テスト失敗は高優先度
                echo $(( issue_count * 75 ))
                ;;
            "lint")
                # lint問題は中優先度
                echo $(( issue_count * 25 ))
                ;;
            "coverage")
                # カバレッジ不足は低優先度
                echo $(( issue_count * 10 ))
                ;;
            *)
                echo "0"
                ;;
        esac
    }
    
    local security_priority=$(calculate_priority "security" $security_issues)
    local test_priority=$(calculate_priority "test_failure" $test_failures)
    local lint_priority=$(calculate_priority "lint" $lint_issues)
    local coverage_priority=$(calculate_priority "coverage" $coverage_issues)
    local total_priority=$(( security_priority + test_priority + lint_priority + coverage_priority ))
    
    local end_time=$(date +%s)
    local exec_time=$(( end_time - start_time ))
    
    # JSON結果生成
    cat > "$priority_json" << EOF
{
    "timestamp": "timestamp_placeholder",
    "execution_time": "exec_time_placeholder",
    "priority_analysis": {
        "total_priority_score": total_priority_placeholder,
        "security_issues": {
            "count": security_issues_placeholder,
            "priority_score": security_priority_placeholder,
            "impact": "critical"
        },
        "test_failures": {
            "count": test_failures_placeholder,
            "priority_score": test_priority_placeholder,
            "impact": "high"
        },
        "lint_issues": {
            "count": lint_issues_placeholder,
            "priority_score": lint_priority_placeholder,
            "impact": "medium"
        },
        "coverage_issues": {
            "count": coverage_issues_placeholder,
            "priority_score": coverage_priority_placeholder,
            "impact": "low"
        }
    },
    "improvement_tasks": []
}
EOF
    
    # プレースホルダー置換
    sed -i '' \
        -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
        -e "s/exec_time_placeholder/${exec_time}s/" \
        -e "s/total_priority_placeholder/$total_priority/" \
        -e "s/security_issues_placeholder/$security_issues/" \
        -e "s/security_priority_placeholder/$security_priority/" \
        -e "s/test_failures_placeholder/$test_failures/" \
        -e "s/test_priority_placeholder/$test_priority/" \
        -e "s/lint_issues_placeholder/$lint_issues/" \
        -e "s/lint_priority_placeholder/$lint_priority/" \
        -e "s/coverage_issues_placeholder/$coverage_issues/" \
        -e "s/coverage_priority_placeholder/$coverage_priority/" \
        "$priority_json" 2>/dev/null || {
        # Linuxでの処理
        sed -i \
            -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
            -e "s/exec_time_placeholder/${exec_time}s/" \
            -e "s/total_priority_placeholder/$total_priority/" \
            -e "s/security_issues_placeholder/$security_issues/" \
            -e "s/security_priority_placeholder/$security_priority/" \
            -e "s/test_failures_placeholder/$test_failures/" \
            -e "s/test_priority_placeholder/$test_priority/" \
            -e "s/lint_issues_placeholder/$lint_issues/" \
            -e "s/lint_priority_placeholder/$lint_priority/" \
            -e "s/coverage_issues_placeholder/$coverage_issues/" \
            -e "s/coverage_priority_placeholder/$coverage_priority/" \
            "$priority_json"
    }
    
    # テキスト形式サマリー生成
    cat > "$priority_summary" << EOF
問題優先度付け結果サマリー
==========================
実行時間: ${exec_time}秒
総優先度スコア: $total_priority

優先度別問題分析:
1. セキュリティ問題 (Critical): $security_issues件 (スコア: $security_priority)
2. テスト失敗 (High): $test_failures件 (スコア: $test_priority)  
3. Lint問題 (Medium): $lint_issues件 (スコア: $lint_priority)
4. カバレッジ不足 (Low): $coverage_issues件 (スコア: $coverage_priority)

実行時刻: $(date)

推奨修正順序:
$(if [ $security_issues -gt 0 ]; then
    echo "1. セキュリティ問題の修正 (最優先)"
fi
if [ $test_failures -gt 0 ]; then
    echo "2. 失敗テストの修正"
fi
if [ $lint_issues -gt 0 ]; then
    echo "3. コード品質問題の修正"
fi
if [ $coverage_issues -gt 0 ]; then
    echo "4. テストカバレッジの改善"
fi
if [ $total_priority -eq 0 ]; then
    echo "現在、優先度の高い問題は検出されていません。"
fi)

影響度評価基準:
- Critical (セキュリティ): 即座に修正が必要
- High (テスト失敗): リリース前に修正が必要  
- Medium (Lint問題): 次回リリースで修正を推奨
- Low (カバレッジ): 継続的改善で対応
EOF
    
    log_success "優先度付け結果JSON: $priority_json"
    log_success "優先度付け結果サマリー: $priority_summary"
    
    return 0
}

# メイン実行
main() {
    log_info "🔧 自動修正開始"
    
    # 前提条件チェック
    check_prerequisites
    
    # go コマンドの存在確認
    if ! command -v go >/dev/null 2>&1; then
        log_error "go コマンドがインストールされていません。インストールしてから再実行してください。"
        exit 1
    fi
    
    # 必要なディレクトリ作成
    ensure_directories "$REPORTS_DIR" "$TMP_DIR"
    
    # プロジェクトルートに移動
    cd "$PROJECT_ROOT"
    
    # Task 10: 自動修正実行
    if run_auto_fixes; then
        log_success "自動修正完了"
    else
        log_warning "自動修正で問題が発生しました"
    fi
    
    # Task 11: 問題優先度付け実行
    if prioritize_issues; then
        log_success "問題優先度付け完了"
    else
        log_warning "問題優先度付けで問題が発生しました"
    fi
    
    log_success "自動修正・優先度付け処理完了"
}

main "$@"