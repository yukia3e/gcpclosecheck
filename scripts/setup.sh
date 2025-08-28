#!/bin/bash

# セットアップ検証スクリプト: 品質チェックツールの環境確認と初期設定
# 品質チェックツール実行に必要な前提条件の確認と自動セットアップを提供

set -e
source "$(dirname "$0")/utils.sh"

# 設定
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REQUIRED_DIRS=("reports" "tmp" "reports/history")

# 必要なツールとバージョンの定義（連想配列の代替）
get_required_tool_version() {
    case "$1" in
        go) echo "1.23" ;;
        golangci-lint) echo "1.50" ;;
        git) echo "2.0" ;;
        *) echo "" ;;
    esac
}

get_optional_tool_version() {
    case "$1" in
        gosec) echo "2.15" ;;
        govulncheck) echo "1.0" ;;
        *) echo "" ;;
    esac
}

# 必要ツールリスト
REQUIRED_TOOLS_LIST=("go" "golangci-lint" "git")
OPTIONAL_TOOLS_LIST=("gosec" "govulncheck")

# セットアップオプション
VERIFY_ONLY=false
INSTALL_OPTIONAL=false
VERBOSE=false
FIX_PERMISSIONS=false

# 使用方法表示
show_help() {
    cat << EOF
品質チェックツール セットアップ・検証スクリプト

使用方法:
    $0 [オプション]

オプション:
    --verify            環境確認のみ実行（デフォルト）
    --install-optional  オプションツールも自動インストール
    --fix-permissions   スクリプトファイルの実行権限を修正
    --verbose           詳細な出力を表示
    --help              このヘルプを表示

例:
    $0                          # 基本的な環境確認
    $0 --install-optional       # オプションツール含む完全セットアップ
    $0 --verify --verbose       # 詳細な検証結果表示
    $0 --fix-permissions        # スクリプト権限修正

EOF
}

# 引数解析
parse_arguments() {
    while [ $# -gt 0 ]; do
        case $1 in
            --verify)
                VERIFY_ONLY=true
                ;;
            --install-optional)
                INSTALL_OPTIONAL=true
                ;;
            --verbose)
                VERBOSE=true
                ;;
            --fix-permissions)
                FIX_PERMISSIONS=true
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

# システム情報表示
show_system_info() {
    log_info "🖥️  システム情報"
    echo "    OS: $(uname -s) $(uname -r)"
    echo "    アーキテクチャ: $(uname -m)"
    echo "    シェル: $SHELL"
    echo "    PWD: $(pwd)"
    echo "    プロジェクトルート: $PROJECT_ROOT"
}

# バージョン比較関数
version_compare() {
    local version1="$1"
    local version2="$2"
    
    if [ "$version1" = "$version2" ]; then
        return 0
    fi
    
    local IFS=.
    local i ver1=($version1) ver2=($version2)
    
    # バージョン番号を配列に分割して比較
    for ((i=0; i<${#ver1[@]} || i<${#ver2[@]}; i++)); do
        if [[ ${ver1[i]:-0} -lt ${ver2[i]:-0} ]]; then
            return 1
        elif [[ ${ver1[i]:-0} -gt ${ver2[i]:-0} ]]; then
            return 0
        fi
    done
    return 0
}

# ツール存在確認
check_tool_exists() {
    local tool="$1"
    if command -v "$tool" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# ツールバージョン取得
get_tool_version() {
    local tool="$1"
    local version=""
    
    case "$tool" in
        go)
            version=$(go version 2>/dev/null | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | sed 's/go//' | head -1)
            ;;
        golangci-lint)
            version=$(golangci-lint --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
            ;;
        git)
            version=$(git --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
            ;;
        gosec)
            version=$(gosec -version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
            ;;
        govulncheck)
            version=$(govulncheck -version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
            ;;
        *)
            version="unknown"
            ;;
    esac
    
    echo "$version"
}

# 必須ツールチェック
check_required_tools() {
    log_info "🔧 必須ツールの確認"
    
    local all_ok=true
    
    for tool in "${REQUIRED_TOOLS_LIST[@]}"; do
        local required_version=$(get_required_tool_version "$tool")
        
        if check_tool_exists "$tool"; then
            local current_version=$(get_tool_version "$tool")
            
            if [ -n "$current_version" ] && [ "$current_version" != "unknown" ]; then
                if version_compare "$current_version" "$required_version"; then
                    log_success "✅ $tool: $current_version (>= $required_version required)"
                else
                    log_warning "⚠️  $tool: $current_version (>= $required_version required)"
                    echo "    推奨: $tool をアップデートしてください"
                fi
            else
                log_success "✅ $tool: インストール済み（バージョン不明）"
            fi
        else
            log_error "❌ $tool: インストールされていません (>= $required_version required)"
            echo "    インストール方法: https://golang.org/doc/install （Go）"
            echo "                    https://golangci-lint.run/usage/install/ （golangci-lint）"
            all_ok=false
        fi
    done
    
    return $([ "$all_ok" = true ] && echo 0 || echo 1)
}

# オプションツールチェック
check_optional_tools() {
    log_info "🔧 オプションツールの確認"
    
    for tool in "${OPTIONAL_TOOLS_LIST[@]}"; do
        local recommended_version=$(get_optional_tool_version "$tool")
        
        if check_tool_exists "$tool"; then
            local current_version=$(get_tool_version "$tool")
            
            if [ -n "$current_version" ] && [ "$current_version" != "unknown" ]; then
                log_success "✅ $tool: $current_version （推奨: >= $recommended_version）"
            else
                log_success "✅ $tool: インストール済み（バージョン不明）"
            fi
        else
            log_info "ℹ️  $tool: インストールされていません（オプション）"
            if [ "$VERBOSE" = true ]; then
                echo "    インストール方法:"
                case "$tool" in
                    gosec)
                        echo "        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
                        ;;
                    govulncheck)
                        echo "        go install golang.org/x/vuln/cmd/govulncheck@latest"
                        ;;
                esac
            fi
        fi
    done
}

