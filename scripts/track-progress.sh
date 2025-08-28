#!/bin/bash

# 継続改善支援機能: 品質指標履歴管理とトレンド分析
# Requirements: 4.4, 5.3 - 継続改善とトレンド分析

set -e
source "$(dirname "$0")/utils.sh"

# 設定
HISTORY_DIR="reports/history"
TIMESTAMP=$(date '+%Y-%m-%d_%H-%M-%S')
CURRENT_DATE=$(date '+%Y-%m-%d')

# 品質指標履歴管理の実装
track_quality_metrics() {
    log_info "品質指標履歴管理を開始"
    
    # 履歴ディレクトリを作成
    mkdir -p "$HISTORY_DIR"
    
    # 現在の品質データを収集
    collect_current_metrics
    
    # 過去データとの比較分析
    analyze_quality_trends
    
    # 改善効果測定
    measure_improvement_effects
    
    log_info "品質指標履歴管理が完了しました"
}

# 現在の品質メトリクスを収集
collect_current_metrics() {
    log_info "現在の品質メトリクスを収集中..."
    
    local current_metrics_file="$HISTORY_DIR/quality_metrics_$TIMESTAMP.json"
    local execution_start_time=$(date '+%s')
    
    # 既存の分析結果から品質指標を抽出
    local test_coverage="0"
    local test_passed="0"
    local test_failed="0"
    local lint_issues="0"
    local security_issues="0"
    local benchmark_count="0"
    local overall_score="0"
    
    # テスト結果から指標を抽出
    if [ -f "reports/test_results.json" ]; then
        if command -v jq >/dev/null 2>&1; then
            test_coverage=$(jq -r '.test_summary.coverage_percent // 0' reports/test_results.json)
            test_passed=$(jq -r '.test_summary.passed // 0' reports/test_results.json)
            test_failed=$(jq -r '.test_summary.failed // 0' reports/test_results.json)
        else
            # jq が使えない場合は grep で抽出
            test_coverage=$(grep -o '"coverage_percent": *[0-9.]*' reports/test_results.json 2>/dev/null | head -1 | grep -o '[0-9.]*' || echo "0")
            test_passed=$(grep -o '"passed": *[0-9]*' reports/test_results.json 2>/dev/null | head -1 | grep -o '[0-9]*' || echo "0")
            test_failed=$(grep -o '"failed": *[0-9]*' reports/test_results.json 2>/dev/null | head -1 | grep -o '[0-9]*' || echo "0")
        fi
    fi
    
    # 静的解析結果から指標を抽出
    if [ -f "reports/lint_results.json" ]; then
        if command -v jq >/dev/null 2>&1; then
            lint_issues=$(jq -r '.lint_summary.total_issues // 0' reports/lint_results.json)
        else
            lint_issues=$(grep -o '"total_issues": *[0-9]*' reports/lint_results.json 2>/dev/null | head -1 | grep -o '[0-9]*' || echo "0")
        fi
    fi
    
    # セキュリティ結果から指標を抽出
    if [ -f "reports/security_results.json" ]; then
        if command -v jq >/dev/null 2>&1; then
            security_issues=$(jq -r '.security_summary.total_security_issues // 0' reports/security_results.json)
        else
            security_issues=$(grep -o '"total_security_issues": *[0-9]*' reports/security_results.json 2>/dev/null | head -1 | grep -o '[0-9]*' || echo "0")
        fi
    fi
    
    # ベンチマーク結果から指標を抽出
    if [ -f "reports/benchmark_results.json" ]; then
        if command -v jq >/dev/null 2>&1; then
            benchmark_count=$(jq -r '.benchmark_summary.total_benchmarks // 0' reports/benchmark_results.json)
        else
            benchmark_count=$(grep -o '"total_benchmarks": *[0-9]*' reports/benchmark_results.json 2>/dev/null | head -1 | grep -o '[0-9]*' || echo "0")
        fi
    fi
    
    # 品質スコア算出（簡易版）
    overall_score=$(calculate_quality_score "$test_coverage" "$test_failed" "$lint_issues" "$security_issues")
    
    local execution_end_time=$(date '+%s')
    local execution_time=$((execution_end_time - execution_start_time))
    
    # 品質メトリクスをJSONで保存
    cat > "$current_metrics_file" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "execution_time": "${execution_time}s",
    "date": "$CURRENT_DATE",
    "quality_metrics": {
        "test_coverage": $test_coverage,
        "test_passed": $test_passed,
        "test_failed": $test_failed,
        "lint_issues": $lint_issues,
        "security_issues": $security_issues,
        "benchmark_count": $benchmark_count,
        "overall_score": $overall_score
    },
    "file_counts": {
        "go_files": $(find . -name "*.go" -not -path "*/vendor/*" 2>/dev/null | wc -l | tr -d ' '),
        "test_files": $(find . -name "*_test.go" -not -path "*/vendor/*" 2>/dev/null | wc -l | tr -d ' ')
    },
    "repository_stats": {
        "commit_hash": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')",
        "branch": "$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')"
    }
}
EOF
    
    log_info "品質メトリクス収集完了: $current_metrics_file"
}

# 品質スコア算出関数
calculate_quality_score() {
    local coverage="$1"
    local failed_tests="$2"
    local lint_issues="$3" 
    local security_issues="$4"
    
    # 基本スコア（カバレッジベース）
    local base_score
    if [ -z "$coverage" ] || [ "$coverage" = "0" ]; then
        base_score=0
    else
        base_score=$coverage
    fi
    
    # 失敗テストでの減点（1テスト失敗 = -2点）
    local test_penalty=$((failed_tests * 2))
    
    # Lint問題での減点（1問題 = -0.5点）
    local lint_penalty=$((lint_issues / 2))
    
    # セキュリティ問題での大幅減点（1問題 = -10点）
    local security_penalty=$((security_issues * 10))
    
    # 総合スコア算出
    local total_score=$((base_score - test_penalty - lint_penalty - security_penalty))
    
    # スコアを0-100の範囲に制限
    if [ $total_score -lt 0 ]; then
        echo "0"
    elif [ $total_score -gt 100 ]; then
        echo "100"
    else
        echo "$total_score"
    fi
}

# 品質トレンド分析
analyze_quality_trends() {
    log_info "品質トレンド分析を実行中..."
    
    local trend_report="reports/trend_analysis.md"
    local history_files=$(find "$HISTORY_DIR" -name "quality_metrics_*.json" 2>/dev/null | sort)
    
    if [ -z "$history_files" ]; then
        log_warn "履歴データが見つかりません。初回実行のため分析をスキップします。"
        return
    fi
    
    # Markdownレポート生成開始
    cat > "$trend_report" << 'EOF'
# 品質トレンド分析レポート

**生成日時**: $(date)
**分析期間**: 過去の品質データに基づく分析

## 📊 品質指標の推移

### テストカバレッジ推移
```
EOF

    # 履歴データから推移を分析
    local file_count=0
    local latest_score=0
    local previous_score=0
    
    for file in $history_files; do
        if [ -f "$file" ]; then
            local date_str=$(basename "$file" .json | sed 's/quality_metrics_//')
            local score
            
            if command -v jq >/dev/null 2>&1; then
                score=$(jq -r '.quality_metrics.overall_score // 0' "$file")
            else
                score=$(grep -o '"overall_score": *[0-9]*' "$file" 2>/dev/null | head -1 | grep -o '[0-9]*' || echo "0")
            fi
            
            echo "$date_str: $score 点" >> "$trend_report"
            
            if [ $file_count -gt 0 ]; then
                previous_score=$latest_score
            fi
            latest_score=$score
            file_count=$((file_count + 1))
        fi
    done
    
    cat >> "$trend_report" << 'EOF'
```

### 品質改善傾向
EOF
    
    # 品質改善の傾向分析
    if [ $file_count -ge 2 ]; then
        local score_diff=$((latest_score - previous_score))
        
        if [ $score_diff -gt 0 ]; then
            echo "✅ **改善中**: 前回から+${score_diff}点向上" >> "$trend_report"
        elif [ $score_diff -lt 0 ]; then
            echo "❌ **悪化**: 前回から${score_diff}点低下" >> "$trend_report"
        else
            echo "🔄 **安定**: 前回から変化なし" >> "$trend_report"
        fi
    else
        echo "📊 **初回データ**: トレンド分析には複数回の実行が必要です" >> "$trend_report"
    fi
    
    cat >> "$trend_report" << 'EOF'

## 🎯 改善推奨事項

### 短期改善項目
- テストカバレッジの向上
- 失敗テストの修正
- Lint問題の解決

### 中長期改善項目  
- セキュリティ問題の予防
- パフォーマンス最適化
- 継続的品質向上のプロセス確立

---
*🤖 このレポートは [Claude Code](https://claude.ai/code) によって自動生成されました。*
EOF

    log_info "品質トレンド分析完了: $trend_report"
}

# 改善効果測定
measure_improvement_effects() {
    log_info "改善効果測定を実行中..."
    
    local effects_report="reports/improvement_effects.json" 
    local execution_start_time=$(date '+%s')
    
    # 履歴データから改善効果を計算
    local history_files=$(find "$HISTORY_DIR" -name "quality_metrics_*.json" 2>/dev/null | sort)
    local file_count=$(echo "$history_files" | wc -w)
    
    local first_score=0
    local latest_score=0
    local average_score=0
    local improvement_rate=0
    
    if [ $file_count -ge 2 ]; then
        # 最初と最新のスコアを取得
        local first_file=$(echo "$history_files" | head -1)
        local latest_file=$(echo "$history_files" | tail -1)
        
        if command -v jq >/dev/null 2>&1; then
            first_score=$(jq -r '.quality_metrics.overall_score // 0' "$first_file")
            latest_score=$(jq -r '.quality_metrics.overall_score // 0' "$latest_file")
        else
            first_score=$(grep -o '"overall_score": *[0-9]*' "$first_file" 2>/dev/null | head -1 | grep -o '[0-9]*' || echo "0")
            latest_score=$(grep -o '"overall_score": *[0-9]*' "$latest_file" 2>/dev/null | head -1 | grep -o '[0-9]*' || echo "0")
        fi
        
        # 改善率計算（分母0回避）
        if [ $first_score -gt 0 ]; then
            improvement_rate=$(( (latest_score - first_score) * 100 / first_score ))
        fi
        
        # 平均スコア計算
        local total_score=0
        for file in $history_files; do
            local score
            if command -v jq >/dev/null 2>&1; then
                score=$(jq -r '.quality_metrics.overall_score // 0' "$file")
            else
                score=$(grep -o '"overall_score": *[0-9]*' "$file" 2>/dev/null | head -1 | grep -o '[0-9]*' || echo "0")
            fi
            total_score=$((total_score + score))
        done
        average_score=$((total_score / file_count))
    fi
    
    local execution_end_time=$(date '+%s')
    local execution_time=$((execution_end_time - execution_start_time))
    
    # 改善効果レポートをJSONで生成
    cat > "$effects_report" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "execution_time": "${execution_time}s",
    "measurement_period": {
        "total_measurements": $file_count,
        "first_measurement_score": $first_score,
        "latest_measurement_score": $latest_score,
        "average_score": $average_score
    },
    "improvement_metrics": {
        "absolute_improvement": $((latest_score - first_score)),
        "improvement_rate_percent": $improvement_rate,
        "trend_analysis": "$(if [ $improvement_rate -gt 0 ]; then echo "improving"; elif [ $improvement_rate -lt 0 ]; then echo "declining"; else echo "stable"; fi)"
    },
    "recommendations": {
        "continue_monitoring": true,
        "focus_areas": [
            "test_coverage",
            "security_issues", 
            "code_quality"
        ],
        "next_review_date": "$(if [ "$(uname)" = "Darwin" ]; then date -v+1w +%Y-%m-%d; else date -d '+1 week' +%Y-%m-%d; fi)"
    }
}
EOF
    
    log_info "改善効果測定完了: $effects_report"
}

