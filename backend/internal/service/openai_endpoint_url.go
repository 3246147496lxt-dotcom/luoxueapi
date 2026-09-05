package service

import (
	"net/url"
	"strings"
)

// deepSeekPlatformName is kept local to the endpoint builder so this small
// compatibility layer can be used while the platform constants are being
// introduced in older deployments.  Platform matching is deliberately
// case-insensitive because account data may have been imported from older
// versions with inconsistent casing.
const deepSeekPlatformName = "deepseek"

func isDeepSeekPlatform(platform string) bool {
	return strings.EqualFold(strings.TrimSpace(platform), deepSeekPlatformName)
}

// buildOpenAIEndpointURLForPlatform applies the endpoint convention of the
// selected upstream platform while retaining the generic URL normalisation
// rules (trailing slashes, explicit version prefixes, query strings, etc.).
//
// DeepSeek's public API uses /chat/completions and /responses directly when a
// root host is configured; unlike OpenAI it does not require a synthetic /v1
// prefix.  An explicitly versioned base (for example https://relay/v1) is
// respected, so existing relay configurations keep their exact path.
func buildOpenAIEndpointURLForPlatform(platform string, base string, endpoint string) string {
	if isDeepSeekPlatform(platform) {
		// Only remove a complete "/v1" path segment; do not turn a future
		// version such as "/v10/..." into a malformed relative path.
		if endpoint == "/v1" {
			endpoint = ""
		} else if strings.HasPrefix(endpoint, "/v1/") {
			endpoint = strings.TrimPrefix(endpoint, "/v1")
		}
		if endpoint == "" {
			endpoint = "/"
		}
	}
	return buildOpenAIEndpointURL(base, endpoint)
}

func buildOpenAIEndpointURL(base string, endpoint string) string {
	normalized := strings.TrimSpace(base)
	endpoint = "/" + strings.TrimLeft(strings.TrimSpace(endpoint), "/")
	relative := strings.TrimPrefix(endpoint, "/v1")
	parsed, err := url.Parse(normalized)
	if err != nil {
		return strings.TrimRight(normalized, "/") + endpoint
	}
	path := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(path, endpoint) && !strings.HasSuffix(path, relative) {
		if openAIBaseURLHasVersionSuffix(path) {
			path += relative
		} else {
			path += endpoint
		}
	}
	parsed.Path = path
	parsed.RawPath = ""
	parsed.Fragment = ""
	return parsed.String()
}

func buildOpenAIResponsesInputTokensURL(base string) string {
	return buildOpenAIEndpointURL(base, "/v1/responses/input_tokens")
}

func openAIBaseURLHasVersionSuffix(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}

	pathValue := ""
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		pathValue = parsed.Path
	} else if slash := strings.Index(trimmed, "/"); slash >= 0 {
		pathValue = trimmed[slash:]
	}

	pathValue = strings.TrimRight(pathValue, "/")
	if pathValue == "" {
		return false
	}
	lastSlash := strings.LastIndex(pathValue, "/")
	segment := pathValue
	if lastSlash >= 0 {
		segment = pathValue[lastSlash+1:]
	}
	return isOpenAIAPIVersionSegment(segment)
}

func isOpenAIAPIVersionSegment(segment string) bool {
	s := strings.ToLower(strings.TrimSpace(segment))
	if len(s) < 2 || s[0] != 'v' || !isASCIIDigit(s[1]) {
		return false
	}

	i := 1
	for i < len(s) && isASCIIDigit(s[i]) {
		i++
	}
	if i == len(s) {
		return true
	}
	if s[i] == '.' {
		i++
		if i == len(s) || !isASCIIDigit(s[i]) {
			return false
		}
		for i < len(s) && isASCIIDigit(s[i]) {
			i++
		}
		return i == len(s)
	}

	suffix := s[i:]
	return strings.HasPrefix(suffix, "alpha") ||
		strings.HasPrefix(suffix, "beta") ||
		strings.HasPrefix(suffix, "preview")
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
