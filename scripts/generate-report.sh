#!/bin/bash

# 統合レポート生成スクリプト: 全分析結果を統合しMarkdown形式レポート生成
# Task 12: 統合レポート生成機能
# Task 13: 詳細レポート機能を含む

set -e

# スクリプトディレクトリの取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 共通関数の読み込み
source "$SCRIPT_DIR/utils.sh"

# 設定
REPORTS_DIR="$PROJECT_ROOT/reports"
TMP_DIR="$PROJECT_ROOT/tmp"

# Task 12: 統合レポート生成機能
generate_integrated_report() {
    log_info "統合レポートを生成中..."
    
    local integrated_report="$REPORTS_DIR/integrated_report.md"
    local quality_summary_json="$REPORTS_DIR/quality_summary.json"
    local start_time=$(date +%s)
    
    # 既存分析結果の収集
    local test_coverage=0
    local lint_issues=0
    local security_issues=0
    local test_failures=0
    local benchmark_count=0
    local total_priority_score=0
    
    # テスト結果の読み取り
    if [ -f "$REPORTS_DIR/test_results.json" ]; then
        if command -v jq >/dev/null 2>&1; then
            test_coverage=$(jq -r '.coverage_summary.coverage_percent // 0' "$REPORTS_DIR/test_results.json" 2>/dev/null || echo "0")
            test_failures=$(jq -r '.test_summary.failed // 0' "$REPORTS_DIR/test_results.json" 2>/dev/null || echo "0")
        else
            test_coverage=$(grep -o '"coverage_percent":[0-9.]*' "$REPORTS_DIR/test_results.json" 2>/dev/null | grep -o '[0-9.]*' | head -1 || echo "0")
            test_failures=$(grep -o '"failed":[0-9]*' "$REPORTS_DIR/test_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
        fi
    fi
    
    # Lint結果の読み取り
    if [ -f "$REPORTS_DIR/lint_results.json" ]; then
        lint_issues=$(grep -o '"total_issues":[0-9]*' "$REPORTS_DIR/lint_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
    fi
    
    # セキュリティ結果の読み取り
    if [ -f "$REPORTS_DIR/security_results.json" ]; then
        security_issues=$(grep -o '"total_issues":[0-9]*' "$REPORTS_DIR/security_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
    fi
    
    # ベンチマーク結果の読み取り
    if [ -f "$REPORTS_DIR/benchmark_results.json" ]; then
        benchmark_count=$(grep -o '"total_benchmarks":[0-9]*' "$REPORTS_DIR/benchmark_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
    fi
    
    # 優先度スコアの読み取り
    if [ -f "$REPORTS_DIR/priority_results.json" ]; then
        total_priority_score=$(grep -o '"total_priority_score":[0-9]*' "$REPORTS_DIR/priority_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
    fi
    
    # 数値の正規化
    [ -z "$test_coverage" ] && test_coverage=0
    [ -z "$lint_issues" ] && lint_issues=0
    [ -z "$security_issues" ] && security_issues=0
    [ -z "$test_failures" ] && test_failures=0
    [ -z "$benchmark_count" ] && benchmark_count=0
    [ -z "$total_priority_score" ] && total_priority_score=0
    
    # 総合品質スコア算出
    local quality_score=$(calculate_quality_score "$test_coverage" "$lint_issues" "$security_issues" "$test_failures")
    
    local end_time=$(date +%s)
    local exec_time=$(( end_time - start_time ))
    
    # Markdown統合レポート生成
    cat > "$integrated_report" << EOF
# gcpclosecheck 品質分析統合レポート

**生成日時**: $(date)  
**分析実行時間**: ${exec_time}秒

## 📊 総合品質スコア

**スコア: $quality_score / 100**

$(generate_quality_badge "$quality_score")

---

## 📈 品質指標サマリー

### テスト品質
- **カバレッジ率**: ${test_coverage}%
- **テスト失敗数**: ${test_failures}件
- **ステータス**: $([ "$test_failures" -eq 0 ] && echo "✅ 正常" || echo "❌ 要修正")

### コード品質  
- **Lint警告数**: ${lint_issues}件
- **セキュリティ問題**: ${security_issues}件
- **ステータス**: $([ "$lint_issues" -eq 0 ] && [ "$security_issues" -eq 0 ] && echo "✅ 正常" || echo "⚠️ 要改善")

### パフォーマンス
- **ベンチマーク実行数**: ${benchmark_count}件
- **ステータス**: $([ "$benchmark_count" -gt 0 ] && echo "✅ 測定済み" || echo "⚠️ 未測定")

### 問題優先度
- **総合優先度スコア**: ${total_priority_score}
- **対応必要性**: $([ "$total_priority_score" -gt 50 ] && echo "🔴 高" || [ "$total_priority_score" -gt 10 ] && echo "🟡 中" || echo "🟢 低")

---

## 📋 詳細分析結果

### 🧪 テスト実行結果
$(if [ -f "$REPORTS_DIR/test_summary.txt" ]; then
    echo "\`\`\`"
    head -20 "$REPORTS_DIR/test_summary.txt"
    echo "\`\`\`"
else
    echo "テスト結果ファイルが見つかりません。"
fi)

### 🔍 静的解析結果
$(if [ -f "$REPORTS_DIR/lint_summary.txt" ]; then
    echo "\`\`\`"
    head -15 "$REPORTS_DIR/lint_summary.txt" 
    echo "\`\`\`"
else
    echo "静的解析結果ファイルが見つかりません。"
fi)

### 🔐 セキュリティスキャン結果
$(if [ -f "$REPORTS_DIR/security_summary.txt" ]; then
    echo "\`\`\`"
    head -15 "$REPORTS_DIR/security_summary.txt"
    echo "\`\`\`"
else
    echo "セキュリティスキャン結果ファイルが見つかりません。"
fi)

### ⚡ パフォーマンス測定結果
$(if [ -f "$REPORTS_DIR/benchmark_summary.txt" ]; then
    echo "\`\`\`"
    head -15 "$REPORTS_DIR/benchmark_summary.txt"
    echo "\`\`\`"
else
    echo "ベンチマーク結果ファイルが見つかりません。"
fi)

---

## 🎯 改善提案

### 最優先対応項目
$(if [ "$security_issues" -gt 0 ]; then
    echo "🔴 **セキュリティ問題の修正** ($security_issues 件)"
fi
if [ "$test_failures" -gt 0 ]; then
    echo "🔴 **失敗テストの修正** ($test_failures 件)"  
fi
if [ "$lint_issues" -gt 5 ]; then
    echo "🟡 **コード品質改善** ($lint_issues 件の警告)"
fi
if [ "${test_coverage%.*}" -lt 80 ] 2>/dev/null; then
    echo "🟡 **テストカバレッジ向上** (現在 ${test_coverage}%)"
fi
if [ "$total_priority_score" -eq 0 ]; then
    echo "🟢 **現在、重要な問題は検出されていません**"
fi)

### 推奨アクション
1. **セキュリティ**: 検出された脆弱性の即座な修正
2. **テスト**: 失敗テストの修正とカバレッジ向上
3. **コード品質**: golangci-lint警告の段階的解決
4. **継続改善**: 定期的な品質測定の実施

---

## 📂 関連レポート

- [テスト詳細結果](./test_summary.txt)
- [カバレッジレポート](./coverage.html) 
- [静的解析詳細](./lint_summary.txt)
- [セキュリティ詳細](./security_summary.txt)
- [ベンチマーク詳細](./benchmark_summary.txt)
- [優先度分析](./priority_summary.txt)

---

*🤖 このレポートは [Claude Code](https://claude.ai/code) によって自動生成されました。*

EOF
    
    # JSON品質サマリー生成
    cat > "$quality_summary_json" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "execution_time": "${exec_time}s",
    "overall_quality_score": $quality_score,
    "metrics": {
        "test_coverage": $test_coverage,
        "test_failures": $test_failures,
        "lint_issues": $lint_issues,
        "security_issues": $security_issues,
        "benchmark_count": $benchmark_count,
        "priority_score": $total_priority_score
    },
    "status": {
        "tests": "$([ "$test_failures" -eq 0 ] && echo "pass" || echo "fail")",
        "quality": "$([ "$lint_issues" -eq 0 ] && [ "$security_issues" -eq 0 ] && echo "good" || echo "needs_improvement")",
        "performance": "$([ "$benchmark_count" -gt 0 ] && echo "measured" || echo "not_measured")"
    },
    "recommendations": []
}
EOF
    
    log_success "統合レポートMarkdown: $integrated_report"
    log_success "品質サマリーJSON: $quality_summary_json"
    
    return 0
}

# Task 13: 詳細レポート機能
generate_detailed_report() {
    log_info "詳細レポートを生成中..."
    
    local detailed_report="$REPORTS_DIR/detailed_report.md"
    local executive_summary="$REPORTS_DIR/executive_summary.md"
    local start_time=$(date +%s)
    
    # 既存分析結果の詳細収集
    local test_coverage=0
    local lint_issues=0
    local security_issues=0
    local test_failures=0
    local total_priority_score=0
    
    # 分析結果の詳細読み込み（統合レポートと同様）
    if [ -f "$REPORTS_DIR/test_results.json" ]; then
        if command -v jq >/dev/null 2>&1; then
            test_coverage=$(jq -r '.coverage_summary.coverage_percent // 0' "$REPORTS_DIR/test_results.json" 2>/dev/null || echo "0")
            test_failures=$(jq -r '.test_summary.failed // 0' "$REPORTS_DIR/test_results.json" 2>/dev/null || echo "0")
        else
            test_coverage=$(grep -o '"coverage_percent":[0-9.]*' "$REPORTS_DIR/test_results.json" 2>/dev/null | grep -o '[0-9.]*' | head -1 || echo "0")
            test_failures=$(grep -o '"failed":[0-9]*' "$REPORTS_DIR/test_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
        fi
    fi
    
    if [ -f "$REPORTS_DIR/lint_results.json" ]; then
        lint_issues=$(grep -o '"total_issues":[0-9]*' "$REPORTS_DIR/lint_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
    fi
    
    if [ -f "$REPORTS_DIR/security_results.json" ]; then
        security_issues=$(grep -o '"total_issues":[0-9]*' "$REPORTS_DIR/security_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
    fi
    
    if [ -f "$REPORTS_DIR/priority_results.json" ]; then
        total_priority_score=$(grep -o '"total_priority_score":[0-9]*' "$REPORTS_DIR/priority_results.json" 2>/dev/null | grep -o '[0-9]*' || echo "0")
    fi
    
    # 数値の正規化
    [ -z "$test_coverage" ] && test_coverage=0
    [ -z "$lint_issues" ] && lint_issues=0
    [ -z "$security_issues" ] && security_issues=0
    [ -z "$test_failures" ] && test_failures=0
    [ -z "$total_priority_score" ] && total_priority_score=0
    
    local end_time=$(date +%s)
    local exec_time=$(( end_time - start_time ))
    
    # 技術者向け詳細レポート生成
    cat > "$detailed_report" << EOF
# gcpclosecheck 詳細分析レポート（技術者向け）

**生成日時**: $(date)  
**分析実行時間**: ${exec_time}秒

## 🔬 詳細分析結果

### 1. テスト実行詳細分析

#### 📊 カバレッジ分析
- **現在のカバレッジ**: ${test_coverage}%
- **推奨カバレッジ**: 80%以上
- **ギャップ**: $((80 - ${test_coverage%.*}))ポイント不足

#### 🧪 テスト実行状況
$(if [ -f "$REPORTS_DIR/test_summary.txt" ]; then
    echo "\`\`\`"
    cat "$REPORTS_DIR/test_summary.txt"
    echo "\`\`\`"
else
    echo "テスト詳細結果が利用できません。"
fi)

#### 📈 カバレッジ不足箇所
$(if [ -f "$REPORTS_DIR/low_coverage_functions.txt" ]; then
    echo "\`\`\`"
    cat "$REPORTS_DIR/low_coverage_functions.txt"
    echo "\`\`\`"
else
    echo "カバレッジ詳細が利用できません。"
fi)

### 2. 静的解析詳細

#### ⚠️ Lint問題の分類と対策
- **総問題数**: ${lint_issues}件
- **影響範囲**: $([ "$lint_issues" -gt 10 ] && echo "高" || [ "$lint_issues" -gt 5 ] && echo "中" || echo "低")
- **推奨対策**: 段階的な修正と継続的改善

#### 🔍 Lint詳細結果
$(if [ -f "$REPORTS_DIR/lint_summary.txt" ]; then
    echo "\`\`\`"
    cat "$REPORTS_DIR/lint_summary.txt"
    echo "\`\`\`"
else
    echo "Lint詳細結果が利用できません。"
fi)

### 3. セキュリティ詳細分析

#### 🛡️ セキュリティ問題の詳細
- **検出問題数**: ${security_issues}件
- **脅威レベル**: $([ "$security_issues" -gt 0 ] && echo "要対応" || echo "問題なし")
- **対応優先度**: $([ "$security_issues" -gt 0 ] && echo "最高" || echo "通常")

#### 🔐 セキュリティスキャン詳細
$(if [ -f "$REPORTS_DIR/security_summary.txt" ]; then
    echo "\`\`\`"
    cat "$REPORTS_DIR/security_summary.txt"
    echo "\`\`\`"
else
    echo "セキュリティ詳細結果が利用できません。"
fi)

### 4. パフォーマンス詳細分析

#### ⚡ ベンチマーク結果詳細
$(if [ -f "$REPORTS_DIR/benchmark_summary.txt" ]; then
    echo "\`\`\`"
    cat "$REPORTS_DIR/benchmark_summary.txt"
    echo "\`\`\`"
else
    echo "パフォーマンス詳細結果が利用できません。"
fi)

#### 📊 プロファイリング結果
$(if [ -f "$REPORTS_DIR/profile_summary.txt" ]; then
    echo "\`\`\`"
    head -20 "$REPORTS_DIR/profile_summary.txt"
    echo "\`\`\`"
else
    echo "プロファイリング結果が利用できません。"
fi)

### 5. 問題優先度詳細分析

#### 📋 優先度マトリックス
$(if [ -f "$REPORTS_DIR/priority_summary.txt" ]; then
    echo "\`\`\`"
    cat "$REPORTS_DIR/priority_summary.txt"
    echo "\`\`\`"
else
    echo "優先度分析結果が利用できません。"
fi)

## 🎯 技術的推奨事項

### 即座に対応すべき問題
$([ "$security_issues" -gt 0 ] && echo "- 🔴 **セキュリティ問題**: $security_issues 件の脆弱性を即座に修正")
$([ "$test_failures" -gt 0 ] && echo "- 🔴 **テスト失敗**: $test_failures 件の失敗テストを修正")

### 短期改善項目（1-2週間）
$([ "$lint_issues" -gt 10 ] && echo "- 🟡 **コード品質**: $lint_issues 件のlint警告を段階的に解決")
$([ "${test_coverage%.*}" -lt 80 ] && echo "- 🟡 **テストカバレッジ**: 現在 $test_coverage% を 80% 以上に向上")

### 中長期改善項目（1-2ヶ月）
- 🟢 **継続的品質向上**: 品質ゲートの自動化
- 🟢 **パフォーマンス最適化**: ベンチマーク継続実施
- 🟢 **技術的負債削減**: リファクタリング計画の策定

## 🔧 実装ガイド

### テストカバレッジ向上
1. 未カバー関数の特定と優先度付け
2. 単体テストの段階的追加
3. 統合テストの充実
4. E2Eテストの導入検討

### 静的解析問題解決
1. golangci-lint設定の最適化
2. 段階的な警告解決
3. CI/CDパイプラインとの統合
4. 開発チーム内のコーディング規約統一

### セキュリティ強化
1. gosec/govulncheckの定期実行
2. 依存関係の脆弱性チェック
3. セキュリティレビュープロセス確立
4. セキュリティ教育の実施

---

*🤖 この技術的詳細レポートは [Claude Code](https://claude.ai/code) によって自動生成されました。*

EOF

    # 経営層向けサマリー生成
    local quality_score=$(calculate_quality_score "$test_coverage" "$lint_issues" "$security_issues" "$test_failures")
    
    cat > "$executive_summary" << EOF
# 品質状況 経営層向けサマリー

**レポート作成日**: $(date +"%Y年%m月%d日")

## 📊 プロジェクト品質概況

### 総合評価
**品質スコア: $quality_score / 100**

$(generate_quality_badge "$quality_score")

### 主要指標サマリー
| 項目 | 現在値 | 目標値 | 状況 |
|------|--------|--------|------|
| テストカバレッジ | ${test_coverage}% | 80% | $([ "${test_coverage%.*}" -lt 80 ] && echo "⚠️ 要改善" || echo "✅ 良好") |
| テスト失敗数 | ${test_failures}件 | 0件 | $([ "$test_failures" -gt 0 ] && echo "❌ 要対応" || echo "✅ 良好") |
| コード品質問題 | ${lint_issues}件 | < 5件 | $([ "$lint_issues" -gt 5 ] && echo "⚠️ 要改善" || echo "✅ 良好") |
| セキュリティ問題 | ${security_issues}件 | 0件 | $([ "$security_issues" -gt 0 ] && echo "🔴 緊急対応" || echo "✅ 良好") |

## 🎯 重要な意思決定事項

### 緊急対応が必要な項目
$([ "$security_issues" -gt 0 ] && echo "🔴 **セキュリティリスク**: $security_issues 件の脆弱性が検出されています。即座な対応が必要です。")
$([ "$test_failures" -gt 0 ] && echo "🔴 **品質リスク**: $test_failures 件のテストが失敗しており、リリース前の修正が必要です。")

### 投資検討が推奨される領域
$([ "${test_coverage%.*}" -lt 70 ] && echo "- 🟡 **テスト自動化**: カバレッジ向上のためのテスト投資")
$([ "$lint_issues" -gt 20 ] && echo "- 🟡 **コード品質向上**: リファクタリングと品質改善投資")

## 💰 ビジネスインパクト

### リスク評価
- **高リスク**: $([ "$security_issues" -gt 0 ] || [ "$test_failures" -gt 10 ] && echo "セキュリティ・品質問題により、プロダクションリスクが存在" || echo "なし")
- **中リスク**: $([ "${test_coverage%.*}" -lt 60 ] && echo "テストカバレッジ不足により、将来的な保守性リスク" || echo "なし")
- **低リスク**: $([ "$lint_issues" -gt 0 ] && [ "$lint_issues" -lt 10 ] && echo "軽微なコード品質問題" || echo "なし")

### 予想される改善効果
- **品質向上**: バグ発生率 20-30% 削減
- **開発効率**: メンテナンス工数 15-25% 削減
- **セキュリティ**: 脆弱性リスク最小化

## 📅 推奨タイムライン

### 即座（今週中）
$([ "$security_issues" -gt 0 ] && echo "- セキュリティ問題の修正")
$([ "$test_failures" -gt 5 ] && echo "- 重要なテスト失敗の修正")

### 短期（1ヶ月以内）
- テストカバレッジ80%達成
- 主要なコード品質問題の解決

### 中期（3ヶ月以内）
- 継続的品質改善プロセス確立
- 開発チームの品質意識向上

## 💡 経営層への提言

$(if [ "$quality_score" -lt 70 ]; then
    echo "現在の品質状況は改善が必要です。短期集中的な投資により、中長期的な開発効率とセキュリティの大幅改善が期待できます。"
elif [ "$quality_score" -lt 85 ]; then
    echo "品質は概ね良好ですが、さらなる向上の余地があります。継続的改善により、より安定したプロダクト開発が可能になります。"
else
    echo "優良な品質レベルを維持しています。現在の品質水準を保持し、継続的な監視体制を整備することを推奨します。"
fi)

---

*このサマリーは技術品質データに基づいて自動生成されています。詳細な技術情報については詳細レポートをご参照ください。*

EOF
    
    log_success "技術者向け詳細レポート: $detailed_report"
    log_success "経営層向けサマリー: $executive_summary"
    
    return 0
}

# 品質スコア算出
calculate_quality_score() {
    local coverage=$1
    local lint_issues=$2
    local security_issues=$3
    local test_failures=$4
    
    local score=100
    
    # カバレッジによる減点（整数部分で比較）
    local coverage_int="${coverage%.*}"
    [ -z "$coverage_int" ] && coverage_int=0
    
    if [ "$coverage_int" -lt 70 ]; then
        score=$((score - 30))
    elif [ "$coverage_int" -lt 80 ]; then
        score=$((score - 20))
    elif [ "$coverage_int" -lt 90 ]; then
        score=$((score - 10))
    fi
    
    # テスト失敗による減点
    score=$((score - test_failures * 10))
    
    # セキュリティ問題による減点
    score=$((score - security_issues * 15))
    
    # lint問題による減点
    if [ "$lint_issues" -gt 20 ]; then
        score=$((score - 20))
    elif [ "$lint_issues" -gt 10 ]; then
        score=$((score - 10))
    elif [ "$lint_issues" -gt 0 ]; then
        score=$((score - 5))
    fi
    
    # 最小値0の保証
    [ "$score" -lt 0 ] && score=0
    
    echo "$score"
}

# 品質バッジ生成
generate_quality_badge() {
    local score=$1
    
    if [ "$score" -ge 90 ]; then
        echo "🟢 **優秀** - 高品質なコードベース"
    elif [ "$score" -ge 80 ]; then
        echo "🟡 **良好** - 一部改善の余地あり"  
    elif [ "$score" -ge 70 ]; then
        echo "🟠 **要改善** - 重要な問題が存在"
    else
        echo "🔴 **要対応** - 緊急な修正が必要"
    fi
}

# メイン実行
main() {
    log_info "📊 統合レポート生成開始"
    
    # 前提条件チェック
    check_prerequisites
    
    # 必要なディレクトリ作成
    ensure_directories "$REPORTS_DIR" "$TMP_DIR"
    
    # プロジェクトルートに移動
    cd "$PROJECT_ROOT"
    
    # Task 12: 統合レポート生成
    if generate_integrated_report; then
        log_success "統合レポート生成完了"
    else
        log_warning "統合レポート生成で問題が発生しました"
    fi
    
    # Task 13: 詳細レポート生成
    if generate_detailed_report; then
        log_success "詳細レポート生成完了"
    else
        log_warning "詳細レポート生成で問題が発生しました"
    fi
    
    log_success "全レポート生成処理完了"
}

main "$@"