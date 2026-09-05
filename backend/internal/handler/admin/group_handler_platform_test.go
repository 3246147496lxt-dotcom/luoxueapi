//go:build unit

package admin

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// bindGroupPlatformJSON exercises the same Gin binding/validator path used by
// the admin group endpoints without constructing the full handler graph.
func bindGroupPlatformJSON(t *testing.T, target any, body string) error {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c.ShouldBindJSON(target)
}

func TestGroupPlatformBinding_DeepSeekAllowed(t *testing.T) {
	var create CreateGroupRequest
	require.NoError(t, bindGroupPlatformJSON(t, &create, `{"name":"deepseek","platform":"deepseek"}`))
	require.Equal(t, "deepseek", create.Platform)

	var update UpdateGroupRequest
	require.NoError(t, bindGroupPlatformJSON(t, &update, `{"platform":"deepseek"}`))
	require.Equal(t, "deepseek", update.Platform)
}

func TestGroupPlatformBinding_ExistingPlatformsRemainAllowed(t *testing.T) {
	for _, platform := range []string{"anthropic", "openai", "gemini", "antigravity", "grok"} {
		t.Run(platform, func(t *testing.T) {
			var req CreateGroupRequest
			body := fmt.Sprintf(`{"name":"g","platform":%q}`, platform)
			require.NoError(t, bindGroupPlatformJSON(t, &req, body))
			require.Equal(t, platform, req.Platform)
		})
	}
}

func TestGroupPlatformBinding_RejectsInvalidDeepSeekAliases(t *testing.T) {
	for _, platform := range []string{"DeepSeek", "deepseek ", "deepseek-api", "deepseek_v4"} {
		t.Run(platform, func(t *testing.T) {
			var req CreateGroupRequest
			body := fmt.Sprintf(`{"name":"g","platform":%q}`, platform)
			require.Error(t, bindGroupPlatformJSON(t, &req, body))
		})
	}
}
