package service

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

// isOpenAIRequestPermissionError identifies a denied request capability, not
// invalid account credentials. Check it before account-wide error policies so
// an image-tool rejection cannot take healthy text traffic out of service.
func isOpenAIRequestPermissionError(account *Account, statusCode int, responseBody []byte) bool {
	if account == nil || (account.Platform != PlatformOpenAI && !account.IsCNProvider()) {
		return false
	}
	if statusCode != http.StatusForbidden {
		return false
	}

	message := strings.TrimSpace(extractUpstreamErrorMessage(responseBody))
	if message == "" {
		message = strings.TrimSpace(gjson.GetBytes(responseBody, "response.error.message").String())
	}
	if message == "" || strings.HasPrefix(message, "{") {
		if detail := gjson.GetBytes(responseBody, "detail.message"); detail.Type == gjson.String {
			message = strings.TrimSpace(detail.String())
		}
	}
	if message == "" && !gjson.ValidBytes(responseBody) {
		message = strings.TrimSpace(string(responseBody))
	}

	// Match only the known capability denial. Generic permission_error / 403,
	// suspended accounts and quoted request data must keep normal protection.
	return strings.EqualFold(message, imageGenerationPermissionMessage)
}
