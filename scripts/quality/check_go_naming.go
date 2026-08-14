package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var initialismReplacements = []struct {
	bad  string
	good string
}{
	{bad: "Oauth", good: "OAuth"},
	{bad: "Uuid", good: "UUID"},
	{bad: "Http", good: "HTTP"},
	{bad: "Json", good: "JSON"},
	{bad: "Skus", good: "SKUs"},
	{bad: "Ids", good: "IDs"},
	{bad: "Api", good: "API"},
	{bad: "Sku", good: "SKU"},
	{bad: "Url", good: "URL"},
	{bad: "Uri", good: "URI"},
	{bad: "Xml", good: "XML"},
	{bad: "Id", good: "ID"},
}

var (
	stageIdentifierPattern      = regexp.MustCompile(`(?:Phase|Batch)[0-9]|^p[0-9]+(?:$|[A-Z])|P[0-9]+$`)
	priorityIdentifierPattern   = regexp.MustCompile(`^PriorityP[0-3]$`)
	percentileIdentifierPattern = regexp.MustCompile(`^p(?:50|90|95|99)$`)
)

type violation struct {
	file       string
	line       int
	identifier string
	suggestion string
}

func main() {
	root := "backend"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	fset := token.NewFileSet()
	var violations []violation
	seen := make(map[string]struct{})
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".gocache" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			for _, ident := range declaredIdentifiers(node) {
				position := fset.Position(ident.Pos())
				if suggestion, bad := normalizedInitialisms(ident.Name); bad {
					key := fmt.Sprintf("%s:%d:%s:initialism", position.Filename, position.Line, ident.Name)
					if _, exists := seen[key]; !exists {
						seen[key] = struct{}{}
						violations = append(violations, violation{
							file:       filepath.ToSlash(position.Filename),
							line:       position.Line,
							identifier: ident.Name,
							suggestion: suggestion,
						})
					}
				}
				if hasStageNumber(ident.Name) {
					key := fmt.Sprintf("%s:%d:%s:stage", position.Filename, position.Line, ident.Name)
					if _, exists := seen[key]; !exists {
						seen[key] = struct{}{}
						violations = append(violations, violation{
							file:       filepath.ToSlash(position.Filename),
							line:       position.Line,
							identifier: ident.Name,
							suggestion: "use a responsibility-based name without phase or batch numbers",
						})
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Go naming check failed: %v\n", err)
		os.Exit(1)
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].file != violations[j].file {
			return violations[i].file < violations[j].file
		}
		if violations[i].line != violations[j].line {
			return violations[i].line < violations[j].line
		}
		return violations[i].identifier < violations[j].identifier
	})
	if len(violations) > 0 {
		fmt.Fprintln(os.Stderr, "Go initialism naming violations:")
		for _, item := range violations {
			fmt.Fprintf(os.Stderr, "- %s:%d: %s should be %s\n", item.file, item.line, item.identifier, item.suggestion)
		}
		os.Exit(1)
	}

	fmt.Println("Go initialism naming check passed.")
}

func hasStageNumber(name string) bool {
	if priorityIdentifierPattern.MatchString(name) || percentileIdentifierPattern.MatchString(name) {
		return false
	}
	return stageIdentifierPattern.MatchString(name)
}

func declaredIdentifiers(node ast.Node) []*ast.Ident {
	switch typed := node.(type) {
	case *ast.FuncDecl:
		return []*ast.Ident{typed.Name}
	case *ast.TypeSpec:
		return []*ast.Ident{typed.Name}
	case *ast.ValueSpec:
		return typed.Names
	case *ast.Field:
		return typed.Names
	case *ast.AssignStmt:
		if typed.Tok != token.DEFINE {
			return nil
		}
		return expressionIdentifiers(typed.Lhs)
	case *ast.RangeStmt:
		if typed.Tok != token.DEFINE {
			return nil
		}
		return expressionIdentifiers([]ast.Expr{typed.Key, typed.Value})
	default:
		return nil
	}
}

func expressionIdentifiers(expressions []ast.Expr) []*ast.Ident {
	identifiers := make([]*ast.Ident, 0, len(expressions))
	for _, expression := range expressions {
		if ident, ok := expression.(*ast.Ident); ok && ident.Name != "_" {
			identifiers = append(identifiers, ident)
		}
	}
	return identifiers
}

func normalizedInitialisms(name string) (string, bool) {
	normalized := name
	for _, replacement := range initialismReplacements {
		for start := 0; start < len(normalized); {
			index := strings.Index(normalized[start:], replacement.bad)
			if index < 0 {
				break
			}
			index += start
			after := index + len(replacement.bad)
			if after == len(normalized) || isUpperOrDigit(normalized[after]) {
				normalized = normalized[:index] + replacement.good + normalized[after:]
				start = index + len(replacement.good)
				continue
			}
			start = after
		}
	}
	return normalized, normalized != name
}

func isUpperOrDigit(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}
