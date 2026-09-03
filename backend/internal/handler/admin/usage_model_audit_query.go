package admin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func parseOptionalBoolQueryParam(c *gin.Context, name string) (*bool, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s value, use true or false", name)
	}
	return &value, nil
}
