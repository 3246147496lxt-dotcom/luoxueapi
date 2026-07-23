package modules_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModulePackagesDoNotImportOuterLegacyLayers(t *testing.T) {
	for _, module := range []string{"scheduler", "accounts", "auth"} {
		module := module
		t.Run(module, func(t *testing.T) {
			err := filepath.Walk(module, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
					return walkErr
				}
				file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
				if err != nil {
					return err
				}
				for _, spec := range file.Imports {
					importPath, err := strconv.Unquote(spec.Path.Value)
					if err != nil {
						return err
					}
					for _, forbidden := range []string{"/internal/handler", "/internal/repository", "/internal/service"} {
						require.NotContains(t, importPath, forbidden, "%s must stay behind module ports", path)
					}
					if module == "accounts" && strings.Contains(path, string(filepath.Separator)+"application"+string(filepath.Separator)) {
						require.NotContains(t, importPath, "/modules/scheduler", "accounts application must not import scheduler implementation")
					}
				}
				return nil
			})
			require.NoError(t, err)
		})
	}
}

func TestGatewayStructsDoNotDependOnConcreteSchedulerSnapshotService(t *testing.T) {
	serviceDir := filepath.Join("..", "service")
	expectedPorts := map[string]string{
		"gateway_service.go":                "GatewayScheduler",
		"openai_gateway_service.go":         "GatewayScheduler",
		"antigravity_gateway_service.go":    "AccountSnapshotReader",
		"gemini_messages_compat_service.go": "GatewayScheduler",
	}
	for name, expectedPort := range expectedPorts {
		path := filepath.Join(serviceDir, name)
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(node ast.Node) bool {
			field, ok := node.(*ast.Field)
			if !ok {
				return true
			}
			for _, fieldName := range field.Names {
				if fieldName.Name != "schedulerSnapshot" {
					continue
				}
				port, ok := field.Type.(*ast.Ident)
				require.True(t, ok, "%s schedulerSnapshot field must use a named narrow port", name)
				if ok {
					require.Equal(t, expectedPort, port.Name, "%s schedulerSnapshot field drifted from its narrow port", name)
				}
			}
			return true
		})
	}
}

func TestLegacyAccountCreationOnlyCrossesUnitOfWorkAdapter(t *testing.T) {
	serviceDir := filepath.Join("..", "service")
	entries, err := os.ReadDir(serviceDir)
	require.NoError(t, err)

	var violations []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		if entry.Name() == "accounts_module_adapter.go" {
			// The adapter is the only permitted legacy repository invocation site.
			continue
		}
		path := filepath.Join(serviceDir, entry.Name())
		contents, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		if strings.Contains(string(contents), ".CreateWithAccountGroups(") {
			violations = append(violations, entry.Name())
		}
	}
	require.Empty(t, violations, "account creation must cross AccountUnitOfWork")
}
