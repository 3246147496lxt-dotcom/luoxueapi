package service

import (
	"os"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

var ginTestModeOnce sync.Once

func setGinTestMode() {
	ginTestModeOnce.Do(func() {
		gin.SetMode(gin.TestMode)
	})
}

func TestMain(m *testing.M) {
	setGinTestMode()
	os.Exit(m.Run())
}
