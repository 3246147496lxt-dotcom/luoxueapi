package handler

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// A write plan retains only hashes, so WS turn bookkeeping never keeps the
// conversation text alive until asynchronous usage and audit work finishes.
type cyberSessionBlockWritePlan struct {
	scopeKey string
	keys     []string
}

// Wait for the block to become visible before the handler returns, but never
// hold up usage recording for an unhealthy cache. Some Redis clients do not
// apply context deadlines to socket reads, so also bound the caller's wait.
func persistCyberSessionBlockPlan(gatewayService *service.OpenAIGatewayService, plan cyberSessionBlockWritePlan, timeout time.Duration) {
	if gatewayService == nil || len(plan.keys) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		gatewayService.MarkCyberSessionBlockedKeys(ctx, plan.scopeKey, plan.keys)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

func buildCyberSessionBlockWritePlan(apiKeyID int64, c *gin.Context, body []byte) cyberSessionBlockWritePlan {
	plan := cyberSessionBlockWritePlan{}
	if key := service.CyberSessionBlockKey(apiKeyID, c, body); key != "" {
		plan.keys = append(plan.keys, key)
	}
	transcriptKeys := service.CyberSessionTranscriptBlockKeys(apiKeyID, body)
	for _, key := range transcriptKeys {
		if len(plan.keys) == 0 || key != plan.keys[0] {
			plan.keys = append(plan.keys, key)
		}
	}
	if len(transcriptKeys) > 0 && c != nil {
		plan.scopeKey = service.CyberSessionScopeKey(apiKeyID, strings.TrimSpace(ip.GetClientIP(c)), c.GetHeader("User-Agent"))
	}
	return plan
}

func findBlockedCyberSessionKey(ctx context.Context, gatewayService *service.OpenAIGatewayService, apiKeyID int64, c *gin.Context, body []byte) string {
	if gatewayService == nil {
		return ""
	}
	clientIP, userAgent := "", ""
	if c != nil {
		clientIP = strings.TrimSpace(ip.GetClientIP(c))
		userAgent = c.GetHeader("User-Agent")
	}
	return gatewayService.FindCyberSessionBlockedForRequest(ctx, apiKeyID, c, body, clientIP, userAgent)
}

type cyberSessionTurnPlans struct {
	mu    sync.Mutex
	plans map[int]cyberSessionBlockWritePlan
}

func newCyberSessionTurnPlans() *cyberSessionTurnPlans {
	return &cyberSessionTurnPlans{plans: make(map[int]cyberSessionBlockWritePlan)}
}

func (p *cyberSessionTurnPlans) set(turn int, plan cyberSessionBlockWritePlan) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.plans[turn] = plan
}

func (p *cyberSessionTurnPlans) take(turn int) cyberSessionBlockWritePlan {
	p.mu.Lock()
	defer p.mu.Unlock()
	plan := p.plans[turn]
	delete(p.plans, turn)
	return plan
}
