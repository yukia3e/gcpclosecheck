#!/bin/bash

# 共通ユーティリティ関数
# プロジェクト品質チェック用の共通機能を提供

# ログ出力関数
log_info() {
    local message="$1"
    echo "ℹ️  $(date '+%H:%M:%S') $message"
}

log_success() {
    local message="$1" 
    echo "✅ $(date '+%H:%M:%S') $message"
}

log_error() {
    local message="$1"
    echo "❌ $(date '+%H:%M:%S') ERROR: $message" >&2
}

log_warning() {
    local message="$1"
    echo "⚠️  $(date '+%H:%M:%S') WARNING: $message"
}

# エラーハンドリング関数
handle_error() {
    local exit_code=$1
    local message="$2"
    log_error "$message"
    exit "$exit_code"
}

# 前提条件チェック関数
check_prerequisites() {
    log_info "前提条件をチェック中..."
    
    # Go言語の確認
    if ! command -v go >/dev/null 2>&1; then
        handle_error 1 "Go言語がインストールされていません"
    fi
    
    log_success "前提条件チェック完了"
}

# ディレクトリ作成関数
ensure_directories() {
    local dirs=("$@")
    for dir in "${dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
            log_info "ディレクトリ作成: $dir"
        fi
    done
}

# 改善されたエラーハンドリング付きコマンド実行
run_with_error_handling() {
    local description="$1"
    shift  # 第一引数（説明）を除去
    
    log_info "$description を実行中..."
    
    if "$@"; then
        log_success "$description 完了"
        return 0
    else
        local exit_code=$?
        log_warning "$description で問題が発生しました（処理継続）"
        return $exit_code
    fi
}

# 時間測定関数
time_command() {
    local description="$1"
    shift
    
    log_info "$description 開始..."
    local start_time=$(date +%s)
    
    if "$@"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "$description 完了 (${duration}秒)"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_error "$description 失敗 (${duration}秒)"
        return 1
    fi
}

# 強化されたエラーハンドリング - graceful exit機能
graceful_exit() {
    local exit_code=${1:-1}
    local message=${2:-"予期しないエラーが発生しました"}
    
    log_error "$message"
    log_info "クリーンアップ処理を実行中..."
    
    # 一時ファイルのクリーンアップ
    if [ -d "${TMP_DIR:-/tmp}" ]; then
        find "${TMP_DIR:-/tmp}" -name "*.tmp" -mtime +1 -delete 2>/dev/null || true
    fi
    
    # バックグラウンドプロセスの終了
    if [ -n "${background_pids:-}" ]; then
        for pid in $background_pids; do
            if kill -0 $pid 2>/dev/null; then
                kill $pid 2>/dev/null || true
            fi
        done
    fi
    
    log_info "クリーンアップ完了"
    exit $exit_code
}

# シグナルハンドラーの設定
setup_signal_handlers() {
    trap 'graceful_exit 130 "プロセスが中断されました"' INT
    trap 'graceful_exit 143 "プロセスが終了されました"' TERM
    trap 'graceful_exit 1 "予期しないエラーが発生しました"' ERR
}

# リソース使用量モニタリング
monitor_system_resources() {
    local memory_threshold_mb=${1:-1000}  # デフォルト1GB
    local disk_threshold_mb=${2:-5000}    # デフォルト5GB
    
    # メモリ使用量チェック（macOS/Linux対応）
    local memory_usage_mb=0
    if [ "$(uname)" = "Darwin" ]; then
        # macOS
        memory_usage_mb=$(ps -o rss= -p $$ | awk '{print int($1/1024)}')
    else
        # Linux
        memory_usage_mb=$(ps -o rss= -p $$ | awk '{print int($1/1024)}')
    fi
    
    if [ $memory_usage_mb -gt $memory_threshold_mb ]; then
        log_warning "高メモリ使用を検出: ${memory_usage_mb}MB (閾値: ${memory_threshold_mb}MB)"
    fi
    
    # ディスク使用量チェック
    local disk_usage_mb=$(du -sm "${PWD}" 2>/dev/null | cut -f1 || echo "0")
    if [ $disk_usage_mb -gt $disk_threshold_mb ]; then
        log_warning "高ディスク使用を検出: ${disk_usage_mb}MB (閾値: ${disk_threshold_mb}MB)"
    fi
    
    return 0
}