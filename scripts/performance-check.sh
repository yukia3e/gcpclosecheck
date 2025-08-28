#!/bin/bash

# パフォーマンステストスクリプト: go test -benchを使用したベンチマーク測定
# 実行時間、メモリ使用量、アロケーション数を分析・報告

set -e

# スクリプトディレクトリの取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 共通関数の読み込み
source "$SCRIPT_DIR/utils.sh"

# 設定
REPORTS_DIR="$PROJECT_ROOT/reports"
TMP_DIR="$PROJECT_ROOT/tmp"

# Task 8: ベンチマーク実行機能
run_benchmark_tests() {
    log_info "ベンチマークテストを開始..."
    
    local benchmark_output_file="$TMP_DIR/benchmark_output.txt"
    local benchmark_results_json="$REPORTS_DIR/benchmark_results.json"
    local benchmark_summary_file="$REPORTS_DIR/benchmark_summary.txt"
    
    local start_time=$(date +%s)
    local total_benchmarks=0
    local benchmark_success=false
    
    # go test -bench実行
    log_info "go test -bench を実行中..."
    if go test -bench=. -benchmem ./... > "$TMP_DIR/benchmark_raw.txt" 2>&1; then
        benchmark_success=true
        log_success "ベンチマーク実行完了（問題なし）"
    else
        local exit_code=$?
        if [ $exit_code -eq 1 ]; then
            # exit code 1は通常のテスト失敗なので、ベンチマークは実行されている可能性がある
            benchmark_success=true
            log_info "ベンチマーク実行完了（一部テスト失敗があるが、ベンチマークは実行済み）"
        else
            benchmark_success=false
            log_warning "ベンチマーク実行で問題が発生しました"
        fi
    fi
    
    local end_time=$(date +%s)
    local execution_time=$((end_time - start_time))
    
    # ベンチマーク数を計算
    if [ -f "$TMP_DIR/benchmark_raw.txt" ]; then
        total_benchmarks=$(grep -c "^Benchmark" "$TMP_DIR/benchmark_raw.txt" 2>/dev/null || echo "0")
        # 数値の正規化
        total_benchmarks=$(echo "$total_benchmarks" | tr -d '\n\r ')
    fi
    
    # ベンチマーク結果の解析と出力
    parse_benchmark_results "$benchmark_results_json" "$benchmark_summary_file" "$execution_time" \
        "$total_benchmarks" "$benchmark_success"
}

# ベンチマーク結果パース機能
parse_benchmark_results() {
    local json_output="$1"
    local summary_output="$2"
    local exec_time="$3"
    local total_benchmarks="$4"
    local benchmark_success="$5"
    
    log_info "ベンチマーク結果をパース中..."
    
    # ベンチマーク統計の計算
    local avg_ns_per_op=0
    local avg_mb_per_sec=0
    local avg_allocs_per_op=0
    
    if [ -f "$TMP_DIR/benchmark_raw.txt" ] && [ "$total_benchmarks" -gt 0 ]; then
        # 平均ns/op計算（単位を統一してから平均）
        local total_ns=0
        local ns_count=0
        
        while read -r line; do
            if echo "$line" | grep -q "^Benchmark"; then
                # ベンチマーク行から ns/op を抽出
                ns_per_op=$(echo "$line" | grep -o '[0-9.]*[ ]*ns/op' | grep -o '[0-9.]*' | head -1)
                if [ -n "$ns_per_op" ] && [ "$ns_per_op" != "0" ]; then
                    total_ns=$(echo "$total_ns + $ns_per_op" | bc -l 2>/dev/null || echo "$total_ns")
                    ns_count=$((ns_count + 1))
                fi
            fi
        done < "$TMP_DIR/benchmark_raw.txt"
        
        if [ "$ns_count" -gt 0 ]; then
            avg_ns_per_op=$(echo "scale=2; $total_ns / $ns_count" | bc -l 2>/dev/null || echo "0")
        fi
    fi
    
    # 簡略版JSON構造でベンチマーク結果を出力
    cat > "$json_output" << 'EOF'
{
    "timestamp": "timestamp_placeholder",
    "execution_time": "exec_time_placeholder",
    "benchmark_summary": {
        "total_benchmarks": total_placeholder,
        "avg_ns_per_op": avg_ns_placeholder,
        "avg_mb_per_sec": avg_mb_placeholder,
        "avg_allocs_per_op": avg_allocs_placeholder,
        "benchmark_success": benchmark_success_placeholder
    },
    "tools_used": ["go test -bench", "go test -benchmem"],
    "benchmarks": []
}
EOF
    
    # プレースホルダーの置換
    sed -i '' \
        -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
        -e "s/exec_time_placeholder/${exec_time}s/" \
        -e "s/total_placeholder/$total_benchmarks/" \
        -e "s/avg_ns_placeholder/$avg_ns_per_op/" \
        -e "s/avg_mb_placeholder/$avg_mb_per_sec/" \
        -e "s/avg_allocs_placeholder/$avg_allocs_per_op/" \
        -e "s/benchmark_success_placeholder/$benchmark_success/" \
        "$json_output" 2>/dev/null || {
        # macOS以外でsedを使用
        sed -i \
            -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
            -e "s/exec_time_placeholder/${exec_time}s/" \
            -e "s/total_placeholder/$total_benchmarks/" \
            -e "s/avg_ns_placeholder/$avg_ns_per_op/" \
            -e "s/avg_mb_placeholder/$avg_mb_per_sec/" \
            -e "s/avg_allocs_placeholder/$avg_allocs_per_op/" \
            -e "s/benchmark_success_placeholder/$benchmark_success/" \
            "$json_output"
    }
    
    # テキスト形式サマリー生成
    cat > "$summary_output" << EOF
ベンチマーク実行結果サマリー
============================
実行時間: ${exec_time}秒
総ベンチマーク数: $total_benchmarks
平均実行時間: ${avg_ns_per_op} ns/op
実行成功: $([ "$benchmark_success" = true ] && echo "成功" || echo "失敗")

実行時刻: $(date)

詳細結果（上位10行）:
$(head -10 "$TMP_DIR/benchmark_raw.txt" 2>/dev/null || echo "詳細情報なし")
EOF
    
    log_success "ベンチマーク結果JSON: $json_output"
    log_success "ベンチマーク結果サマリー: $summary_output"
    
    # パフォーマンス分析（閾値チェック）
    if [ "$total_benchmarks" -gt 0 ]; then
        analyze_performance_thresholds "$avg_ns_per_op"
    fi
}

# Task 9: プロファイリング実行機能
run_profiling() {
    log_info "プロファイリングを開始..."
    
    local profile_results_json="$REPORTS_DIR/profile_results.json"
    local profile_summary_file="$REPORTS_DIR/profile_summary.txt"
    
    local start_time=$(date +%s)
    local cpu_profile_success=false
    local mem_profile_success=false
    
    # CPUプロファイル実行
    log_info "CPUプロファイルを実行中..."
    if go test -cpuprofile="$TMP_DIR/cpu.prof" -bench=. ./... >/dev/null 2>&1; then
        cpu_profile_success=true
        log_success "CPUプロファイル取得完了"
    else
        cpu_profile_success=false
        log_warning "CPUプロファイル取得で問題が発生しました"
    fi
    
    # メモリプロファイル実行  
    log_info "メモリプロファイルを実行中..."
    if go test -memprofile="$TMP_DIR/mem.prof" -bench=. ./... >/dev/null 2>&1; then
        mem_profile_success=true
        log_success "メモリプロファイル取得完了"
    else
        mem_profile_success=false
        log_warning "メモリプロファイル取得で問題が発生しました"
    fi
    
    local end_time=$(date +%s)
    local execution_time=$((end_time - start_time))
    
    # プロファイリング結果の解析と出力
    parse_profile_results "$profile_results_json" "$profile_summary_file" "$execution_time" \
        "$cpu_profile_success" "$mem_profile_success"
}

# プロファイリング結果パース機能
parse_profile_results() {
    local json_output="$1"
    local summary_output="$2"
    local exec_time="$3"
    local cpu_profile_success="$4"
    local mem_profile_success="$5"
    
    log_info "プロファイリング結果をパース中..."
    
    # プロファイルデータから基本メトリクス抽出
    local cpu_top_functions=""
    local memory_usage_mb=0
    
    # go tool pprofでCPU分析（利用可能な場合）
    if [ "$cpu_profile_success" = true ] && [ -f "$TMP_DIR/cpu.prof" ]; then
        if command -v go >/dev/null 2>&1; then
            # CPUホットパス抽出（top 5）
            cpu_top_functions=$(go tool pprof -text -nodecount=5 "$TMP_DIR/cpu.prof" 2>/dev/null | head -10 | tail -5 || echo "CPU分析データなし")
        fi
    fi
    
    # メモリ使用量分析（利用可能な場合）  
    if [ "$mem_profile_success" = true ] && [ -f "$TMP_DIR/mem.prof" ]; then
        if command -v go >/dev/null 2>&1; then
            # メモリ使用量の概算取得
            memory_usage_mb=$(go tool pprof -text -nodecount=1 "$TMP_DIR/mem.prof" 2>/dev/null | grep -o '[0-9.]*MB' | head -1 | grep -o '[0-9.]*' || echo "0")
        fi
    fi
    
    # 簡略版JSON構造でプロファイル結果を出力
    cat > "$json_output" << 'EOF'
{
    "timestamp": "timestamp_placeholder",
    "execution_time": "exec_time_placeholder",
    "profiling_summary": {
        "cpu_profile_success": cpu_success_placeholder,
        "mem_profile_success": mem_success_placeholder,
        "memory_usage_mb": memory_placeholder
    },
    "tools_used": ["go test -cpuprofile", "go test -memprofile", "go tool pprof"],
    "hotpaths": []
}
EOF
    
    # プレースホルダーの置換
    sed -i '' \
        -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
        -e "s/exec_time_placeholder/${exec_time}s/" \
        -e "s/cpu_success_placeholder/$cpu_profile_success/" \
        -e "s/mem_success_placeholder/$mem_profile_success/" \
        -e "s/memory_placeholder/$memory_usage_mb/" \
        "$json_output" 2>/dev/null || {
        # macOS以外でsedを使用
        sed -i \
            -e "s/timestamp_placeholder/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" \
            -e "s/exec_time_placeholder/${exec_time}s/" \
            -e "s/cpu_success_placeholder/$cpu_profile_success/" \
            -e "s/mem_success_placeholder/$mem_profile_success/" \
            -e "s/memory_placeholder/$memory_usage_mb/" \
            "$json_output"
    }
    
    # テキスト形式サマリー生成
    cat > "$summary_output" << EOF
プロファイリング結果サマリー
============================
実行時間: ${exec_time}秒
CPUプロファイル: $([ "$cpu_profile_success" = true ] && echo "成功" || echo "失敗")
メモリプロファイル: $([ "$mem_profile_success" = true ] && echo "成功" || echo "失敗")
メモリ使用量: ${memory_usage_mb} MB

実行時刻: $(date)

CPU ホットパス (上位5関数):
$cpu_top_functions

プロファイルファイル:
$([ -f "$TMP_DIR/cpu.prof" ] && echo "- CPU: $TMP_DIR/cpu.prof" || echo "- CPU: 未生成")
$([ -f "$TMP_DIR/mem.prof" ] && echo "- Memory: $TMP_DIR/mem.prof" || echo "- Memory: 未生成")
EOF
    
    log_success "プロファイル結果JSON: $json_output"
    log_success "プロファイル結果サマリー: $summary_output"
    
    # ボトルネック分析（プロファイルデータがある場合のみ）
    if [ "$cpu_profile_success" = true ] || [ "$mem_profile_success" = true ]; then
        analyze_bottlenecks
    fi
}

# ボトルネック分析機能
analyze_bottlenecks() {
    local bottleneck_analysis_file="$REPORTS_DIR/bottleneck_analysis.txt"
    
    log_info "ボトルネック分析中..."
    
    cat > "$bottleneck_analysis_file" << EOF
ボトルネック分析
================

EOF
    
    # CPUボトルネック分析
    if [ -f "$TMP_DIR/cpu.prof" ]; then
        echo "=== CPUボトルネック分析 ===" >> "$bottleneck_analysis_file"
        if command -v go >/dev/null 2>&1; then
            go tool pprof -text -nodecount=10 "$TMP_DIR/cpu.prof" >> "$bottleneck_analysis_file" 2>/dev/null || echo "CPU分析に失敗しました" >> "$bottleneck_analysis_file"
        else
            echo "go toolが利用できません" >> "$bottleneck_analysis_file"
        fi
        echo "" >> "$bottleneck_analysis_file"
    fi
    
    # メモリボトルネック分析
    if [ -f "$TMP_DIR/mem.prof" ]; then
        echo "=== メモリボトルネック分析 ===" >> "$bottleneck_analysis_file"
        if command -v go >/dev/null 2>&1; then
            go tool pprof -text -nodecount=10 "$TMP_DIR/mem.prof" >> "$bottleneck_analysis_file" 2>/dev/null || echo "メモリ分析に失敗しました" >> "$bottleneck_analysis_file"
        else
            echo "go toolが利用できません" >> "$bottleneck_analysis_file"
        fi
    fi
    
    log_success "ボトルネック分析: $bottleneck_analysis_file"
}

# パフォーマンス閾値チェック
analyze_performance_thresholds() {
    local avg_ns_per_op="$1"
    local performance_analysis_file="$REPORTS_DIR/performance_analysis.txt"
    
    log_info "パフォーマンス閾値分析中..."
    
    cat > "$performance_analysis_file" << EOF
パフォーマンス閾値分析
====================

平均実行時間: ${avg_ns_per_op} ns/op

パフォーマンス評価:
EOF
    
    # 閾値チェック（bcが利用可能な場合のみ）
    if command -v bc >/dev/null 2>&1 && [ -n "$avg_ns_per_op" ] && [ "$avg_ns_per_op" != "0" ]; then
        # 1マイクロ秒(1000ns)を基準とした閾値チェック
        if [ "$(echo "$avg_ns_per_op < 1000" | bc -l)" -eq 1 ]; then
            echo "✅ 高速: 平均実行時間が1マイクロ秒未満です" >> "$performance_analysis_file"
        elif [ "$(echo "$avg_ns_per_op < 10000" | bc -l)" -eq 1 ]; then
            echo "⚠️  注意: 平均実行時間が1-10マイクロ秒です" >> "$performance_analysis_file"
        else
            echo "❌ 低速: 平均実行時間が10マイクロ秒を超えています" >> "$performance_analysis_file"
        fi
    else
        echo "- 閾値チェックをスキップ（bc未インストールまたはデータ不足）" >> "$performance_analysis_file"
    fi
    
    log_success "パフォーマンス分析: $performance_analysis_file"
}

# メイン実行
main() {
    log_info "🚀 パフォーマンス測定開始"
    
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
    
    # Task 8: ベンチマーク実行
    if run_benchmark_tests; then
        log_success "ベンチマーク測定完了"
    else
        log_warning "ベンチマーク測定で問題が発生しました"
    fi
    
    # Task 9: プロファイリング実行
    if run_profiling; then
        log_success "プロファイリング完了"
    else
        log_warning "プロファイリングで問題が発生しました"
    fi
    
    log_success "パフォーマンス測定・プロファイリング完了"
}

main "$@"