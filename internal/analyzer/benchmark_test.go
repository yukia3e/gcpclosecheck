package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// BenchmarkAnalyzer はアナライザーのパフォーマンスをベンチマークする
func BenchmarkAnalyzer(b *testing.B) {
	testCases := []struct {
		name string
		src  string
	}{
		{
			name: "small_file",
			src: `package main
import (
	"context"
	"cloud.google.com/go/storage"
)
func main() {
	client, _ := storage.NewClient(context.Background())
	defer client.Close()
}`,
		},
		{
			name: "medium_file",
			src: generateMediumFile(),
		},
		{
			name: "large_file", 
			src: generateLargeFile(),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tc.src, parser.ParseComments)
			if err != nil {
				b.Fatalf("Failed to parse test file: %v", err)
			}

			pass := &analysis.Pass{
				Fset:  fset,
				Files: []*ast.File{file},
				Report: func(analysis.Diagnostic) {
					// ベンチマーク中は出力を抑制
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Analyzer.Run(pass)
				if err != nil {
					b.Fatalf("Analyzer failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkMemoryUsage はメモリ使用量をベンチマークする
func BenchmarkMemoryUsage(b *testing.B) {
	src := generateLargeFile()
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		b.Fatalf("Failed to parse test file: %v", err)
	}

	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{file},
		Report: func(analysis.Diagnostic) {},
	}

	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := Analyzer.Run(pass)
		if err != nil {
			b.Fatalf("Analyzer failed: %v", err)
		}
	}
}

// BenchmarkConcurrentAnalysis は並行解析のパフォーマンスをテストする
func BenchmarkConcurrentAnalysis(b *testing.B) {
	testFiles := []string{
		generateSmallFile(),
		generateMediumFile(),
		generateLargeFile(),
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for i, src := range testFiles {
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, 
					"test"+string(rune(i))+".go", src, parser.ParseComments)
				if err != nil {
					b.Fatalf("Failed to parse test file: %v", err)
				}

				pass := &analysis.Pass{
					Fset:  fset,
					Files: []*ast.File{file},
					Report: func(analysis.Diagnostic) {},
				}

				_, err = Analyzer.Run(pass)
				if err != nil {
					b.Fatalf("Analyzer failed: %v", err)
				}
			}
		}
	})
}

// generateSmallFile は小さなテストファイルを生成する
func generateSmallFile() string {
	return `package main
import (
	"context"
	"cloud.google.com/go/storage"
)
func main() {
	client, _ := storage.NewClient(context.Background())
	defer client.Close()
}`
}

// generateMediumFile は中程度のテストファイルを生成する  
func generateMediumFile() string {
	var sb strings.Builder
	sb.WriteString(`package main
import (
	"context"
	"cloud.google.com/go/storage"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
)
`)

	// 10個の関数を生成
	for i := 0; i < 10; i++ {
		sb.WriteString("func function")
		sb.WriteString(string(rune('0' + i)))
		sb.WriteString(`(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	
	pubsubClient, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	defer pubsubClient.Close()
	
	return nil
}
`)
	}

	return sb.String()
}

// generateLargeFile は大きなテストファイルを生成する
func generateLargeFile() string {
	var sb strings.Builder
	sb.WriteString(`package main
import (
	"context"
	"cloud.google.com/go/storage"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	vision "cloud.google.com/go/vision/apiv1"
)
`)

	// 100個の関数を生成
	for i := 0; i < 100; i++ {
		sb.WriteString("func function")
		sb.WriteString(fmt.Sprintf("%d", i))
		sb.WriteString(`(ctx context.Context) error {
	// Storage
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer storageClient.Close()
	
	// PubSub
	pubsubClient, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		return err
	}
	defer pubsubClient.Close()
	
	// Spanner
	spannerClient, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		return err
	}
	defer spannerClient.Close()
	
	// Vision
	visionClient, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	defer visionClient.Close()
	
	// Context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	
	return nil
}
`)
	}

	return sb.String()
}