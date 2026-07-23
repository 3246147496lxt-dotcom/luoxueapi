package admin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoreAdminHandlersDependOnNarrowCapabilities(t *testing.T) {
	want := map[string]string{
		"account_handler.go": "AccountAdminUseCases",
		"group_handler.go":   "GroupAdminUseCases",
		"user_handler.go":    "UserAdminUseCases",
		"proxy_handler.go":   "ProxyAdminUseCases",
	}
	for path, capability := range want {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		require.NoError(t, err, path)
		found := false
		ast.Inspect(file, func(node ast.Node) bool {
			field, ok := node.(*ast.Field)
			if !ok {
				return true
			}
			ident, ok := field.Type.(*ast.Ident)
			if !ok || ident.Name != capability {
				return true
			}
			for _, name := range field.Names {
				if name.Name == "adminService" {
					found = true
				}
			}
			return true
		})
		require.True(t, found, "%s must depend on %s", path, capability)
	}
}
