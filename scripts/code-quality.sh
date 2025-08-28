#!/bin/bash

# 静的解析スクリプト: golangci-lintを使用したコード品質分析
# 既存の.golangci.yml設定を活用してlint問題を分析・報告

set -e

# スクリプトディレクトリの取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 共通関数の読み込み
source "$SCRIPT_DIR/utils.sh"

# 設定
REPORTS_DIR="$PROJECT_ROOT/reports"
TMP_DIR="$PROJECT_ROOT/tmp"

# セキュリティスキャン実行機能
run_security_scan() {
    log_info "セキュリティスキャンを開始..."
    
    local security_output_file="$TMP_DIR/security_output.txt"
    local security_results_json="$REPORTS_DIR/security_results.json"
    local security_summary_file="$REPORTS_DIR/security_summary.txt"
    
    local start_time=$(date +%s)
    local total_issues=0
    local gosec_issues=0
    local vuln_issues=0
    local gosec_available=false
    local govulncheck_available=false
    
    # gosec実行
    if command -v gosec >/dev/null 2>&1; then
        log_info "gosecを実行中..."
        gosec_available=true
        if gosec -fmt json ./... > "$TMP_DIR/gosec_raw.json" 2>/dev/null; then
            log_success "gosec実行完了"
        else
            log_warning "gosec実行で問題発見"
        fi
        # gosec結果から問題数を計算
        if [ -f "$TMP_DIR/gosec_raw.json" ]; then
            gosec_issues=$(jq '.Issues | length' "$TMP_DIR/gosec_raw.json" 2>/dev/null || echo "0")
        fi
        gosec ./... > "$TMP_DIR/gosec_text.txt" 2>/dev/null || true
    else
        log_warning "gosecが利用できません"
    fi
    
    # govulncheck実行
    if command -v govulncheck >/dev/null 2>&1; then
        log_info "govulncheckを実行中..."
        govulncheck_available=true
        if govulncheck -json ./... > "$TMP_DIR/govulncheck_raw.json" 2>/dev/null; then
            log_success "govulncheck実行完了"
        else
            log_warning "govulncheck実行で問題発見"
        fi
        # govulncheck結果から脆弱性数を計算
        if [ -f "$TMP_DIR/govulncheck_raw.json" ]; then
            vuln_issues=$(grep -c '"finding"' "$TMP_DIR/govulncheck_raw.json" 2>/dev/null || echo "0")
        fi
        govulncheck ./... > "$TMP_DIR/govulncheck_text.txt" 2>/dev/null || true
    else
        log_warning "govulncheckが利用できません"
    fi
    
    local end_time=$(date +%s)
    local execution_time=$((end_time - start_time))
    
    # 総問題数計算
    total_issues=$((gosec_issues + vuln_issues))
    
    # セキュリティスキャン結果の解析と出力
    parse_security_results "$security_results_json" "$security_summary_file" "$execution_time" \
        "$total_issues" "$gosec_issues" "$vuln_issues" "$gosec_available" "$govulncheck_available"
}

# セキュリティスキャン結果パース機能
parse_security_results() {
    local json_output="$1"
    local summary_output="$2"
    local exec_time="$3"
    local total_issues="$4"
    local gosec_issues="$5"
    local vuln_issues="$6"
    local gosec_available="$7"
    local govulncheck_available="$8"
    
    log_info "セキュリティスキャン結果をパース中..."
    
    # 簡略版JSON構造でセキュリティ結果を出力
    cat > "$json_output" << 'EOF'
{
    "timestamp": "timestamp_placeholder",
    "execution_time": "exec_time_placeholder",
    "security_summary": {
        "total_issues": total_placeholder,
        "gosec_issues": gosec_placeholder,
        "vulnerability_issues": vuln_placeholder,
        "gosec_available": gosec_avail_placeholder,
        "govulncheck_available": govulncheck_avail_placeholder
    },
    "tools_used": [],
    "issues": []
}
EOF
    
    # プレースホルダーの置換
    sed -i '' \
        -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
        -e "s/exec_time_placeholder/${exec_time}s/" \
        -e "s/total_placeholder/$total_issues/" \
        -e "s/gosec_placeholder/$gosec_issues/" \
        -e "s/vuln_placeholder/$vuln_issues/" \
        -e "s/gosec_avail_placeholder/$gosec_available/" \
        -e "s/govulncheck_avail_placeholder/$govulncheck_available/" \
        "$json_output" 2>/dev/null || {
        # macOS以外でsedを使用
        sed -i \
            -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
            -e "s/exec_time_placeholder/${exec_time}s/" \
            -e "s/total_placeholder/$total_issues/" \
            -e "s/gosec_placeholder/$gosec_issues/" \
            -e "s/vuln_placeholder/$vuln_issues/" \
            -e "s/gosec_avail_placeholder/$gosec_available/" \
            -e "s/govulncheck_avail_placeholder/$govulncheck_available/" \
            "$json_output"
    }
    
    # テキスト形式サマリー生成
    cat > "$summary_output" << EOF
セキュリティスキャン結果サマリー
====================================
実行時間: ${exec_time}秒
総セキュリティ問題数: $total_issues
gosec問題数: $gosec_issues
脆弱性問題数: $vuln_issues

ツール利用状況:
- gosec: $([ "$gosec_available" = true ] && echo "利用可能" || echo "利用不可")
- govulncheck: $([ "$govulncheck_available" = true ] && echo "利用可能" || echo "利用不可")

実行時刻: $(date)

詳細結果:
$(if [ -f "$TMP_DIR/gosec_text.txt" ]; then echo "=== gosec結果 ==="; head -10 "$TMP_DIR/gosec_text.txt" 2>/dev/null; fi)
$(if [ -f "$TMP_DIR/govulncheck_text.txt" ]; then echo "=== govulncheck結果 ==="; head -10 "$TMP_DIR/govulncheck_text.txt" 2>/dev/null; fi)
EOF
    
    log_success "セキュリティ結果JSON: $json_output"
    log_success "セキュリティ結果サマリー: $summary_output"
    
    # セキュリティ問題の詳細分析（問題がある場合のみ）
    if [ $total_issues -gt 0 ]; then
        analyze_security_issues
    fi
}

# ビルド検証実行機能
run_build_verification() {
    log_info "ビルド検証を開始..."
    
    local build_output_file="$TMP_DIR/build_output.txt"
    local build_results_json="$REPORTS_DIR/build_results.json"
    local build_summary_file="$REPORTS_DIR/build_summary.txt"
    
    local start_time=$(date +%s)
    local total_issues=0
    local build_issues=0
    local vet_issues=0
    local build_success=false
    local vet_success=false
    
    # go build実行
    log_info "go buildを実行中..."
    if go build ./... > "$TMP_DIR/build_raw.txt" 2>&1; then
        build_success=true
        log_success "go build実行完了（問題なし）"
    else
        build_success=false
        log_warning "go build実行完了（問題発見）"
        # エラー数を計算
        if [ -f "$TMP_DIR/build_raw.txt" ]; then
            build_issues=$(grep -c "error\|Error" "$TMP_DIR/build_raw.txt" 2>/dev/null || echo "0")
        fi
    fi
    
    # go vet実行
    log_info "go vetを実行中..."
    if go vet ./... > "$TMP_DIR/vet_raw.txt" 2>&1; then
        vet_success=true
        log_success "go vet実行完了（問題なし）"
    else
        vet_success=false
        log_warning "go vet実行完了（問題発見）"
        # 警告数を計算
        if [ -f "$TMP_DIR/vet_raw.txt" ]; then
            vet_issues=$(grep -c "warning\|vet:" "$TMP_DIR/vet_raw.txt" 2>/dev/null || echo "0")
        fi
    fi
    
    local end_time=$(date +%s)
    local execution_time=$((end_time - start_time))
    
    # 総問題数計算
    total_issues=$((build_issues + vet_issues))
    
    # ビルド検証結果の解析と出力
    parse_build_results "$build_results_json" "$build_summary_file" "$execution_time" \
        "$total_issues" "$build_issues" "$vet_issues" "$build_success" "$vet_success"
}

# ビルド検証結果パース機能
parse_build_results() {
    local json_output="$1"
    local summary_output="$2"
    local exec_time="$3"
    local total_issues="$4"
    local build_issues="$5"
    local vet_issues="$6"
    local build_success="$7"
    local vet_success="$8"
    
    log_info "ビルド検証結果をパース中..."
    
    # 簡略版JSON構造でビルド結果を出力
    cat > "$json_output" << 'EOF'
{
    "timestamp": "timestamp_placeholder",
    "execution_time": "exec_time_placeholder",
    "build_summary": {
        "total_issues": total_placeholder,
        "build_errors": build_placeholder,
        "vet_warnings": vet_placeholder,
        "build_success": build_success_placeholder,
        "vet_success": vet_success_placeholder
    },
    "tools_used": ["go build", "go vet"],
    "issues": []
}
EOF
    
    # プレースホルダーの置換
    sed -i '' \
        -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
        -e "s/exec_time_placeholder/${exec_time}s/" \
        -e "s/total_placeholder/$total_issues/" \
        -e "s/build_placeholder/$build_issues/" \
        -e "s/vet_placeholder/$vet_issues/" \
        -e "s/build_success_placeholder/$build_success/" \
        -e "s/vet_success_placeholder/$vet_success/" \
        "$json_output" 2>/dev/null || {
        # macOS以外でsedを使用
        sed -i \
            -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
            -e "s/exec_time_placeholder/${exec_time}s/" \
            -e "s/total_placeholder/$total_issues/" \
            -e "s/build_placeholder/$build_issues/" \
            -e "s/vet_placeholder/$vet_issues/" \
            -e "s/build_success_placeholder/$build_success/" \
            -e "s/vet_success_placeholder/$vet_success/" \
            "$json_output"
    }
    
    # テキスト形式サマリー生成
    cat > "$summary_output" << EOF
ビルド検証結果サマリー
========================
実行時間: ${exec_time}秒
総問題数: $total_issues
ビルドエラー数: $build_issues
go vet警告数: $vet_issues

ツール実行結果:
- go build: $([ "$build_success" = true ] && echo "成功" || echo "失敗")
- go vet: $([ "$vet_success" = true ] && echo "成功" || echo "失敗")

実行時刻: $(date)

詳細結果:
$(if [ -f "$TMP_DIR/build_raw.txt" ]; then echo "=== go build結果 ==="; head -10 "$TMP_DIR/build_raw.txt" 2>/dev/null; fi)
$(if [ -f "$TMP_DIR/vet_raw.txt" ]; then echo "=== go vet結果 ==="; head -10 "$TMP_DIR/vet_raw.txt" 2>/dev/null; fi)
EOF
    
    log_success "ビルド結果JSON: $json_output"
    log_success "ビルド結果サマリー: $summary_output"
    
    # ビルド問題の詳細分析（問題がある場合のみ）
    if [ $total_issues -gt 0 ]; then
        analyze_build_issues
    fi
}

# ビルド問題詳細分析
analyze_build_issues() {
    local build_detail_file="$REPORTS_DIR/build_issues_detail.txt"
    
    log_info "ビルド問題詳細を分析中..."
    
    cat > "$build_detail_file" << EOF
ビルド問題詳細分析
==================

EOF
    
    # ビルドエラーの詳細を追加
    if [ -f "$TMP_DIR/build_raw.txt" ]; then
        echo "=== go build詳細結果 ===" >> "$build_detail_file"
        head -50 "$TMP_DIR/build_raw.txt" >> "$build_detail_file" 2>/dev/null || echo "ビルド詳細情報の抽出に失敗しました" >> "$build_detail_file"
        echo "" >> "$build_detail_file"
    fi
    
    # go vet警告の詳細を追加
    if [ -f "$TMP_DIR/vet_raw.txt" ]; then
        echo "=== go vet詳細結果 ===" >> "$build_detail_file"
        head -50 "$TMP_DIR/vet_raw.txt" >> "$build_detail_file" 2>/dev/null || echo "go vet詳細情報の抽出に失敗しました" >> "$build_detail_file"
    fi
    
    log_success "ビルド問題詳細: $build_detail_file"
}

# セキュリティ問題詳細分析
analyze_security_issues() {
    local security_detail_file="$REPORTS_DIR/security_issues_detail.txt"
    
    log_info "セキュリティ問題詳細を分析中..."
    
    cat > "$security_detail_file" << EOF
セキュリティ問題詳細分析
========================

EOF
    
    # gosecの詳細を追加
    if [ -f "$TMP_DIR/gosec_text.txt" ]; then
        echo "=== gosec詳細結果 ===" >> "$security_detail_file"
        head -50 "$TMP_DIR/gosec_text.txt" >> "$security_detail_file" 2>/dev/null || echo "gosec詳細情報の抽出に失敗しました" >> "$security_detail_file"
        echo "" >> "$security_detail_file"
    fi
    
    # govulncheckの詳細を追加
    if [ -f "$TMP_DIR/govulncheck_text.txt" ]; then
        echo "=== govulncheck詳細結果 ===" >> "$security_detail_file"
        head -50 "$TMP_DIR/govulncheck_text.txt" >> "$security_detail_file" 2>/dev/null || echo "govulncheck詳細情報の抽出に失敗しました" >> "$security_detail_file"
    fi
    
    log_success "セキュリティ問題詳細: $security_detail_file"
}

# 静的解析実行機能
run_static_analysis() {
    log_info "静的解析を開始..."
    
    local lint_output_file="$TMP_DIR/lint_output.txt"
    local lint_results_json="$REPORTS_DIR/lint_results.json"
    local lint_summary_file="$REPORTS_DIR/lint_summary.txt"
    
    # golangci-lint実行
    log_info "golangci-lintを実行中..."
    local start_time=$(date +%s)
    
    # golangci-lint実行（JSON形式とテキスト形式の両方を取得）
    if golangci-lint run --out-format json > "$TMP_DIR/lint_raw.json" 2>&1; then
        local exit_code=0
        log_success "golangci-lint実行完了（問題なし）"
    else
        local exit_code=$?
        log_warning "golangci-lint実行完了（問題発見）"
    fi
    
    # テキスト形式の結果も取得
    golangci-lint run > "$lint_output_file" 2>&1 || true
    
    local end_time=$(date +%s)
    local execution_time=$((end_time - start_time))
    
    # 結果解析
    parse_lint_results "$TMP_DIR/lint_raw.json" "$lint_output_file" "$lint_results_json" "$lint_summary_file" "$execution_time" "$exit_code"
}

# lint結果パース機能
parse_lint_results() {
    local raw_json="$1"
    local text_output="$2"
    local json_output="$3"
    local summary_output="$4"
    local exec_time="$5"
    local exit_code="$6"
    
    log_info "lint結果をパース中..."
    
    # 問題カウント（テキスト出力から計算）
    local warning_count=0
    local error_count=0
    local total_issues=0
    
    if [ -f "$text_output" ]; then
        warning_count=$(grep -c "warning" "$text_output" 2>/dev/null || echo "0")
        error_count=$(grep -c "error" "$text_output" 2>/dev/null || echo "0")
        # 数値の正規化（改行削除）
        warning_count=$(echo "$warning_count" | tr -d '\n\r ')
        error_count=$(echo "$error_count" | tr -d '\n\r ')
        # 重複計算を修正（エラーメッセージに":"が含まれるため）
        total_issues=$((warning_count + error_count))
    fi
    
    # 簡略版JSON構造でlint結果を出力
    cat > "$json_output" << 'EOF'
{
    "timestamp": "timestamp_placeholder",
    "execution_time": "exec_time_placeholder",
    "exit_code": exit_code_placeholder,
    "lint_summary": {
        "total_issues": total_placeholder,
        "warnings": warning_placeholder,
        "errors": error_placeholder,
        "files_analyzed": 0
    },
    "issues": []
}
EOF
    
    # プレースホルダーの置換
    sed -i '' \
        -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
        -e "s/exec_time_placeholder/${exec_time}s/" \
        -e "s/exit_code_placeholder/$exit_code/" \
        -e "s/total_placeholder/$total_issues/" \
        -e "s/warning_placeholder/$warning_count/" \
        -e "s/error_placeholder/$error_count/" \
        "$json_output" 2>/dev/null || {
        # macOS以外でsedを使用
        sed -i \
            -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
            -e "s/exec_time_placeholder/${exec_time}s/" \
            -e "s/exit_code_placeholder/$exit_code/" \
            -e "s/total_placeholder/$total_issues/" \
            -e "s/warning_placeholder/$warning_count/" \
            -e "s/error_placeholder/$error_count/" \
            "$json_output"
    }
    
    # テキスト形式サマリー生成
    cat > "$summary_output" << EOF
静的解析結果サマリー
========================
実行時間: ${exec_time}秒
終了コード: $exit_code
総問題数: $total_issues
警告数: $warning_count
エラー数: $error_count

実行時刻: $(date)

詳細結果:
$(head -20 "$text_output" 2>/dev/null || echo "詳細なし")
EOF
    
    log_success "lint結果JSON: $json_output"
    log_success "lint結果サマリー: $summary_output"
    
    # 問題分類の分析（問題がある場合のみ）
    if [ $total_issues -gt 0 ]; then
        analyze_lint_issues "$text_output"
    fi
}

# lint問題分析
analyze_lint_issues() {
    local lint_output="$1"
    local issues_detail_file="$REPORTS_DIR/lint_issues_detail.txt"
    
    log_info "lint問題詳細を分析中..."
    
    cat > "$issues_detail_file" << EOF
Lint問題詳細分析
==================

EOF
    
    # lint問題の詳細を抽出
    if [ -f "$lint_output" ]; then
        head -50 "$lint_output" >> "$issues_detail_file" 2>/dev/null || echo "詳細情報の抽出に失敗しました" >> "$issues_detail_file"
    fi
    
    log_success "lint問題詳細: $issues_detail_file"
}

# メイン実行
main() {
    log_info "🔍 コード品質・セキュリティ分析開始"
    
    # 前提条件チェック
    check_prerequisites
    
    # golangci-lintの存在確認
    if ! command -v golangci-lint >/dev/null 2>&1; then
        log_error "golangci-lintがインストールされていません。インストールしてから再実行してください。"
        exit 1
    fi
    
    # 必要なディレクトリ作成
    ensure_directories "$REPORTS_DIR" "$TMP_DIR"
    
    # プロジェクトルートに移動
    cd "$PROJECT_ROOT"
    
    # ビルド検証実行
    if run_build_verification; then
        log_success "ビルド検証完了"
    else
        log_warning "ビルド検証で問題が発生しました（処理継続）"
    fi
    
    # セキュリティスキャン実行
    if run_security_scan; then
        log_success "セキュリティスキャン完了"
    else
        log_warning "セキュリティスキャンで問題が発生しました（処理継続）"
    fi
    
    # 静的解析実行
    if run_static_analysis; then
        log_success "静的解析完了"
    else
        log_warning "静的解析で問題が発生しました（処理継続）"
    fi
    
    log_success "コード品質・セキュリティ・ビルド分析完了"
}

main "$@"