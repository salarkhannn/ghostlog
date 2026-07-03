package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/salarkhannn/ghostlog/internal/git"
)

type FileAnalysis struct {
	Path        string
	LinesAdded  int
	LinesRemoved int
	IsTest      bool
	IsGenerated bool
	FuncCount   int
}

func Analyze(diff git.FileDiff) FileAnalysis {
	fa := FileAnalysis{
		Path:         diff.Path,
		LinesAdded:   diff.LinesAdded,
		LinesRemoved: diff.LinesRemoved,
		IsTest:       isTestFile(diff.Path),
		IsGenerated:  isGeneratedFile(diff.RawDiff),
	}

	if strings.HasSuffix(diff.Path, ".go") {
		fa.FuncCount = countFunctions(diff.RawDiff)
	}

	return fa
}

func isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

func isGeneratedFile(rawDiff string) bool {
	markers := []string{
		"Code generated",
		"DO NOT EDIT",
		"@generated",
		"Auto-generated",
	}
	for _, m := range markers {
		if strings.Contains(rawDiff, m) {
			return true
		}
	}
	return false
}

func countFunctions(rawDiff string) int {
	var addedLines strings.Builder
	for _, line := range strings.Split(rawDiff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			addedLines.WriteString(strings.TrimPrefix(line, "+"))
			addedLines.WriteByte('\n')
		}
	}

	src := addedLines.String()
	if src == "" {
		return 0
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", "package p\n"+src, parser.SkipObjectResolution)
	if err != nil {
		return 0
	}

	count := 0
	ast.Inspect(f, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncDecl); ok {
			count++
		}
		return true
	})
	return count
}
