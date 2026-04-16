package indexer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
)

// Symbol represents a top-level named declaration extracted from a source file.
type Symbol struct {
	Name string // identifier name
	Kind string // "func" | "method" | "type" | "const" | "var"
	Code string // relevant source snippet (first 500 chars)
}

// ExtractSymbols extracts top-level named symbols from a source file.
// For .go files it uses go/ast. For other files it uses language-specific regex.
func ExtractSymbols(filename string, src []byte) []Symbol {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return extractGoSymbols(src)
	case ".ts", ".tsx", ".js", ".jsx":
		return extractPatternSymbols(src, tsPatterns)
	case ".py":
		return extractPatternSymbols(src, pyPatterns)
	default:
		return nil
	}
}

// extractGoSymbols uses go/ast to extract symbols from Go source.
func extractGoSymbols(src []byte) []Symbol {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		// Partial parse still yields some declarations — continue.
		if f == nil {
			return nil
		}
	}

	srcStr := string(src)
	var symbols []Symbol

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			kind := "func"
			if d.Recv != nil && len(d.Recv.List) > 0 {
				kind = "method"
			}
			start := fset.Position(d.Pos()).Offset
			end := fset.Position(d.End()).Offset
			if end > len(srcStr) {
				end = len(srcStr)
			}
			code := srcStr[start:end]
			if len(code) > 500 {
				code = code[:500] + "..."
			}
			symbols = append(symbols, Symbol{Name: d.Name.Name, Kind: kind, Code: code})

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					start := fset.Position(d.Pos()).Offset
					end := fset.Position(d.End()).Offset
					if end > len(srcStr) {
						end = len(srcStr)
					}
					code := srcStr[start:end]
					if len(code) > 500 {
						code = code[:500] + "..."
					}
					symbols = append(symbols, Symbol{Name: s.Name.Name, Kind: "type", Code: code})
				}
			}
		}
	}
	return symbols
}

// patternSet defines regex patterns for extracting symbols from a language.
type patternSet struct {
	kind    string
	pattern *regexp.Regexp
}

var tsPatterns = []patternSet{
	{"func", regexp.MustCompile(`(?m)^(?:export\s+)?(?:async\s+)?function\s+(\w+)`)},
	{"type", regexp.MustCompile(`(?m)^(?:export\s+)?class\s+(\w+)`)},
	{"type", regexp.MustCompile(`(?m)^(?:export\s+)?(?:interface|type)\s+(\w+)`)},
}

var pyPatterns = []patternSet{
	{"func", regexp.MustCompile(`(?m)^def\s+(\w+)`)},
	{"type", regexp.MustCompile(`(?m)^class\s+(\w+)`)},
}

// extractPatternSymbols applies regex patterns to extract symbol names.
// Code snippet is the matched line only.
func extractPatternSymbols(src []byte, patterns []patternSet) []Symbol {
	var symbols []Symbol
	lines := strings.Split(string(src), "\n")
	for _, ps := range patterns {
		for _, line := range lines {
			m := ps.pattern.FindStringSubmatch(line)
			if len(m) >= 2 {
				name := m[1]
				code := strings.TrimSpace(line)
				if len(code) > 200 {
					code = code[:200]
				}
				symbols = append(symbols, Symbol{Name: name, Kind: ps.kind, Code: code})
			}
		}
	}
	return symbols
}
