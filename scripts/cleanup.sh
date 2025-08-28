#!/bin/bash

# クリーンアップスクリプト: 一時ファイルと古いレポートの削除
# 定期メンテナンス用のクリーンアップ機能を提供

set -e
source "$(dirname "$0")/utils.sh"

# 設定
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPORTS_DIR="$PROJECT_ROOT/reports"
TMP_DIR="$PROJECT_ROOT/tmp"

# 保持期間（日数）
DEFAULT_RETENTION_DAYS=30
TEMP_FILE_RETENTION_HOURS=24

# クリーンアップオプション
CLEANUP_REPORTS=false
CLEANUP_TEMP=false
CLEANUP_HISTORY=false
CLEANUP_ALL=false
RETENTION_DAYS=$DEFAULT_RETENTION_DAYS

# 使用方法表示
show_help() {
    cat << EOF
品質チェックツール クリーンアップスクリプト

使用方法:
    $0 [オプション]

オプション:
    --reports           古いレポートファイルを削除
    --temp              一時ファイルを削除
    --history           古い履歴データを削除
    --all               全てのクリーンアップを実行
    --days N            保持期間を指定（デフォルト: $DEFAULT_RETENTION_DAYS日）
    --dry-run           実際の削除は行わず、削除対象を表示のみ
    --help              このヘルプを表示

例:
    $0 --reports --days 7      # 7日以上古いレポートを削除
    $0 --temp                  # 一時ファイルのみ削除
    $0 --all --dry-run         # 全削除対象を表示（実際の削除なし）
    $0 --all                   # 全クリーンアップ実行

EOF
}

# 引数解析
parse_arguments() {
    local dry_run=false
    
    while [ $# -gt 0 ]; do
        case $1 in
            --reports)
                CLEANUP_REPORTS=true
                ;;
            --temp)
                CLEANUP_TEMP=true
                ;;
            --history)
                CLEANUP_HISTORY=true
                ;;
            --all)
                CLEANUP_ALL=true
                ;;
            --days)
                if [ -n "$2" ] && [ "$2" -gt 0 ] 2>/dev/null; then
                    RETENTION_DAYS=$2
                    shift
                else
                    log_error "無効な保持期間: $2"
                    exit 1
                fi
                ;;
            --dry-run)
                dry_run=true
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
    
    # デフォルト: なにも指定されていない場合はヘルプ表示
    if [ "$CLEANUP_REPORTS" = false ] && [ "$CLEANUP_TEMP" = false ] && \
       [ "$CLEANUP_HISTORY" = false ] && [ "$CLEANUP_ALL" = false ]; then
        show_help
        exit 0
    fi
    
    # dry-runモードの場合は警告表示
    if [ "$dry_run" = true ]; then
        log_info "🔍 DRY-RUN モード: 実際の削除は行いません"
        export DRY_RUN=true
    fi
}

# ファイル削除実行（dry-runサポート付き）
safe_remove() {
    local target="$1"
    local description="$2"
    
    if [ "$DRY_RUN" = true ]; then
        echo "    [DRY-RUN] 削除対象: $target ($description)"
        return 0
    fi
    
    if [ -f "$target" ] || [ -d "$target" ]; then
        rm -rf "$target"
        log_info "削除完了: $target ($description)"
        return 0
    else
        log_warning "対象が見つかりません: $target"
        return 1
    fi
}

# レポートファイルのクリーンアップ
cleanup_reports() {
    log_info "📋 レポートファイルのクリーンアップを開始"
    
    if [ ! -d "$REPORTS_DIR" ]; then
        log_info "レポートディレクトリが存在しません: $REPORTS_DIR"
        return 0
    fi
    
    local cleanup_count=0
    local total_size=0
    
    # 指定期間より古いレポートファイルを検索・削除
    find "$REPORTS_DIR" -type f -name "*.json" -o -name "*.txt" -o -name "*.md" -o -name "*.html" | while read -r file; do
        if [ -f "$file" ]; then
            local file_age_days
            if [ "$(uname)" = "Darwin" ]; then
                # macOS
                file_age_days=$(( ( $(date +%s) - $(stat -f %m "$file") ) / 86400 ))
            else
                # Linux
                file_age_days=$(( ( $(date +%s) - $(stat -c %Y "$file") ) / 86400 ))
            fi
            
            if [ $file_age_days -gt $RETENTION_DAYS ]; then
                local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
                total_size=$((total_size + file_size))
                
                safe_remove "$file" "${file_age_days}日前のレポート"
                cleanup_count=$((cleanup_count + 1))
            fi
        fi
    done
    
    if [ $cleanup_count -gt 0 ]; then
        local size_mb=$((total_size / 1024 / 1024))
        log_success "レポートクリーンアップ完了: ${cleanup_count}ファイル削除、${size_mb}MB解放"
    else
        log_info "削除対象のレポートファイルはありませんでした"
    fi
}

# 一時ファイルのクリーンアップ
cleanup_temp_files() {
    log_info "🗂️  一時ファイルのクリーンアップを開始"
    
    if [ ! -d "$TMP_DIR" ]; then
        log_info "一時ディレクトリが存在しません: $TMP_DIR"
        return 0
    fi
    
    local cleanup_count=0
    local total_size=0
    
    # 24時間以上古い一時ファイルを削除
    find "$TMP_DIR" -type f -name "*.tmp" -o -name "*.temp" -o -name "*.log" | while read -r file; do
        if [ -f "$file" ]; then
            local file_age_hours
            if [ "$(uname)" = "Darwin" ]; then
                # macOS
                file_age_hours=$(( ( $(date +%s) - $(stat -f %m "$file") ) / 3600 ))
            else
                # Linux
                file_age_hours=$(( ( $(date +%s) - $(stat -c %Y "$file") ) / 3600 ))
            fi
            
            if [ $file_age_hours -gt $TEMP_FILE_RETENTION_HOURS ]; then
                local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
                total_size=$((total_size + file_size))
                
                safe_remove "$file" "${file_age_hours}時間前の一時ファイル"
                cleanup_count=$((cleanup_count + 1))
            fi
        fi
    done
    
    # 空のディレクトリを削除
    find "$TMP_DIR" -type d -empty | while read -r dir; do
        if [ "$dir" != "$TMP_DIR" ]; then
            safe_remove "$dir" "空のディレクトリ"
        fi
    done
    
    if [ $cleanup_count -gt 0 ]; then
        local size_kb=$((total_size / 1024))
        log_success "一時ファイルクリーンアップ完了: ${cleanup_count}ファイル削除、${size_kb}KB解放"
    else
        log_info "削除対象の一時ファイルはありませんでした"
    fi
}

# 履歴データのクリーンアップ
cleanup_history() {
    log_info "📊 履歴データのクリーンアップを開始"
    
    local history_dir="$REPORTS_DIR/history"
    if [ ! -d "$history_dir" ]; then
        log_info "履歴ディレクトリが存在しません: $history_dir"
        return 0
    fi
    
    local cleanup_count=0
    local retention_period=$((RETENTION_DAYS * 2))  # 履歴は2倍の期間保持
    
    find "$history_dir" -type f -name "quality_metrics_*.json" | while read -r file; do
        if [ -f "$file" ]; then
            local file_age_days
            if [ "$(uname)" = "Darwin" ]; then
                # macOS
                file_age_days=$(( ( $(date +%s) - $(stat -f %m "$file") ) / 86400 ))
            else
                # Linux
                file_age_days=$(( ( $(date +%s) - $(stat -c %Y "$file") ) / 86400 ))
            fi
            
            if [ $file_age_days -gt $retention_period ]; then
                safe_remove "$file" "${file_age_days}日前の履歴データ"
                cleanup_count=$((cleanup_count + 1))
            fi
        fi
    done
    
    if [ $cleanup_count -gt 0 ]; then
        log_success "履歴データクリーンアップ完了: ${cleanup_count}ファイル削除"
    else
        log_info "削除対象の履歴データはありませんでした"
    fi
}

# システム情報表示
show_system_info() {
    log_info "💾 ディスク使用量情報"
    
    if [ -d "$REPORTS_DIR" ]; then
        local reports_size=$(du -sh "$REPORTS_DIR" 2>/dev/null | cut -f1 || echo "0B")
        echo "    レポートディレクトリ: $reports_size ($REPORTS_DIR)"
    fi
    
    if [ -d "$TMP_DIR" ]; then
        local tmp_size=$(du -sh "$TMP_DIR" 2>/dev/null | cut -f1 || echo "0B")
        echo "    一時ディレクトリ: $tmp_size ($TMP_DIR)"
    fi
    
    # プロジェクト全体のサイズ
    local project_size=$(du -sh "$PROJECT_ROOT" 2>/dev/null | cut -f1 || echo "0B")
    echo "    プロジェクト全体: $project_size ($PROJECT_ROOT)"
}

# メイン実行
main() {
    echo "🧹 品質チェックツール クリーンアップ開始"
    echo "================================================="
    
    # 引数解析
    parse_arguments "$@"
    
    # システム情報表示
    show_system_info
    echo
    
    # プロジェクトルートに移動
    cd "$PROJECT_ROOT"
    
    # 実行開始時間
    local start_time=$(date +%s)
    
    # クリーンアップ実行
    if [ "$CLEANUP_ALL" = true ]; then
        cleanup_reports
        cleanup_temp_files
        cleanup_history
    else
        if [ "$CLEANUP_REPORTS" = true ]; then
            cleanup_reports
        fi
        
        if [ "$CLEANUP_TEMP" = true ]; then
            cleanup_temp_files
        fi
        
        if [ "$CLEANUP_HISTORY" = true ]; then
            cleanup_history
        fi
    fi
    
    # 実行時間計算
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo
    echo "================================================="
    log_success "クリーンアップ完了! 実行時間: ${duration}秒"
    
    if [ "$DRY_RUN" = true ]; then
        echo
        log_info "💡 実際にクリーンアップを実行する場合は、--dry-run オプションを外してください"
    fi
    
    echo
}

# スクリプトが直接実行された場合のみメイン関数を実行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi