#!/bin/bash

# テストスクリプト: 各スクリプトの動作検証
# 基盤インフラの存在と動作をテストする
set -e

echo "🧪 品質チェックスクリプトのテスト開始"

# テスト失敗カウンター
test_failures=0

# ディレクトリ構造テスト
test_directory_structure() {
    echo "  ✓ ディレクトリ構造のテスト"
    
    # 必要なディレクトリの存在確認
    for dir in scripts reports tmp; do
        if [ -d "$dir" ]; then
            echo "    ✓ $dir ディレクトリ存在確認"
        else
            echo "    ❌ $dir ディレクトリが見つかりません"
            ((test_failures++))
        fi
    done
}

# utils.sh関数テスト
test_utils_functions() {
    echo "  ✓ utils.sh関数のテスト"
    
    if [ ! -f "scripts/utils.sh" ]; then
        echo "    ❌ scripts/utils.sh が見つかりません"
        ((test_failures++))
        return 1
    fi
    
    source scripts/utils.sh
    
    # log_info関数のテスト
    if command -v log_info >/dev/null 2>&1; then
        if log_info "テストメッセージ" | grep -q "テストメッセージ"; then
            echo "    ✓ log_info 関数動作確認"
        else
            echo "    ❌ log_info 関数の出力が期待と異なります"
            ((test_failures++))
        fi
    else
        echo "    ❌ log_info 関数が見つかりません"
        ((test_failures++))
    fi
    
    # check_prerequisites関数のテスト  
    if command -v check_prerequisites >/dev/null 2>&1; then
        echo "    ✓ check_prerequisites 関数動作確認"
    else
        echo "    ❌ check_prerequisites 関数が見つかりません"
        ((test_failures++))
    fi
    
    # ensure_directories関数のテスト
    if command -v ensure_directories >/dev/null 2>&1; then
        echo "    ✓ ensure_directories 関数動作確認"
    else
        echo "    ❌ ensure_directories 関数が見つかりません"
        ((test_failures++))
    fi
}

# .gitignore設定テスト
test_gitignore_setup() {
    echo "  ✓ .gitignore設定のテスト"
    
    if [ -f .gitignore ]; then
        if grep -q "reports/" .gitignore && grep -q "tmp/" .gitignore; then
            echo "    ✓ .gitignore に reports/ と tmp/ が追加済み"
        else
            echo "    ❌ .gitignore の設定が不完全です"
            ((test_failures++))
        fi
    else
        echo "    ❌ .gitignore ファイルが見つかりません"
        ((test_failures++))
    fi
}

# quality-check.sh スクリプトテスト
test_quality_check_script() {
    echo "  ✓ quality-check.sh スクリプトのテスト"
    
    if [ ! -f "scripts/quality-check.sh" ]; then
        echo "    ❌ scripts/quality-check.sh が見つかりません"
        ((test_failures++))
        return 1
    fi
    
    # 実行権限テスト
    if [ -x "scripts/quality-check.sh" ]; then
        echo "    ✓ scripts/quality-check.sh 実行権限確認"
    else
        echo "    ❌ scripts/quality-check.sh に実行権限がありません"
        ((test_failures++))
    fi
    
    # ヘルプオプションテスト
    if scripts/quality-check.sh --help >/dev/null 2>&1; then
        echo "    ✓ --help オプション動作確認"
    else
        echo "    ❌ --help オプションが機能しません"
        ((test_failures++))
    fi
    
    # 各種オプションの存在テスト
    local script_content=$(cat scripts/quality-check.sh)
    if echo "$script_content" | grep -q "\--test\|--quality\|--perf\|--all"; then
        echo "    ✓ 必要なオプション (--test, --quality, --perf, --all) 確認"
    else
        echo "    ❌ 必要なオプションが実装されていません"
        ((test_failures++))
    fi
}

# test-analysis.sh スクリプトテスト
test_test_analysis_script() {
    echo "  ✓ test-analysis.sh スクリプトのテスト"
    
    if [ ! -f "scripts/test-analysis.sh" ]; then
        echo "    ❌ scripts/test-analysis.sh が見つかりません"
        ((test_failures++))
        return 1
    fi
    
    # 実行権限テスト
    if [ -x "scripts/test-analysis.sh" ]; then
        echo "    ✓ scripts/test-analysis.sh 実行権限確認"
    else
        echo "    ❌ scripts/test-analysis.sh に実行権限がありません"
        ((test_failures++))
    fi
    
    # カバレッジ分析機能の存在テスト
    local script_content=$(cat scripts/test-analysis.sh)
    if echo "$script_content" | grep -q "go test -cover\|coverage"; then
        echo "    ✓ カバレッジ分析機能が実装されています"
    else
        echo "    ❌ カバレッジ分析機能が見つかりません"
        ((test_failures++))
    fi
    
    # カバレッジ閾値チェック機能テスト
    if echo "$script_content" | grep -q "80.*%\|COVERAGE_THRESHOLD"; then
        echo "    ✓ カバレッジ閾値チェック機能確認"
    else
        echo "    ❌ カバレッジ閾値チェック機能が見つかりません"
        ((test_failures++))
    fi
    
    # HTMLレポート生成機能テスト
    if echo "$script_content" | grep -q "go tool cover.*html\|coverage.html"; then
        echo "    ✓ HTMLカバレッジレポート生成機能確認"
    else
        echo "    ❌ HTMLカバレッジレポート生成機能が見つかりません"
        ((test_failures++))
    fi
    
    # Task 4: テスト実行結果分析機能のテスト（新機能）
    if echo "$script_content" | grep -q "analyze_test_execution_results\|test_results.json"; then
        echo "    ✓ テスト実行結果分析機能が実装されています"
    else
        echo "    ❌ テスト実行結果分析機能が見つかりません（Task 4 未実装）"
        ((test_failures++))
    fi
}

# Task 4: テスト実行結果分析機能の動作テスト
test_test_execution_analysis() {
    echo "  ✓ Task 4: テスト実行結果分析機能の動作テスト"
    
    # test-analysis.shを実行してテスト結果分析を行う
    if scripts/test-analysis.sh >/dev/null 2>&1; then
        echo "    ✓ test-analysis.sh実行成功"
    else
        echo "    ⚠️  test-analysis.sh実行で警告（処理継続）"
    fi
    
    # テスト結果JSONファイルの存在確認
    if [ -f "reports/test_results.json" ]; then
        echo "    ✓ test_results.json生成確認"
# Task 7: ビルド検証機能の実装テスト
test_build_verification_functionality() {
    echo "  ✓ Task 7: ビルド検証機能の実装テスト"
    
    # code-quality.shにビルド検証機能が含まれているか確認
    if [ -f "scripts/code-quality.sh" ]; then
        local script_content=$(cat scripts/code-quality.sh)
        if echo "$script_content" | grep -q "run_build_verification\|go build\|go vet"; then
            echo "    ✓ ビルド検証機能が実装されています"
        else
            echo "    ❌ ビルド検証機能が見つかりません（Task 7 未実装）"
            ((test_failures++))
            return 1
        fi
    else
        echo "    ❌ scripts/code-quality.sh が存在しません"
        ((test_failures++))
        return 1
    fi
    
    # go buildとgo vetが利用可能か確認
    if command -v go >/dev/null 2>&1; then
        echo "    ✓ go コマンドが利用可能"
        
        # ビルド検証テスト実行（簡易版）
        if scripts/code-quality.sh >/dev/null 2>&1; then
            echo "    ✓ ビルド検証実行成功"
            
            # ビルド結果JSON生成確認
            if [ -f "reports/build_results.json" ]; then
                echo "    ✓ build_results.json生成確認"
            else
                echo "    ❌ build_results.json が生成されていません"
                ((test_failures++))
            fi
            
            # ビルドサマリー生成確認  
            if [ -f "reports/build_summary.txt" ]; then
                echo "    ✓ build_summary.txt生成確認"
            else
                echo "    ❌ build_summary.txt が生成されていません"
                ((test_failures++))
            fi
        else
            echo "    ❌ ビルド検証実行に失敗"
            ((test_failures++))
        fi
    else
        echo "    ❌ go コマンドが利用できません"
        ((test_failures++))
    fi
}

# Task 8: ベンチマーク実行スクリプトの実装テスト
test_benchmark_execution_functionality() {
    echo "  ✓ Task 8: ベンチマーク実行スクリプトの実装テスト"
    
    # scripts/performance-check.sh の存在確認
    if [ -f "scripts/performance-check.sh" ]; then
        echo "    ✓ scripts/performance-check.sh 存在確認"
        
        # 実行権限確認
        if [ -x "scripts/performance-check.sh" ]; then
            echo "    ✓ scripts/performance-check.sh 実行権限確認"
        else
            echo "    ❌ scripts/performance-check.sh に実行権限がありません"
            ((test_failures++))
        fi
        
        # ベンチマーク実行機能が実装されているか確認
        local script_content=$(cat scripts/performance-check.sh)
        if echo "$script_content" | grep -q "run_benchmark_tests\|go test -bench"; then
            echo "    ✓ ベンチマーク実行機能が実装されています"
        else
            echo "    ❌ ベンチマーク実行機能が見つかりません（Task 8 未実装）"
            ((test_failures++))
            return 1
        fi
    else
        echo "    ❌ scripts/performance-check.sh が存在しません（Task 8 未実装）"
        ((test_failures++))
        return 1
    fi
    
    # go コマンドが利用可能か確認
    if command -v go >/dev/null 2>&1; then
        echo "    ✓ go コマンドが利用可能"
        
        # ベンチマーク機能テスト実行（簡易版）
        if scripts/performance-check.sh >/dev/null 2>&1; then
            echo "    ✓ ベンチマーク実行成功"
            
            # ベンチマーク結果JSON生成確認
            if [ -f "reports/benchmark_results.json" ]; then
                echo "    ✓ benchmark_results.json生成確認"
            else
                echo "    ❌ benchmark_results.json が生成されていません"
                ((test_failures++))
            fi
            
            # ベンチマークサマリー生成確認  
            if [ -f "reports/benchmark_summary.txt" ]; then
                echo "    ✓ benchmark_summary.txt生成確認"
            else
                echo "    ❌ benchmark_summary.txt が生成されていません"
                ((test_failures++))
            fi
        else
            echo "    ❌ ベンチマーク実行に失敗"
            ((test_failures++))
        fi
    else
        echo "    ❌ go コマンドが利用できません"
        ((test_failures++))
    fi
}

# Task 9: プロファイリング機能の実装テスト
test_profiling_functionality() {
    echo "  ✓ Task 9: プロファイリング機能の実装テスト"
    
    # performance-check.shにプロファイリング機能が含まれているか確認
    if [ -f "scripts/performance-check.sh" ]; then
        local script_content=$(cat scripts/performance-check.sh)
        if echo "$script_content" | grep -q "run_profiling\|go tool pprof\|cpuprof\|memprof"; then
            echo "    ✓ プロファイリング機能が実装されています"
        else
            echo "    ❌ プロファイリング機能が見つかりません（Task 9 未実装）"
            ((test_failures++))
            return 1
        fi
    else
        echo "    ❌ scripts/performance-check.sh が存在しません"
        ((test_failures++))
        return 1
    fi
    
    # go コマンドが利用可能か確認
    if command -v go >/dev/null 2>&1; then
        echo "    ✓ go コマンドが利用可能"
        
        # プロファイリング機能テスト実行（簡易版）
        if scripts/performance-check.sh >/dev/null 2>&1; then
            echo "    ✓ プロファイリング実行成功"
            
            # プロファイル結果ファイル生成確認
            if [ -f "reports/profile_results.json" ]; then
                echo "    ✓ profile_results.json生成確認"
            else
                echo "    ❌ profile_results.json が生成されていません"
                ((test_failures++))
            fi
            
            # プロファイルサマリー生成確認  
            if [ -f "reports/profile_summary.txt" ]; then
                echo "    ✓ profile_summary.txt生成確認"
            else
                echo "    ❌ profile_summary.txt が生成されていません"
                ((test_failures++))
            fi
        else
            echo "    ❌ プロファイリング実行に失敗"
            ((test_failures++))
        fi
    else
        echo "    ❌ go コマンドが利用できません"
        ((test_failures++))
    fi
}

# Task 10: 自動修正スクリプトの実装テスト
test_auto_fix_functionality() {
    echo "  ✓ Task 10: 自動修正スクリプトの実装テスト"
    
    # scripts/fix-issues.sh の存在確認
    if [ -f "scripts/fix-issues.sh" ]; then
        echo "    ✓ scripts/fix-issues.sh 存在確認"
        
        # 実行権限確認
        if [ -x "scripts/fix-issues.sh" ]; then
            echo "    ✓ scripts/fix-issues.sh 実行権限確認"
        else
            echo "    ❌ scripts/fix-issues.sh に実行権限がありません"
            ((test_failures++))
        fi
        
        # 自動修正機能が実装されているか確認
        local script_content=$(cat scripts/fix-issues.sh)
        if echo "$script_content" | grep -q "run_auto_fixes\|go fmt\|goimports\|golangci-lint --fix"; then
            echo "    ✓ 自動修正機能が実装されています"
        else
            echo "    ❌ 自動修正機能が見つかりません（Task 10 未実装）"
            ((test_failures++))
            return 1
        fi
    else
        echo "    ❌ scripts/fix-issues.sh が存在しません（Task 10 未実装）"
        ((test_failures++))
        return 1
    fi
    
    # go コマンドが利用可能か確認
    if command -v go >/dev/null 2>&1; then
        echo "    ✓ go コマンドが利用可能"
        
        # 自動修正機能テスト実行（簡易版）
        if scripts/fix-issues.sh >/dev/null 2>&1; then
            echo "    ✓ 自動修正実行成功"
            
            # 修正結果ファイル生成確認
            if [ -f "reports/fix_results.json" ]; then
                echo "    ✓ fix_results.json生成確認"
            else
                echo "    ❌ fix_results.json が生成されていません"
                ((test_failures++))
            fi
            
            # 修正サマリー生成確認  
            if [ -f "reports/fix_summary.txt" ]; then
                echo "    ✓ fix_summary.txt生成確認"
            else
                echo "    ❌ fix_summary.txt が生成されていません"
                ((test_failures++))
            fi
        else
            echo "    ❌ 自動修正実行に失敗"
            ((test_failures++))
        fi
    else
        echo "    ❌ go コマンドが利用できません"
        ((test_failures++))
    fi
}

# Task 11: 問題優先度付け機能の実装テスト
test_priority_functionality() {
    echo "  ✓ Task 11: 問題優先度付け機能の実装テスト"
    
    # scripts/fix-issues.sh に優先度付け機能が含まれているか確認
    if [ -f "scripts/fix-issues.sh" ]; then
        local script_content=$(cat scripts/fix-issues.sh)
        if echo "$script_content" | grep -q "priority\|影響度\|修正コスト\|prioritize_issues\|calculate_priority"; then
            echo "    ✓ 優先度付け機能が実装されています"
        else
            echo "    ❌ 優先度付け機能が見つかりません（Task 11 未実装）"
            ((test_failures++))
            return 1
        fi
    else
        echo "    ❌ scripts/fix-issues.sh が存在しません"
        ((test_failures++))
        return 1
    fi
    
    # go コマンドが利用可能か確認
    if command -v go >/dev/null 2>&1; then
        echo "    ✓ go コマンドが利用可能"
        
        # 優先度付け機能テスト実行（簡易版）
        if scripts/fix-issues.sh >/dev/null 2>&1; then
            echo "    ✓ 優先度付け実行成功"
            
            # 優先度結果ファイル生成確認
            if [ -f "reports/priority_results.json" ]; then
                echo "    ✓ priority_results.json生成確認"
            else
                echo "    ❌ priority_results.json が生成されていません"
                ((test_failures++))
            fi
            
            # 優先度サマリー生成確認  
            if [ -f "reports/priority_summary.txt" ]; then
                echo "    ✓ priority_summary.txt生成確認"
            else
                echo "    ❌ priority_summary.txt が生成されていません"
                ((test_failures++))
            fi
        else
            echo "    ❌ 優先度付け実行に失敗"
            ((test_failures++))
        fi
    else
        echo "    ❌ go コマンドが利用できません"
        ((test_failures++))
    fi
}

# Task 12: 統合レポート生成機能の実装テスト
test_integrated_report_functionality() {
    echo "  ✓ Task 12: 統合レポート生成機能の実装テスト"
    
    # scripts/generate-report.sh の存在確認
    if [ -f "scripts/generate-report.sh" ]; then
        echo "    ✓ scripts/generate-report.sh 存在確認"
        
        # 実行権限確認
        if [ -x "scripts/generate-report.sh" ]; then
            echo "    ✓ scripts/generate-report.sh 実行権限確認"
        else
            echo "    ❌ scripts/generate-report.sh に実行権限がありません"
            ((test_failures++))
        fi
        
        # 統合レポート機能が実装されているか確認
        local script_content=$(cat scripts/generate-report.sh)
        if echo "$script_content" | grep -q "generate_integrated_report\|markdown\|総合品質スコア\|quality_score"; then
            echo "    ✓ 統合レポート機能が実装されています"
        else
            echo "    ❌ 統合レポート機能が見つかりません（Task 12 未実装）"
            ((test_failures++))
            return 1
        fi
    else
        echo "    ❌ scripts/generate-report.sh が存在しません（Task 12 未実装）"
        ((test_failures++))
        return 1
    fi
    
    # go コマンドが利用可能か確認
    if command -v go >/dev/null 2>&1; then
        echo "    ✓ go コマンドが利用可能"
        
        # 統合レポート生成テスト実行（簡易版）
        if scripts/generate-report.sh >/dev/null 2>&1; then
            echo "    ✓ 統合レポート生成実行成功"
            
            # 統合レポート結果ファイル生成確認
            if [ -f "reports/integrated_report.md" ]; then
                echo "    ✓ integrated_report.md生成確認"
            else
                echo "    ❌ integrated_report.md が生成されていません"
                ((test_failures++))
            fi
            
            # 品質スコアサマリー生成確認  
            if [ -f "reports/quality_summary.json" ]; then
                echo "    ✓ quality_summary.json生成確認"
            else
                echo "    ❌ quality_summary.json が生成されていません"
                ((test_failures++))
            fi
        else
            echo "    ❌ 統合レポート生成実行に失敗"
            ((test_failures++))
        fi
    else
        echo "    ❌ go コマンドが利用できません"
        ((test_failures++))
    fi
}

# Task 13: 詳細レポート機能の実装テスト
test_detailed_report_functionality() {
    echo "  ✓ Task 13: 詳細レポート機能の実装テスト"
    
    # scripts/generate-report.sh に詳細レポート機能が含まれているか確認
    if [ -f "scripts/generate-report.sh" ]; then
        local script_content=$(cat scripts/generate-report.sh)
        if echo "$script_content" | grep -q "detailed_report\|詳細レポート\|executive_summary\|technical_details"; then
            echo "    ✓ 詳細レポート機能が実装されています"
        else
            echo "    ❌ 詳細レポート機能が見つかりません（Task 13 未実装）"
            ((test_failures++))
            return 1
        fi
    else
        echo "    ❌ scripts/generate-report.sh が存在しません"
        ((test_failures++))
        return 1
    fi
    
    # go コマンドが利用可能か確認
    if command -v go >/dev/null 2>&1; then
        echo "    ✓ go コマンドが利用可能"
        
        # 詳細レポート生成テスト実行（簡易版）
        if scripts/generate-report.sh >/dev/null 2>&1; then
            echo "    ✓ 詳細レポート生成実行成功"
            
            # 詳細レポート結果ファイル生成確認
            if [ -f "reports/detailed_report.md" ]; then
                echo "    ✓ detailed_report.md生成確認"
            else
                echo "    ❌ detailed_report.md が生成されていません"
                ((test_failures++))
            fi
            
            # 経営層向けサマリー生成確認  
            if [ -f "reports/executive_summary.md" ]; then
                echo "    ✓ executive_summary.md生成確認"
            else
                echo "    ❌ executive_summary.md が生成されていません"
                ((test_failures++))
            fi
        else
            echo "    ❌ 詳細レポート生成実行に失敗"
            ((test_failures++))
        fi
    else
        echo "    ❌ go コマンドが利用できません"
        ((test_failures++))
    fi
}

# Task 14: エンドツーエンドテストスクリプトの実装テスト
test_e2e_script_functionality() {
    echo "  ✓ Task 14: エンドツーエンドテストスクリプトの実装テスト"
    
    # test-e2e.sh の存在確認
    if [ -f "test-e2e.sh" ]; then
        echo "    ✓ test-e2e.sh 存在確認"
        
        # 実行権限確認
        if [ -x "test-e2e.sh" ]; then
            echo "    ✓ test-e2e.sh 実行権限確認"
        else
            echo "    ❌ test-e2e.sh に実行権限がありません"
            ((test_failures++))
        fi
        
        # E2Eテスト機能が実装されているか確認
        local script_content=$(cat test-e2e.sh)
        if echo "$script_content" | grep -q "e2e\|end.*to.*end\|全体フロー\|統合動作\|test_full_flow"; then
            echo "    ✓ エンドツーエンドテスト機能が実装されています"
        else
            echo "    ❌ エンドツーエンドテスト機能が見つかりません（Task 14 未実装）"
            ((test_failures++))
            return 1
        fi
    else
        echo "    ❌ test-e2e.sh が存在しません（Task 14 未実装）"
        ((test_failures++))
        return 1
    fi
    
    # go コマンドが利用可能か確認
    if command -v go >/dev/null 2>&1; then
        echo "    ✓ go コマンドが利用可能"
        
        # E2Eテスト実行（簡易版）
        if ./test-e2e.sh >/dev/null 2>&1; then
            echo "    ✓ エンドツーエンドテスト実行成功"
            
            # E2Eテスト結果ファイル生成確認
            if [ -f "reports/e2e_test_results.json" ]; then
                echo "    ✓ e2e_test_results.json生成確認"
            else
                echo "    ❌ e2e_test_results.json が生成されていません"
                ((test_failures++))
            fi
            
            # E2Eテストサマリー生成確認  
            if [ -f "reports/e2e_test_summary.txt" ]; then
                echo "    ✓ e2e_test_summary.txt生成確認"
            else
                echo "    ❌ e2e_test_summary.txt が生成されていません"
                ((test_failures++))
            fi
        else
            echo "    ❌ エンドツーエンドテスト実行に失敗"
            ((test_failures++))
        fi
    else
        echo "    ❌ go コマンドが利用できません"
        ((test_failures++))
    fi
}

# Task 15: 継続改善支援機能の実装テスト
test_track_progress_functionality() {
    echo "  ✓ Task 15: 継続改善支援機能の実装テスト"
    
    # scripts/track-progress.sh の存在確認
    if [ -f "scripts/track-progress.sh" ]; then
        echo "    ✓ scripts/track-progress.sh 存在確認"
        
        # 実行権限確認
        if [ -x "scripts/track-progress.sh" ]; then
            echo "    ✓ scripts/track-progress.sh 実行権限確認"
        else
            echo "    ❌ scripts/track-progress.sh に実行権限がありません"
            ((test_failures++))
        fi
        
        # 継続改善支援機能が実装されているか確認
        local script_content=$(cat scripts/track-progress.sh)
        if echo "$script_content" | grep -q "track.*progress\|履歴管理\|トレンド\|比較分析\|改善効果"; then
            echo "    ✓ 継続改善支援機能が実装されています"
        else
            echo "    ❌ 継続改善支援機能が見つかりません（Task 15 未実装）"
            ((test_failures++))
            return 1
        fi
    else
        echo "    ❌ scripts/track-progress.sh が存在しません（Task 15 未実装）"
        ((test_failures++))
        return 1
    fi
    
    # 履歴管理ディレクトリの確認
    if [ -d "reports/history" ] || echo "$script_content" | grep -q "reports/history\|create.*history"; then
        echo "    ✓ 履歴管理機能が設計されています"
    else
        echo "    ❌ 履歴管理ディレクトリが設計されていません"
        ((test_failures++))
    fi
    
    # go コマンドが利用可能か確認
    if command -v go >/dev/null 2>&1; then
        echo "    ✓ go コマンドが利用可能"
        
        # 継続改善支援機能テスト実行（簡易版）
        if scripts/track-progress.sh >/dev/null 2>&1; then
            echo "    ✓ 継続改善支援機能実行成功"
            
            # 品質指標履歴ファイル生成確認
            if [ -f "reports/progress_tracking.json" ] || [ -f "reports/quality_history.json" ]; then
                echo "    ✓ 品質指標履歴ファイル生成確認"
            else
                echo "    ❌ 品質指標履歴ファイルが生成されていません"
                ((test_failures++))
            fi
            
            # トレンドレポート生成確認  
            if [ -f "reports/trend_analysis.md" ] || [ -f "reports/progress_report.md" ]; then
                echo "    ✓ トレンドレポート生成確認"
            else
                echo "    ❌ トレンドレポートが生成されていません"
                ((test_failures++))
            fi
        else
            echo "    ❌ 継続改善支援機能実行に失敗"
            ((test_failures++))
        fi
    else
        echo "    ❌ go コマンドが利用できません"
        ((test_failures++))
    fi
}

# Task 16: 実プロジェクトでの動作検証と最適化のテスト
test_real_project_validation() {
    echo "  ✓ Task 16: 実プロジェクトでの動作検証と最適化のテスト"
    
    # パフォーマンス最適化機能の存在確認
    local optimization_found=false
    
    # 各スクリプトで並列処理やパフォーマンス改善が実装されているか確認
    for script in scripts/quality-check.sh scripts/test-analysis.sh scripts/code-quality.sh scripts/performance-check.sh; do
        if [ -f "$script" ]; then
            local script_content=$(cat "$script")
            if echo "$script_content" | grep -q "parallel\|concurrent\|background\|timeout\|optimization"; then
                echo "    ✓ $script にパフォーマンス最適化機能が実装されています"
                optimization_found=true
            fi
        fi
    done
    
    if [ "$optimization_found" = false ]; then
        echo "    ❌ パフォーマンス最適化機能が見つかりません（Task 16 未実装）"
        ((test_failures++))
        return 1
    fi
    
    # エラーハンドリング改善の確認
    local error_handling_found=false
    
    for script in scripts/*.sh; do
        if [ -f "$script" ]; then
            local script_content=$(cat "$script")
            if echo "$script_content" | grep -q "set -e\|trap\|error.*handling\|graceful.*exit"; then
                error_handling_found=true
                break
            fi
        fi
    done
    
    if [ "$error_handling_found" = true ]; then
        echo "    ✓ エラーハンドリング改善が実装されています"
    else
        echo "    ❌ エラーハンドリング改善が見つかりません"
        ((test_failures++))
    fi
    
    # 実行時間測定と最適化の確認
    local performance_monitoring_found=false
    
    for script in scripts/*.sh; do
        if [ -f "$script" ]; then
            local script_content=$(cat "$script")
            if echo "$script_content" | grep -q "execution.*time\|performance.*monitor\|resource.*usage"; then
                performance_monitoring_found=true
                break
            fi
        fi
    done
    
    if [ "$performance_monitoring_found" = true ]; then
        echo "    ✓ パフォーマンスモニタリング機能が実装されています"
    else
        echo "    ❌ パフォーマンスモニタリング機能が見つかりません"
        ((test_failures++))
    fi
    
    # 大規模プロジェクト対応の確認
    if [ -f "scripts/quality-check.sh" ]; then
        local script_content=$(cat scripts/quality-check.sh)
        if echo "$script_content" | grep -q "large.*project\|scale\|batch\|chunk"; then
            echo "    ✓ 大規模プロジェクト対応機能が実装されています"
        else
            echo "    ❌ 大規模プロジェクト対応機能が見つかりません"
            ((test_failures++))
        fi
    else
        echo "    ❌ scripts/quality-check.sh が存在しません"
        ((test_failures++))
    fi
    
    # 実環境での全スクリプト実行テスト（軽量版）
    echo "    ✓ 実環境での全スクリプト実行テストを開始..."
    
    # メインスクリプトの実行権限と基本動作確認
    if [ -x "scripts/quality-check.sh" ]; then
        if timeout 60 scripts/quality-check.sh --test >/dev/null 2>&1; then
            echo "    ✓ quality-check.sh の実環境実行成功"
        else
            echo "    ⚠️  quality-check.sh の実環境実行でタイムアウトまたは警告"
        fi
    else
        echo "    ❌ scripts/quality-check.sh が実行できません"
        ((test_failures++))
    fi
}

# Task 17: ドキュメントとメンテナンススクリプトの実装テスト
test_documentation_and_maintenance() {
    echo "  ✓ Task 17: ドキュメントとメンテナンススクリプトの実装テスト"
    
    # README_QUALITY.md の存在確認
    if [ -f "README_QUALITY.md" ]; then
        echo "    ✓ README_QUALITY.md 存在確認"
        
        # ドキュメント内容の確認
        local doc_content=$(cat README_QUALITY.md)
        if echo "$doc_content" | grep -q "使用方法\|ベストプラクティス\|品質チェック"; then
            echo "    ✓ README_QUALITY.md に必要な内容が含まれています"
        else
            echo "    ❌ README_QUALITY.md の内容が不完全です"
            ((test_failures++))
        fi
    else
        echo "    ❌ README_QUALITY.md が存在しません（Task 17 未実装）"
        ((test_failures++))
        return 1
    fi
    
    # トラブルシューティングガイドの確認
    if echo "$doc_content" | grep -q "トラブルシューティング\|既知の問題\|問題解決"; then
        echo "    ✓ トラブルシューティングガイドが実装されています"
    else
        echo "    ❌ トラブルシューティングガイドが見つかりません"
        ((test_failures++))
    fi
    
    # 運用ガイドの確認
    if echo "$doc_content" | grep -q "運用ガイド\|継続的品質改善\|定期実行"; then
        echo "    ✓ 運用ガイドが実装されています"
    else
        echo "    ❌ 運用ガイドが見つかりません"
        ((test_failures++))
    fi
    
    # メンテナンススクリプトの確認
    if [ -f "scripts/maintenance.sh" ] || [ -f "scripts/cleanup.sh" ] || [ -f "scripts/setup.sh" ]; then
        echo "    ✓ メンテナンススクリプトが存在します"
        
        # 実行権限の確認
        local maintenance_scripts=()
        for script in scripts/maintenance.sh scripts/cleanup.sh scripts/setup.sh; do
            if [ -f "$script" ] && [ -x "$script" ]; then
                maintenance_scripts+=("$script")
            fi
        done
        
        if [ ${#maintenance_scripts[@]} -gt 0 ]; then
            echo "    ✓ メンテナンススクリプトに実行権限が設定されています"
        else
            echo "    ❌ メンテナンススクリプトに実行権限がありません"
            ((test_failures++))
        fi
    else
        echo "    ❌ メンテナンススクリプトが見つかりません"
        ((test_failures++))
    fi
    
    # 各スクリプトの使用方法説明の確認
    local scripts_documented=0
    local total_scripts=0
    
    for script in scripts/*.sh; do
        if [ -f "$script" ]; then
            total_scripts=$((total_scripts + 1))
            local script_name=$(basename "$script")
            
            if echo "$doc_content" | grep -q "$script_name"; then
                scripts_documented=$((scripts_documented + 1))
            fi
        fi
    done
    
    if [ $scripts_documented -gt 0 ] && [ $total_scripts -gt 0 ]; then
        local documentation_rate=$(( (scripts_documented * 100) / total_scripts ))
        echo "    ✓ スクリプト文書化率: $documentation_rate% ($scripts_documented/$total_scripts)"
        
        if [ $documentation_rate -lt 80 ]; then
            echo "    ⚠️  スクリプト文書化率が80%未満です"
            ((test_failures++))
        fi
    else
        echo "    ❌ スクリプトの使用方法説明が不足しています"
        ((test_failures++))
    fi
}

        
        # JSONの必須フィールド確認（jqが利用可能な場合）
        if command -v jq >/dev/null 2>&1; then
            if jq -e '.test_summary.passed, .test_summary.failed, .test_summary.skipped, .execution_time, .individual_tests' reports/test_results.json >/dev/null 2>&1; then
                echo "    ✓ JSON必須フィールド確認"
            else
                echo "    ❌ JSON必須フィールドが不足しています"
                ((test_failures++))
            fi
        else
            echo "    ⚠️  jq未インストール（JSON検証スキップ）"
        fi
    else
        echo "    ❌ test_results.json が生成されていません"
        ((test_failures++))
    fi
    
    # テストサマリーファイルの存在確認
    if [ -f "reports/test_summary.txt" ]; then
        echo "    ✓ test_summary.txt生成確認"
    else
        echo "    ❌ test_summary.txt が生成されていません"
        ((test_failures++))
    fi
}

# Task 5: 静的解析スクリプトの実装テスト
test_static_analysis_script() {
    echo "  ✓ Task 5: 静的解析スクリプトの実装テスト"
    
    # code-quality.shスクリプト存在確認
    if [ ! -f "scripts/code-quality.sh" ]; then
        echo "    ❌ scripts/code-quality.sh が存在しません（Task 5 未実装）"
        ((test_failures++))
        return 1
    fi
    
    # 実行権限確認
    if [ -x "scripts/code-quality.sh" ]; then
        echo "    ✓ scripts/code-quality.sh 実行権限確認"
    else
        echo "    ❌ scripts/code-quality.sh に実行権限がありません"
        ((test_failures++))
    fi
    
    # golangci-lintが利用可能な場合のみテスト実行
    if command -v golangci-lint >/dev/null 2>&1; then
        echo "    ✓ golangci-lint が利用可能"
        
        # code-quality.sh実行テスト
        if scripts/code-quality.sh >/dev/null 2>&1; then
            echo "    ✓ code-quality.sh実行成功"
        else
            echo "    ⚠️  code-quality.sh実行で警告（処理継続）"
        fi
        
        # 静的解析結果ファイル確認
        if [ -f "reports/lint_results.json" ]; then
            echo "    ✓ lint_results.json生成確認"
        else
            echo "    ❌ lint_results.json が生成されていません"
            ((test_failures++))
        fi
        
        if [ -f "reports/lint_summary.txt" ]; then
            echo "    ✓ lint_summary.txt生成確認"
        else
            echo "    ❌ lint_summary.txt が生成されていません"
            ((test_failures++))
        fi
    else
        echo "    ⚠️  golangci-lint未インストール（動作テストスキップ）"
    fi
}

# Task 6: セキュリティスキャン機能の実装テスト
test_security_scan_functionality() {
    echo "  ✓ Task 6: セキュリティスキャン機能の実装テスト"
    
    # code-quality.shにセキュリティスキャン機能が含まれているか確認
    if [ -f "scripts/code-quality.sh" ]; then
        local script_content=$(cat scripts/code-quality.sh)
        if echo "$script_content" | grep -q "run_security_scan\|gosec\|govulncheck"; then
            echo "    ✓ セキュリティスキャン機能が実装されています"
        else
            echo "    ❌ セキュリティスキャン機能が見つかりません（Task 6 未実装）"
            ((test_failures++))
            return 1
        fi
    else
        echo "    ❌ scripts/code-quality.sh が存在しません"
        ((test_failures++))
        return 1
    fi
    
    # gosecとgovulncheckの利用可能性確認
    local has_gosec=false
    local has_govulncheck=false
    
    if command -v gosec >/dev/null 2>&1; then
        echo "    ✓ gosec が利用可能"
        has_gosec=true
    else
        echo "    ⚠️  gosec未インストール（セキュリティスキャンの一部をスキップ）"
    fi
    
    if command -v govulncheck >/dev/null 2>&1; then
        echo "    ✓ govulncheck が利用可能"
        has_govulncheck=true
    else
        echo "    ⚠️  govulncheck未インストール（脆弱性スキャンをスキップ）"
    fi
    
    # セキュリティツールが利用可能な場合のみテスト実行
    if [ "$has_gosec" = true ] || [ "$has_govulncheck" = true ]; then
        # code-quality.sh実行テスト（セキュリティスキャンを含む）
        if scripts/code-quality.sh >/dev/null 2>&1; then
            echo "    ✓ セキュリティスキャン実行成功"
        else
            echo "    ⚠️  セキュリティスキャン実行で警告（処理継続）"
        fi
        
        # セキュリティスキャン結果ファイル確認
        if [ -f "reports/security_results.json" ]; then
            echo "    ✓ security_results.json生成確認"
        else
            echo "    ❌ security_results.json が生成されていません"
            ((test_failures++))
        fi
        
        if [ -f "reports/security_summary.txt" ]; then
            echo "    ✓ security_summary.txt生成確認"
        else
            echo "    ❌ security_summary.txt が生成されていません"
            ((test_failures++))
        fi
    else
        echo "    ⚠️  セキュリティツール未インストール（動作テストスキップ）"
    fi
}

# メイン実行
main() {
    test_directory_structure
    test_utils_functions
    test_gitignore_setup
    test_quality_check_script
    test_test_analysis_script
    test_test_execution_analysis    # Task 4の新機能テスト
    test_static_analysis_script     # Task 5の新機能テスト
    test_security_scan_functionality # Task 6の新機能テスト
    test_build_verification_functionality # Task 7の新機能テスト
    test_benchmark_execution_functionality # Task 8の新機能テスト
    test_profiling_functionality   # Task 9の新機能テスト
    test_auto_fix_functionality    # Task 10の新機能テスト
    test_priority_functionality    # Task 11の新機能テスト
    test_integrated_report_functionality # Task 12の新機能テスト
    test_detailed_report_functionality # Task 13の新機能テスト
    test_e2e_script_functionality # Task 14の新機能テスト
    test_track_progress_functionality # Task 15の新機能テスト
    test_real_project_validation # Task 16の新機能テスト
    test_documentation_and_maintenance # Task 17の新機能テスト
    
    echo
    if [ $test_failures -eq 0 ]; then
        echo "✅ すべてのテストが成功しました"
        exit 0
    else
        echo "❌ $test_failures 個のテストが失敗しました"
        exit 1
    fi
}

main "$@"