# 過去結果との比較分析
compare_with_previous() {
    log_info "過去結果との比較分析を実行中..."
    
    local comparison_report="reports/progress_report.md"
    
    cat > "$comparison_report" << 'EOF'
# 品質向上進捗レポート

## 📈 継続改善の進捗状況

### 今回の品質指標
EOF
    
    # 現在の指標を表示
    if [ -f "reports/quality_summary.json" ]; then
        echo "- 総合品質スコア: $(grep -o '"overall_score": *[0-9]*' reports/quality_summary.json 2>/dev/null | head -1 | grep -o '[0-9]*' || echo '未測定')点" >> "$comparison_report"
    fi
    
    if [ -f "reports/test_results.json" ]; then
        echo "- テストカバレッジ: $(grep -o '"coverage_percent": *[0-9.]*' reports/test_results.json 2>/dev/null | head -1 | grep -o '[0-9.]*' || echo '0')%" >> "$comparison_report"
    fi
    
    cat >> "$comparison_report" << 'EOF'

### 継続改善のための推奨アクション

1. **定期実行の継続**: 週次での品質チェック実行を推奨
2. **問題の早期解決**: 新たに検出された問題への迅速な対応
3. **トレンド監視**: 品質指標の悪化傾向の早期発見

### 次回チェック推奨日時
次回の品質チェックは **1週間後** に実行することを推奨します。

---
*🤖 このレポートは継続改善支援機能によって自動生成されました。*
EOF
    
    log_info "比較分析レポート生成完了: $comparison_report"
}

# プログレス追跡のメイン関数
track_progress() {
    log_info "🔄 継続改善支援機能を開始します"
    
    local execution_start_time=$(date '+%s')
    
    # 品質指標履歴管理の実行
    track_quality_metrics
    
    # 過去結果との比較
    compare_with_previous
    
    local execution_end_time=$(date '+%s')
    local total_execution_time=$((execution_end_time - execution_start_time))
    
    # 実行結果サマリー
    local summary_file="reports/progress_tracking.json"
    cat > "$summary_file" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "execution_time": "${total_execution_time}s",
    "tracking_summary": {
        "history_entries": $(find "$HISTORY_DIR" -name "quality_metrics_*.json" 2>/dev/null | wc -l | tr -d ' '),
        "reports_generated": [
            "trend_analysis.md",
            "improvement_effects.json", 
            "progress_report.md"
        ],
        "tracking_status": "success"
    },
    "next_actions": {
        "recommended_frequency": "weekly",
        "focus_areas": ["test_coverage", "code_quality", "security"],
        "automation_status": "manual"
    }
}
EOF
    
    log_info "✅ 継続改善支援機能が完了しました（実行時間: ${total_execution_time}秒）"
    log_info "📊 生成されたレポート:"
    log_info "   - 品質指標履歴: $HISTORY_DIR/"
    log_info "   - トレンド分析: reports/trend_analysis.md"
    log_info "   - 改善効果測定: reports/improvement_effects.json"
    log_info "   - 進捗レポート: reports/progress_report.md"
}

# メイン実行部
main() {
    check_prerequisites
    ensure_directories
    
    case "${1:-}" in
        --help)
            echo "継続改善支援機能"
            echo "使用方法: $0 [オプション]"
            echo ""
            echo "オプション:"
            echo "  --help     このヘルプを表示"
            echo "  --track    品質指標追跡を実行"
            echo "  --trend    トレンド分析のみ実行"
            echo "  --compare  比較分析のみ実行"
            echo ""
            echo "例:"
            echo "  $0                    # 全機能実行"
            echo "  $0 --track           # 品質指標追跡のみ"
            echo "  $0 --trend           # トレンド分析のみ"
            ;;
        --track)
            track_quality_metrics
            ;;
        --trend)
            analyze_quality_trends
            ;;
        --compare)
            compare_with_previous
            ;;
        *)
            track_progress
            ;;
    esac
}

# スクリプトが直接実行された場合のみメイン関数を実行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi