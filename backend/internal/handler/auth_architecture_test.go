package handler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	authapplication "github.com/Wei-Shaw/sub2api/internal/modules/auth/application"
	"github.com/stretchr/testify/require"
)

func TestProductionAuthHandlersDoNotImportEnt(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "auth") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		require.NoError(t, err, name)
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			require.NoError(t, err, name)
			require.NotContains(t, importPath, "/ent", "%s must use auth ports instead of ORM entities", name)
		}
	}
}

func TestAuthHandlerUsesNarrowUseCaseFields(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("auth_handler.go"), nil, 0)
	require.NoError(t, err)

	want := map[string]string{
		"loginUseCases":   "LoginUseCases",
		"signupUseCases":  "SignupUseCases",
		"pendingIdentity": "PendingIdentityUseCases",
	}
	found := make(map[string]string, len(want))
	ast.Inspect(file, func(node ast.Node) bool {
		field, ok := node.(*ast.Field)
		if !ok {
			return true
		}
		ident, ok := field.Type.(*ast.Ident)
		if !ok {
			return true
		}
		for _, name := range field.Names {
			if _, tracked := want[name.Name]; tracked {
				found[name.Name] = ident.Name
			}
		}
		return true
	})
	require.Equal(t, want, found)
}

func TestAuthServiceDoesNotExposeEntClientEscapeHatch(t *testing.T) {
	servicePath := filepath.Join("..", "service", "auth_service.go")
	source, err := os.ReadFile(servicePath)
	require.NoError(t, err)
	require.NotContains(t, string(source), "EntClient()")
}

func TestAuthModuleFacadeIsLiveWiredWithoutChangingLoginSignupResponseFlow(t *testing.T) {
	read := func(path string) string {
		source, err := os.ReadFile(path)
		require.NoError(t, err, path)
		return string(source)
	}

	portsSource := read(filepath.Join("..", "modules", "auth", "ports", "use_cases.go"))
	require.Contains(t, portsSource, "type AuthUnitOfWork interface")
	require.Contains(t, portsSource, "WithinAuthUnit")

	serviceWire := read(filepath.Join("..", "service", "wire.go"))
	require.Contains(t, serviceWire, "NewAuthModuleFacade")
	handlerWire := read("wire.go")
	require.Contains(t, handlerWire, "ProvideAuthHandler")
	generatedWire := read(filepath.Join("..", "..", "cmd", "server", "wire_gen.go"))
	require.Contains(t, generatedWire, "service.NewAuthModuleFacade(authService)")
	require.Contains(t, generatedWire, "handler.ProvideAuthHandler(")

	handlerSource := read("auth_handler.go")
	require.Contains(t, handlerSource, "authModule")
	require.Contains(t, handlerSource, "*authapplication.Facade")
	require.Contains(t, handlerSource, "h.authModule.Refresh(")
	require.Contains(t, handlerSource, "h.authModule.Revoke(")
	require.Contains(t, handlerSource, "h.authModule.RevokeAll(")

	// Login and signup still require service.User for the exact JSON user
	// snapshot, 2FA ordering, and access-token fallback. Keep these calls as a
	// characterization guard until the module owns an equivalent user snapshot.
	require.Contains(t, handlerSource, "h.loginCases().Login(")
	require.Contains(t, handlerSource, "h.signupCases().RegisterWithVerification(")
	require.NotContains(t, handlerSource, "h.authModule.Login(")
	require.NotContains(t, handlerSource, "h.authModule.Signup(")

	pendingSource := read("auth_oauth_pending_flow.go")
	require.Contains(t, pendingSource, "h.authModule.ConsumePendingIdentity(")
}

func TestProvideAuthHandlerInjectsModuleFacade(t *testing.T) {
	facade := authapplication.NewFacade(authapplication.Dependencies{})
	authHandler := ProvideAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, facade)

	require.Same(t, facade, authHandler.authModule)
}
