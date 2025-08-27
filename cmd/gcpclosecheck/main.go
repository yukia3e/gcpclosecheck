package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/yukia3e/gcpclosecheck/internal/analyzer"
)

var (
	version    = "dev"
	buildDate  = "unknown"
	commitHash = "unknown"
)

func main() {
	// バージョンとヘルプフラグの処理（singlecheckerの前に処理）
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-V", "--version":
			printVersion()
			os.Exit(0)
		case "-h", "--help", "help":
			usage()
			os.Exit(0)
		}
	}

	// フラグを解析する前にヘルプメッセージを設定
	flag.Usage = usage

	// Analyzerにカスタムフラグを追加（go vetとの競合を避けるため固有の名前を使用）
	analyzer.Analyzer.Flags.Bool("gcpdebug", false, "enable GCP close check debug mode")
	analyzer.Analyzer.Flags.String("gcpconfig", "", "path to GCP close check configuration file")

	// デバッグモードの環境変数チェック
	if os.Getenv("GCPCLOSECHECK_DEBUG") == "1" {
		// 環境変数でデバッグモードを有効化
		os.Args = append(os.Args, "-gcpdebug")
	}

	// singlechecker パッケージを使用して analysis フレームワークと統合
	singlechecker.Main(analyzer.Analyzer)
}

func usage() {
	fmt.Fprintf(os.Stderr, `gcpclosecheck - GCP リソースの Close/Stop/Cancel 漏れを検出します

Usage: gcpclosecheck [flags] packages...

Examples:
  gcpclosecheck ./...                      # 現在のディレクトリ以下を再帰的に解析
  gcpclosecheck ./pkg/handlers             # 特定のパッケージを解析（推奨）
  gcpclosecheck -gcpdebug ./...            # デバッグモードで解析
  gcpclosecheck -gcpconfig custom.yaml ./...  # カスタム設定で解析
  go vet -vettool=gcpclosecheck ./...      # go vet との統合

推奨事項:
  - パッケージ単位での解析を推奨（./internal/service/ 形式）
  - 個別ファイル指定は依存関係エラーの原因となる場合があります
  - 複雑な依存関係がある場合は go vet との統合を推奨

Flags:
`)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Environment Variables:
  GCPCLOSECHECK_DEBUG=1  デバッグモードを有効にする

For more information, see: https://github.com/yukia3e/gcpclosecheck
`)
}

func printVersion() {
	fmt.Printf("gcpclosecheck %s\n", version)
	fmt.Printf("Build Date: %s\n", buildDate)
	fmt.Printf("Commit: %s\n", commitHash)
	fmt.Printf("Go Version: %s\n", getGoVersion())
}

func getGoVersion() string {
	return runtime.Version()
}