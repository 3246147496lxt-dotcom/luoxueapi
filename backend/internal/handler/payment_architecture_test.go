package handler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestPaymentHandlersDoNotDependOnEntEntities(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate payment architecture test")
	}
	handlerDir := filepath.Dir(currentFile)
	files := []string{
		filepath.Join(handlerDir, "payment_handler.go"),
		filepath.Join(handlerDir, "admin", "payment_handler.go"),
	}
	rawEntityCalls := map[string]struct{}{
		"GetOrder":                    {},
		"GetOrderByID":                {},
		"GetUserOrders":               {},
		"AdminListOrders":             {},
		"VerifyOrderByOutTradeNo":     {},
		"VerifyOrderPublic":           {},
		"GetPublicOrderByResumeToken": {},
		"GetOrderAuditLogs":           {},
	}

	for _, path := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, spec := range parsed.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatalf("unquote import in %s: %v", path, err)
			}
			if importPath == "github.com/Wei-Shaw/sub2api/ent" || strings.HasPrefix(importPath, "github.com/Wei-Shaw/sub2api/ent/") {
				t.Errorf("%s imports persistence package %q", path, importPath)
			}
		}

		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if _, forbidden := rawEntityCalls[selector.Sel.Name]; forbidden {
				t.Errorf("%s calls raw entity service method %s", path, selector.Sel.Name)
			}
			return true
		})
	}
}