# ディレクトリ構造確認・作成
check_create_directories() {
    log_info "📁 ディレクトリ構造の確認"
    
    cd "$PROJECT_ROOT"
    
    for dir in "${REQUIRED_DIRS[@]}"; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
            log_success "✅ ディレクトリ作成: $dir"
        else
            if [ "$VERBOSE" = true ]; then
                log_success "✅ ディレクトリ存在: $dir"
            fi
        fi
    done
    
    # .gitignore確認
    if [ -f ".gitignore" ]; then
        if ! grep -q "reports/" .gitignore || ! grep -q "tmp/" .gitignore; then
            log_warning "⚠️  .gitignore に reports/ または tmp/ が含まれていません"
            echo "    推奨: .gitignore に以下を追加してください:"
            echo "        reports/"
            echo "        tmp/"
        else
            if [ "$VERBOSE" = true ]; then
                log_success "✅ .gitignore設定確認済み"
            fi
        fi
    else
        log_warning "⚠️  .gitignoreファイルが存在しません"
    fi
}

# スクリプト実行権限確認・修正
check_fix_permissions() {
    log_info "🔐 スクリプト実行権限の確認"
    
    local script_files=(
        "scripts/quality-check.sh"
        "scripts/test-analysis.sh"
        "scripts/code-quality.sh"
        "scripts/performance-check.sh"
        "scripts/fix-issues.sh"
        "scripts/generate-report.sh"
        "scripts/track-progress.sh"
        "scripts/cleanup.sh"
        "scripts/setup.sh"
    )
    
    local fixed_count=0
    
    for script in "${script_files[@]}"; do
        if [ -f "$PROJECT_ROOT/$script" ]; then
            if [ ! -x "$PROJECT_ROOT/$script" ]; then
                if [ "$FIX_PERMISSIONS" = true ]; then
                    chmod +x "$PROJECT_ROOT/$script"
                    log_success "✅ 実行権限付与: $script"
                    fixed_count=$((fixed_count + 1))
                else
                    log_warning "⚠️  実行権限なし: $script"
                    echo "    修正方法: chmod +x $script"
                fi
            else
                if [ "$VERBOSE" = true ]; then
                    log_success "✅ 実行権限確認済み: $script"
                fi
            fi
        else
            log_warning "⚠️  スクリプトファイルが存在しません: $script"
        fi
    done
    
    if [ $fixed_count -gt 0 ]; then
        log_success "実行権限を $fixed_count ファイルに付与しました"
    fi
}

# Go モジュール確認
check_go_module() {
    log_info "📦 Go モジュール設定の確認"
    
    cd "$PROJECT_ROOT"
    
    if [ ! -f "go.mod" ]; then
        log_error "❌ go.mod ファイルが存在しません"
        echo "    修正方法: go mod init <module-name>"
        return 1
    fi
    
    # go.sum確認
    if [ ! -f "go.sum" ]; then
        log_warning "⚠️  go.sum ファイルが存在しません"
        echo "    推奨: go mod tidy を実行してください"
    fi
    
    # モジュールの整合性確認
    if go mod verify >/dev/null 2>&1; then
        if [ "$VERBOSE" = true ]; then
            log_success "✅ Go モジュールの整合性確認済み"
        fi
    else
        log_warning "⚠️  Go モジュールの整合性に問題があります"
        echo "    修正方法: go mod tidy && go mod verify"
    fi
    
    return 0
}

# 設定ファイル確認
check_config_files() {
    log_info "⚙️  設定ファイルの確認"
    
    cd "$PROJECT_ROOT"
    
    # .golangci.yml確認
    if [ -f ".golangci.yml" ] || [ -f ".golangci.yaml" ]; then
        if [ "$VERBOSE" = true ]; then
            log_success "✅ golangci-lint設定ファイル確認済み"
        fi
    else
        log_info "ℹ️  .golangci.yml設定ファイルが存在しません（オプション）"
        echo "    推奨: プロジェクト用のlint設定を作成することを推奨します"
    fi
}

# オプションツール自動インストール
install_optional_tools() {
    if [ "$INSTALL_OPTIONAL" != true ]; then
        return 0
    fi
    
    log_info "📥 オプションツールの自動インストール"
    
    for tool in "${OPTIONAL_TOOLS_LIST[@]}"; do
        if ! check_tool_exists "$tool"; then
            log_info "Installing $tool..."
            
            case "$tool" in
                gosec)
                    if go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; then
                        log_success "✅ gosec インストール完了"
                    else
                        log_error "❌ gosec インストール失敗"
                    fi
                    ;;
                govulncheck)
                    if go install golang.org/x/vuln/cmd/govulncheck@latest; then
                        log_success "✅ govulncheck インストール完了"
                    else
                        log_error "❌ govulncheck インストール失敗"
                    fi
                    ;;
            esac
        fi
    done
}

# 統合テスト実行
run_integration_test() {
    log_info "🧪 統合テストの実行"
    
    cd "$PROJECT_ROOT"
    
    # 基本的なGoコマンドテスト
    if go version >/dev/null 2>&1; then
        if [ "$VERBOSE" = true ]; then
            log_success "✅ Go コマンド動作確認"
        fi
    else
        log_error "❌ Go コマンドが動作しません"
        return 1
    fi
    
    # go mod tidy実行テスト
    if go mod tidy >/dev/null 2>&1; then
        if [ "$VERBOSE" = true ]; then
            log_success "✅ go mod tidy 動作確認"
        fi
    else
        log_warning "⚠️  go mod tidy で問題が発生しました"
    fi
    
    # 簡易コンパイルテスト
    if go build ./... >/dev/null 2>&1; then
        if [ "$VERBOSE" = true ]; then
            log_success "✅ プロジェクトビルド確認"
        fi
    else
        log_warning "⚠️  プロジェクトのビルドで問題があります"
        echo "    これは既存の問題の可能性があります"
    fi
}

# セットアップ結果サマリー
show_setup_summary() {
    echo
    echo "=================================================="
    log_info "📋 セットアップ結果サマリー"
    echo "=================================================="
    
    # 環境情報再表示
    echo "プロジェクト: $(basename "$PROJECT_ROOT")"
    echo "場所: $PROJECT_ROOT"
    
    # 推奨次のアクション
    echo
    log_info "💡 推奨される次のアクション:"
    echo "    1. 品質チェック実行: scripts/quality-check.sh"
    echo "    2. テスト結果確認: reports/ ディレクトリ"
    echo "    3. 定期実行設定: crontab またはCI/CD設定"
    
    if [ "$VERBOSE" = true ]; then
        echo
        echo "詳細なドキュメント: README_QUALITY.md"
        echo "問題解決方法: README_QUALITY.md のトラブルシューティング"
    fi
}

# メイン実行
main() {
    echo "🛠️  品質チェックツール セットアップ・検証開始"
    echo "=================================================="
    
    # 引数解析
    parse_arguments "$@"
    
    # システム情報表示
    if [ "$VERBOSE" = true ]; then
        show_system_info
        echo
    fi
    
    # プロジェクトルートに移動
    cd "$PROJECT_ROOT"
    
    # 実行開始時間
    local start_time=$(date +%s)
    local all_checks_passed=true
    
    # セットアップ・検証実行
    if ! check_required_tools; then
        all_checks_passed=false
    fi
    
    check_optional_tools
    check_create_directories
    check_fix_permissions
    
    if ! check_go_module; then
        all_checks_passed=false
    fi
    
    check_config_files
    
    # オプションツールインストール
    install_optional_tools
    
    # 統合テスト
    run_integration_test
    
    # 実行時間計算
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # 結果表示
    show_setup_summary
    
    echo
    echo "=================================================="
    if [ "$all_checks_passed" = true ]; then
        log_success "✅ セットアップ・検証完了! 実行時間: ${duration}秒"
        echo
        log_info "🚀 品質チェックツールの準備が整いました"
        exit 0
    else
        log_warning "⚠️  セットアップ・検証完了（一部問題あり） 実行時間: ${duration}秒"
        echo
        log_info "🔧 上記の問題を解決してから品質チェックツールをご利用ください"
        exit 1
    fi
}

# スクリプトが直接実行された場合のみメイン関数を実行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi