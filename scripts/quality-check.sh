#!/bin/bash

# メイン品質チェックオーケストレーションスクリプト
# プロジェクトの包括的な品質とパフォーマンス分析を実行

set -e

# スクリプトディレクトリの取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 共通関数の読み込み
source "$SCRIPT_DIR/utils.sh"

# 設定
REPORTS_DIR="$PROJECT_ROOT/reports"
TMP_DIR="$PROJECT_ROOT/tmp"

# パフォーマンス最適化設定
PARALLEL_EXECUTION=true
MAX_PARALLEL_JOBS=4
TIMEOUT_SECONDS=300
LARGE_PROJECT_THRESHOLD=1000  # ファイル数の閾値

# 実行フラグ
RUN_TEST=false
RUN_QUALITY=false
RUN_PERF=false
RUN_ALL=false

# 使用方法表示
show_help() {
    cat << EOF
品質チェックスクリプト - プロジェクトの包括的な品質とパフォーマンス分析

使用方法:
    $0 [オプション]

オプション:
    --test      テスト分析のみ実行（カバレッジ、テスト結果）
    --quality   品質検証のみ実行（静的解析、セキュリティ）
    --perf      パフォーマンス測定のみ実行（ベンチマーク、プロファイル）
    --all       全ての分析を実行（デフォルト）
    --help      このヘルプを表示

例:
    $0 --test           # テスト分析のみ
    $0 --quality        # 品質検証のみ  
    $0 --perf           # パフォーマンス測定のみ
    $0 --all            # 全分析実行
    $0                  # 全分析実行（--allと同じ）

EOF
}

# 引数解析
parse_arguments() {
    if [ $# -eq 0 ]; then
        RUN_ALL=true
        return
    fi

    while [ $# -gt 0 ]; do
        case $1 in
            --test)
                RUN_TEST=true
                ;;
            --quality)
                RUN_QUALITY=true
                ;;
            --perf)
                RUN_PERF=true
                ;;
            --all)
                RUN_ALL=true
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "不明なオプション: $1"
                show_help
                exit 1
                ;;
        esac
        shift
    done
}

# プログレス表示
show_progress() {
    local current=$1
    local total=$2
    local task="$3"
    
    echo
    log_info "[$current/$total] $task"
    echo "progress: $(( (current * 100) / total ))%"
}

# 大規模プロジェクト検出
detect_project_scale() {
    local go_file_count=$(find "$PROJECT_ROOT" -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" | wc -l | tr -d ' ')
    log_info "Goファイル数: $go_file_count 件"
    
    if [ "$go_file_count" -gt "$LARGE_PROJECT_THRESHOLD" ]; then
        log_info "大規模プロジェクトを検出しました（$go_file_count ファイル）"
        log_info "最適化モードを有効化します"
        return 0  # 大規模プロジェクト
    else
        log_info "中小規模プロジェクト（$go_file_count ファイル）"
        return 1  # 通常規模
    fi
}

# 並列実行機能付きバックグラウンド実行
run_parallel_analysis() {
    local tasks=("$@")
    local pids=()
    local task_names=()
    
    log_info "並列分析を開始（最大並列数: $MAX_PARALLEL_JOBS）"
    
    local job_count=0
    for task in "${tasks[@]}"; do
        if [ $job_count -ge $MAX_PARALLEL_JOBS ]; then
            # 並列数制限のため一つ完了を待つ
            wait ${pids[0]}
            unset pids[0]
            unset task_names[0]
            pids=("${pids[@]}")
            task_names=("${task_names[@]}")
            job_count=$((job_count - 1))
        fi
        
        case $task in
            "test")
                timeout $TIMEOUT_SECONDS "$SCRIPT_DIR/test-analysis.sh" &
                ;;
            "quality") 
                timeout $TIMEOUT_SECONDS "$SCRIPT_DIR/code-quality.sh" &
                ;;
            "performance")
                timeout $TIMEOUT_SECONDS "$SCRIPT_DIR/performance-check.sh" &
                ;;
            *)
                log_warning "不明なタスク: $task"
                continue
                ;;
        esac
        
        local pid=$!
        pids+=($pid)
        task_names+=($task)
        job_count=$((job_count + 1))
        
        log_info "バックグラウンド実行開始: $task (PID: $pid)"
    done
    
    # 全ジョブの完了を待つ
    for i in "${!pids[@]}"; do
        local pid=${pids[$i]}
        local task_name=${task_names[$i]}
        
        if wait $pid; then
            log_success "$task_name 分析完了"
        else
            log_warning "$task_name 分析でエラーまたはタイムアウト（処理継続）"
        fi
    done
}

# 最適化されたリソース使用量モニタリング
monitor_resource_usage() {
    local start_time=$(date +%s)
    local start_memory=0
    
    # macOSとLinuxで異なるメモリ取得方法
    if [ "$(uname)" = "Darwin" ]; then
        start_memory=$(ps -o rss= -p $$ | tr -d ' ')
    else
        start_memory=$(ps -o rss= -p $$ | tr -d ' ')
    fi
    
    return 0  # monitoring setup complete
}

# テスト分析実行
run_test_analysis() {
    show_progress 1 3 "テスト分析を実行中..."
    
    # テスト分析スクリプトが存在する場合のみ実行
    if [ -f "$SCRIPT_DIR/test-analysis.sh" ]; then
        run_with_error_handling "テスト分析" "$SCRIPT_DIR/test-analysis.sh"
    else
        log_warning "test-analysis.sh が見つかりません"
    fi
}

# 品質検証実行（プレースホルダー）
run_quality_checks() {
    show_progress 2 3 "品質検証を実行中..."
    log_info "品質検証機能は今後実装予定です"
}

# パフォーマンス測定実行（プレースホルダー）
run_performance_tests() {
    show_progress 3 3 "パフォーマンス測定を実行中..."
    
    # ベンチマーク実行スクリプトを呼び出し
    if [ -f "$SCRIPT_DIR/performance-check.sh" ]; then
        log_info "ベンチマーク測定を実行中..."
        if "$SCRIPT_DIR/performance-check.sh"; then
            log_success "ベンチマーク測定完了"
        else
            log_warning "ベンチマーク測定で問題が発生しました（処理継続）"
        fi
    else
        log_warning "performance-check.sh が見つかりません（スキップ）"
    fi
}

# メイン実行
main() {
    echo "🔍 Go品質・パフォーマンスチェック開始"
    echo "========================================="
    
    # 引数解析
    parse_arguments "$@"
    
    # 前提条件チェック
    check_prerequisites
    
    # 必要なディレクトリ作成
    ensure_directories "$REPORTS_DIR" "$TMP_DIR"
    
    # プロジェクトルートに移動
    cd "$PROJECT_ROOT"
    
    # 実行時間とリソース使用量のモニタリング開始
    local start_time=$(date +%s)
    monitor_resource_usage
    
    # プロジェクト規模の検出と最適化モード選択
    local is_large_project=false
    if detect_project_scale; then
        is_large_project=true
        log_info "大規模プロジェクト対応モードで実行します"
    fi
    
    # 並列実行可能な分析の実行
    if [ "$RUN_ALL" = true ]; then
        log_info "全分析を実行します"
        
        if [ "$PARALLEL_EXECUTION" = true ] && [ "$is_large_project" = true ]; then
            # 大規模プロジェクトでは並列実行
            run_parallel_analysis "test" "quality" "performance"
        else
            # 通常の順次実行
            run_test_analysis
            run_quality_checks  
            run_performance_tests
        fi
    else
        # 個別オプション実行
        local analysis_tasks=()
        
        if [ "$RUN_TEST" = true ]; then
            analysis_tasks+=("test")
        fi
        
        if [ "$RUN_QUALITY" = true ]; then
            analysis_tasks+=("quality")
        fi
        
        if [ "$RUN_PERF" = true ]; then
            analysis_tasks+=("performance")
        fi
        
        # 複数タスクがある場合は並列実行を検討
        if [ ${#analysis_tasks[@]} -gt 1 ] && [ "$PARALLEL_EXECUTION" = true ]; then
            run_parallel_analysis "${analysis_tasks[@]}"
        else
            # 単一タスクまたは並列実行無効の場合は順次実行
            for task in "${analysis_tasks[@]}"; do
                case $task in
                    "test")
                        run_test_analysis
                        ;;
                    "quality")
                        run_quality_checks
                        ;;
                    "performance")
                        run_performance_tests
                        ;;
                esac
            done
        fi
    fi
    
    # 実行時間とリソース使用量の計算・報告
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # パフォーマンスサマリー生成
    cat > "$REPORTS_DIR/performance_summary.json" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "execution_time": "${duration}s",
    "project_scale": "$(if [ "$is_large_project" = true ]; then echo "large"; else echo "normal"; fi)",
    "parallel_execution": $PARALLEL_EXECUTION,
    "max_parallel_jobs": $MAX_PARALLEL_JOBS,
    "timeout_seconds": $TIMEOUT_SECONDS,
    "optimization_applied": $is_large_project
}
EOF
    
    echo
    echo "========================================="
    log_success "分析完了! 実行時間: ${duration}秒"
    if [ "$is_large_project" = true ]; then
        log_info "大規模プロジェクト最適化が適用されました"
    fi
    log_info "詳細結果: $REPORTS_DIR/ 内のレポートファイルを確認してください"
    echo
}

main "$@"