package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	openaiwsv2 "github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// openAIWSPolicyEnforcingFrameConn wraps a client-side FrameConn and runs
// every client→upstream frame through the OpenAI Fast Policy. It is the
// passthrough-relay equivalent of the parseClientPayload integration in the
// ingress session path. filter returns:
//   - newPayload, nil, nil: forward the (possibly mutated) payload
//   - _, *OpenAIFastBlockedError, nil: block — the wrapper sends an error
//     event via onBlock and surfaces a transport-level error so the relay
//     stops reading from the client.
//   - _, _, err: a transport error other than block.
type openAIWSPolicyEnforcingFrameConn struct {
	inner   openaiwsv2.FrameConn
	filter  func(msgType coderws.MessageType, payload []byte) ([]byte, *OpenAIFastBlockedError, error)
	onBlock func(blocked *OpenAIFastBlockedError)
}

var _ openaiwsv2.FrameConn = (*openAIWSPolicyEnforcingFrameConn)(nil)

func (c *openAIWSPolicyEnforcingFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	if c == nil || c.inner == nil {
		return coderws.MessageText, nil, errOpenAIWSConnClosed
	}
	msgType, payload, err := c.inner.ReadFrame(ctx)
	if err != nil {
		return msgType, payload, err
	}
	if c.filter == nil {
		return msgType, payload, nil
	}
	updated, blocked, filterErr := c.filter(msgType, payload)
	if filterErr != nil {
		return msgType, payload, filterErr
	}
	if blocked != nil {
		if c.onBlock != nil {
			c.onBlock(blocked)
		}
		return msgType, nil, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, blocked.Message, blocked)
	}
	return msgType, updated, nil
}

func (c *openAIWSPolicyEnforcingFrameConn) WriteFrame(ctx context.Context, msgType coderws.MessageType, payload []byte) error {
	if c == nil || c.inner == nil {
		return errOpenAIWSConnClosed
	}
	return c.inner.WriteFrame(ctx, msgType, payload)
}

func (c *openAIWSPolicyEnforcingFrameConn) Close() error {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.Close()
}

// openAIWSPassthroughPolicyModelForFrame returns the upstream-perspective
// model name that should be passed to evaluateOpenAIFastPolicy for a single
// passthrough WS frame. Mirrors the HTTP-side normalization
// (account.GetMappedModel + normalizeOpenAIModelForUpstream) so the WS path
// matches model whitelists identically.
func openAIWSPassthroughPolicyModelForFrame(account *Account, payload []byte) string {
	if account == nil || len(payload) == 0 {
		return ""
	}
	original := strings.TrimSpace(gjson.GetBytes(payload, "model").String())
	if original == "" {
		return ""
	}
	return normalizeOpenAIModelForUpstream(account, account.GetMappedModel(original))
}

func normalizeOpenAIWSPassthroughLockedModelFrame(
	payload []byte,
	eventType string,
	lockedOriginalModel string,
	lockedForwardModel string,
) ([]byte, error) {
	modelPath := ""
	switch strings.TrimSpace(eventType) {
	case "response.create":
		modelPath = "model"
	case "session.update":
		modelPath = "session.model"
	default:
		return payload, nil
	}

	modelResult := gjson.GetBytes(payload, modelPath)
	if !modelResult.Exists() {
		return payload, nil
	}
	if modelResult.Type != gjson.String {
		return nil, errors.New("websocket model must be a non-empty string")
	}
	model := strings.TrimSpace(modelResult.String())
	if model == "" {
		return nil, errors.New("websocket model must be a non-empty string")
	}
	if model != lockedOriginalModel && model != lockedForwardModel {
		return nil, errors.New("changing model within a websocket connection is not supported")
	}
	if model == lockedForwardModel {
		return payload, nil
	}
	updated, err := sjson.SetBytes(payload, modelPath, lockedForwardModel)
	if err != nil {
		return nil, fmt.Errorf("normalize websocket model: %w", err)
	}
	return updated, nil
}

type openAIWSPassthroughUsageMeta struct {
	serviceTier     atomic.Pointer[string]
	reasoningEffort atomic.Pointer[string]

	// The request-model identity is fixed for the lifetime of the connection.
	sessionRequestModel string
}

func newOpenAIWSPassthroughUsageMeta(initialRequestModel string, firstFrame []byte) *openAIWSPassthroughUsageMeta {
	meta := &openAIWSPassthroughUsageMeta{
		sessionRequestModel: strings.TrimSpace(initialRequestModel),
	}
	if meta.sessionRequestModel == "" {
		meta.sessionRequestModel = openAIWSPassthroughRequestModelForFrame(firstFrame)
	}
	return meta
}

func (m *openAIWSPassthroughUsageMeta) initFromFirstFrame(policyOutput []byte, mappedModel string) {
	if m == nil {
		return
	}
	m.serviceTier.Store(extractOpenAIServiceTierFromBody(policyOutput))
	m.reasoningEffort.Store(extractOpenAIReasoningEffortFromBody(policyOutput, mappedModel, m.sessionRequestModel))
}

func (m *openAIWSPassthroughUsageMeta) requestModelForFrame(_ []byte) string {
	if m == nil {
		return ""
	}
	return m.sessionRequestModel
}

func (m *openAIWSPassthroughUsageMeta) updateFromResponseCreate(policyOutput []byte, mappedModel string, requestModelForFrame string) {
	if m == nil {
		return
	}
	m.serviceTier.Store(extractOpenAIServiceTierFromBody(policyOutput))
	m.reasoningEffort.Store(extractOpenAIReasoningEffortFromBody(policyOutput, mappedModel, requestModelForFrame))
}

func openAIWSPassthroughRequestModelForFrame(payload []byte) string {
	if len(payload) == 0 || strings.TrimSpace(gjson.GetBytes(payload, "type").String()) != "response.create" {
		return ""
	}
	return strings.TrimSpace(gjson.GetBytes(payload, "model").String())
}

const openaiWSV2PassthroughModeFields = "ws_mode=passthrough ws_router=v2"

func (s *OpenAIGatewayService) proxyResponsesWebSocketV2Passthrough(
	ctx context.Context,
	c *gin.Context,
	clientAttempt *OpenAIWSIngressClientAttempt,
	account *Account,
	token string,
	firstClientMessage []byte,
	hooks *OpenAIWSIngressHooks,
	wsDecision OpenAIWSProtocolDecision,
) error {
	if s == nil {
		return errors.New("service is nil")
	}
	if clientAttempt == nil {
		return errors.New("client websocket is nil")
	}
	if account == nil {
		return errors.New("account is nil")
	}
	if err := validateOpenAIWSBearerToken(account, token); err != nil {
		return err
	}
	if account.IsOpenAIOAuth() && isOpenAIResponsesLiteWebSocketPayload(firstClientMessage) {
		liteFirstMessage, _, liteErr := normalizeOpenAIResponsesLiteToolsPayload(firstClientMessage)
		if liteErr != nil {
			return NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, liteErr.Error(), liteErr)
		}
		firstClientMessage = liteFirstMessage
	}
	requestModel := strings.TrimSpace(gjson.GetBytes(firstClientMessage, "model").String())
	requestPreviousResponseID := strings.TrimSpace(gjson.GetBytes(firstClientMessage, "previous_response_id").String())
	logOpenAIWSV2Passthrough(
		"relay_start account_id=%d model=%s previous_response_id=%s first_message_type=%s first_message_bytes=%d",
		account.ID,
		truncateOpenAIWSLogValue(requestModel, openAIWSLogValueMaxLen),
		truncateOpenAIWSLogValue(requestPreviousResponseID, openAIWSIDValueMaxLen),
		openaiwsv2RelayMessageTypeName(coderws.MessageText),
		len(firstClientMessage),
	)

	// Apply OpenAI Fast Policy on the first response.create frame. Subsequent
	// frames are filtered via a wrapping FrameConn below so every client→
	// upstream frame goes through the same policy evaluator/normalize/scope as
	// HTTP entrypoints.
	//
	// Supporting an in-connection model switch requires per-turn channel
	// authorization, mapping and billing snapshots. Until that contract
	// exists, lock the original, forwarded and policy model identities.
	lockedForwardModel := requestModel
	lockedOriginalModel := ""
	if hooks != nil {
		lockedOriginalModel = strings.TrimSpace(hooks.InitialRequestModel)
	}
	if lockedOriginalModel == "" {
		lockedOriginalModel = lockedForwardModel
	}
	lockedPolicyModel := openAIWSPassthroughPolicyModelForFrame(account, firstClientMessage)
	usageMeta := newOpenAIWSPassthroughUsageMeta(lockedOriginalModel, firstClientMessage)
	if !IsImageGenerationIntentForPlatform(openAIResponsesEndpoint, lockedOriginalModel, firstClientMessage, account.Platform) {
		if updatedFirst, injected, injectErr := s.injectDefaultOpenAIServiceTier(ctx, c, account, lockedPolicyModel, firstClientMessage); injectErr != nil {
			return NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", injectErr)
		} else if injected {
			firstClientMessage = updatedFirst
		}
	}
	updatedFirst, blocked, policyErr := s.applyOpenAIFastPolicyToWSResponseCreate(ctx, account, lockedPolicyModel, firstClientMessage)
	if policyErr != nil {
		return NewOpenAIWSClientCloseError(
			coderws.StatusPolicyViolation,
			"invalid websocket request payload",
			policyErr,
		)
	}
	if blocked != nil {
		MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
		// coder/websocket@v1.8.14 Conn.Write is synchronous: it acquires
		// writeFrameMu, writes the entire frame, and Flushes the underlying
		// bufio writer before returning (write.go:42 → write.go:307-311).
		// The subsequent close handshake re-acquires the same writeFrameMu
		// to send the close frame, so the error event is guaranteed to
		// reach the kernel send buffer before any close frame is queued.
		// No explicit flush hop is required here.
		eventBytes := buildOpenAIFastPolicyBlockedWSEvent(blocked)
		if eventBytes != nil {
			writeCtx, cancelWrite := context.WithTimeout(ctx, s.openAIWSWriteTimeout())
			_ = clientAttempt.WriteFrame(writeCtx, coderws.MessageText, eventBytes)
			cancelWrite()
		}
		return NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, blocked.Message, blocked)
	}
	firstClientMessage = updatedFirst

	// 在 policy filter 之后再提取 service_tier / reasoning_effort 用于
	// usage 上报：filter
	// 命中时 service_tier 已经从 firstClientMessage 中删除，billing 应当
	// 反映上游实际处理的 tier（nil = default），而不是用户最初请求的
	// "priority"。HTTP 入口（line ~2728 extractOpenAIServiceTier(reqBody)）
	// 与 WS ingress（openai_ws_forwarder.go:2991 取自 payload）的语义一致。
	//
	// 多轮 passthrough：OpenAI Realtime / Responses WS 协议允许客户端在
	// 同一连接的不同 response.create 帧上发送不同 service_tier（参考
	// codex-rs/core/src/client.rs build_responses_request 每次重新填值）。
	// 因此使用 atomic.Pointer[string] 在 filter（runClientToUpstream
	// goroutine）和 OnTurnComplete / final result（runUpstreamToClient
	// goroutine）之间同步当前 turn 的 usage metadata。
	usageMeta.initFromFirstFrame(firstClientMessage, lockedPolicyModel)
	promptCacheKey := strings.TrimSpace(gjson.GetBytes(firstClientMessage, "prompt_cache_key").String())

	wsURL, err := s.buildOpenAIResponsesWSURL(account)
	if err != nil {
		return fmt.Errorf("build ws url: %w", err)
	}
	wsHost := "-"
	wsPath := "-"
	if parsedURL, parseErr := url.Parse(wsURL); parseErr == nil && parsedURL != nil {
		wsHost = normalizeOpenAIWSLogValue(parsedURL.Host)
		wsPath = normalizeOpenAIWSLogValue(parsedURL.Path)
	}
	logOpenAIWSV2Passthrough(
		"relay_dial_start account_id=%d ws_host=%s ws_path=%s proxy_enabled=%v",
		account.ID,
		wsHost,
		wsPath,
		account.ProxyID != nil && account.Proxy != nil,
	)

	isCodexCLI := false
	if c != nil {
		isCodexCLI = openai.IsCodexOfficialClientByHeaders(c.GetHeader("User-Agent"), c.GetHeader("originator"))
	}
	if s.cfg != nil && s.cfg.Gateway.ForceCodexCLI {
		isCodexCLI = true
	}
	turnState := ""
	turnMetadata := ""
	if c != nil {
		turnState = strings.TrimSpace(c.GetHeader(openAIWSTurnStateHeader))
		turnMetadata = strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader))
	}
	headers, _, buildHdrErr := s.buildOpenAIWSHeaders(ctx, c, account, token, wsDecision, isCodexCLI, turnState, turnMetadata, promptCacheKey)
	if buildHdrErr != nil {
		return fmt.Errorf("build ws headers: %w", buildHdrErr)
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	dialer := s.getOpenAIWSPassthroughDialer()
	if dialer == nil {
		return errors.New("openai ws passthrough dialer is nil")
	}

	agentTaskRecoveryTried := false
	var upstreamConn openAIWSClientConn
	statusCode := 0
	var handshakeHeaders http.Header
	for {
		headers, err = s.refreshOpenAIAgentIdentityHeaders(ctx, account, headers)
		if err != nil {
			return fmt.Errorf("refresh ws authentication headers: %w", err)
		}
		dialCtx, cancelDial := context.WithTimeout(ctx, s.openAIWSDialTimeout())
		upstreamConn, statusCode, handshakeHeaders, err = dialer.Dial(dialCtx, wsURL, headers, proxyURL)
		cancelDial()
		if err == nil {
			break
		}
		var handshakeErr *openAIWSHandshakeError
		responseBody := []byte(nil)
		if errors.As(err, &handshakeErr) && handshakeErr != nil {
			responseBody = handshakeErr.Body
		}
		dialErr := &openAIWSDialError{StatusCode: statusCode, ResponseHeaders: cloneHeader(handshakeHeaders), ResponseBody: responseBody, Err: err}
		if s.isAgentIdentityAccount(ctx, account) && isAgentIdentityTaskInvalidWSDialError(dialErr) && !agentTaskRecoveryTried {
			agentTaskRecoveryTried = true
			if recoveryErr := s.recoverAgentIdentityTask(ctx, account, account.GetCredential("task_id")); recoveryErr != nil {
				return fmt.Errorf("agent identity task recovery failed: %w", recoveryErr)
			}
			continue
		}
		logOpenAIWSV2Passthrough(
			"relay_dial_failed account_id=%d status_code=%d err=%s",
			account.ID,
			statusCode,
			truncateOpenAIWSLogValue(err.Error(), openAIWSLogValueMaxLen),
		)
		s.handleOpenAIWSDialTransientFailure(ctx, account, lockedPolicyModel, dialErr)
		if statusCode == http.StatusTooManyRequests {
			s.persistOpenAIWSRateLimitSignal(ctx, account, handshakeHeaders, nil, "rate_limit_exceeded", "rate_limit_error", strings.TrimSpace(err.Error()))
			return &UpstreamFailoverError{
				StatusCode:      http.StatusTooManyRequests,
				ResponseHeaders: cloneHeader(handshakeHeaders),
			}
		}
		return s.mapOpenAIWSPassthroughDialError(err, statusCode, handshakeHeaders)
	}
	defer func() {
		_ = upstreamConn.Close()
	}()
	logOpenAIWSV2Passthrough(
		"relay_dial_ok account_id=%d status_code=%d upstream_request_id=%s",
		account.ID,
		statusCode,
		openAIWSHeaderValueForLog(handshakeHeaders, "x-request-id"),
	)

	upstreamFrameConn, ok := upstreamConn.(openaiwsv2.FrameConn)
	if !ok {
		return errors.New("openai ws passthrough upstream connection does not support frame relay")
	}

	completedTurns := atomic.Int32{}
	cyberBlocked := atomic.Bool{}
	turnActive := atomic.Bool{}
	turnActive.Store(true)
	var turnLifecycleMu sync.Mutex
	completedResponseIDs := make(map[string]struct{})
	turnBeginWritePending := atomic.Bool{}
	unwrittenClientTurn := atomic.Bool{}
	policyClientConn := &openAIWSPolicyEnforcingFrameConn{
		inner: clientAttempt,
		// filter is called only by the client->upstream relay goroutine.
		// The three locked model identities above are immutable.
		filter: func(msgType coderws.MessageType, payload []byte) ([]byte, *OpenAIFastBlockedError, error) {
			if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
				return payload, nil, nil
			}
			eventType, validationErr := ValidateOpenAIWSClientFrameJSON(payload)
			if validationErr != nil {
				return payload, nil, NewOpenAIWSClientCloseError(
					coderws.StatusPolicyViolation,
					"invalid websocket request payload",
					validationErr,
				)
			}
			turnBeginLocked := false
			switch eventType {
			case "response.create":
				// Classify a frame that arrives while the previous turn is still
				// finishing as pipelined. Waiting for AfterTurn to release the
				// lifecycle lock must not turn it into a valid next turn.
				if turnActive.Load() {
					return payload, nil, NewOpenAIWSClientCloseError(
						coderws.StatusPolicyViolation,
						"previous websocket response is still active",
						nil,
					)
				}
				turnLifecycleMu.Lock()
				turnBeginLocked = true
				defer func() {
					if turnBeginLocked {
						turnLifecycleMu.Unlock()
					}
				}()
				if cyberBlocked.Load() {
					return payload, nil, NewOpenAIWSClientCloseError(
						coderws.StatusPolicyViolation,
						OpenAICyberSessionBlockedClientMessage,
						nil,
					)
				}
				if turnActive.Load() {
					return payload, nil, NewOpenAIWSClientCloseError(
						coderws.StatusPolicyViolation,
						"previous websocket response is still active",
						nil,
					)
				}
				normalizedPayload, modelErr := normalizeOpenAIWSPassthroughLockedModelFrame(
					payload,
					eventType,
					lockedOriginalModel,
					lockedForwardModel,
				)
				if modelErr != nil {
					return payload, nil, NewOpenAIWSClientCloseError(
						coderws.StatusPolicyViolation,
						modelErr.Error(),
						modelErr,
					)
				}
				payload = normalizedPayload
				if account.IsOpenAIOAuth() && isOpenAIResponsesLiteWebSocketPayload(payload) {
					litePayload, _, liteErr := normalizeOpenAIResponsesLiteToolsPayload(payload)
					if liteErr != nil {
						return payload, nil, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, liteErr.Error(), liteErr)
					}
					payload = litePayload
				}
				if !turnActive.CompareAndSwap(false, true) {
					return payload, nil, NewOpenAIWSClientCloseError(
						coderws.StatusPolicyViolation,
						"previous websocket response is still active",
						nil,
					)
				}
				unwrittenClientTurn.Store(true)
			case "session.update":
				normalizedPayload, modelErr := normalizeOpenAIWSPassthroughLockedModelFrame(
					payload,
					eventType,
					lockedOriginalModel,
					lockedForwardModel,
				)
				if modelErr != nil {
					return payload, nil, NewOpenAIWSClientCloseError(
						coderws.StatusPolicyViolation,
						modelErr.Error(),
						modelErr,
					)
				}
				payload = normalizedPayload
			}
			if eventType == "response.create" && hooks != nil && hooks.BeforeRequest != nil {
				turnNo := int(completedTurns.Load()) + 1
				if turnNo < 2 {
					turnNo = 2
				}
				if err := hooks.BeforeRequest(turnNo, payload, lockedOriginalModel); err != nil {
					return payload, nil, err
				}
			}
			if eventType == "response.create" && hooks != nil && hooks.BeforeTurn != nil {
				turnNo := int(completedTurns.Load()) + 1
				if turnNo < 2 {
					turnNo = 2
				}
				if err := hooks.BeforeTurn(turnNo); err != nil {
					return payload, nil, err
				}
			}
			requestModelForThisFrame := usageMeta.requestModelForFrame(payload)
			model := lockedPolicyModel
			if !IsImageGenerationIntentForPlatform(openAIResponsesEndpoint, requestModelForThisFrame, payload, account.Platform) {
				if updatedPayload, injected, injectErr := s.injectDefaultOpenAIServiceTier(ctx, c, account, model, payload); injectErr != nil {
					return payload, nil, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", injectErr)
				} else if injected {
					payload = updatedPayload
				}
			}
			out, blocked, policyErr := s.applyOpenAIFastPolicyToWSResponseCreate(ctx, account, model, payload)
			// 多轮 passthrough usage：仅在成功（non-block / non-err）
			// 的 response.create 帧上更新 usageMeta，使用
			// filter 处理后的 payload，与首帧 policy-after-extract 语义
			// 保持一致（参见上方 extractOpenAIServiceTierFromBody 注释）。
			//   - 非 response.create 帧（response.cancel /
			//     conversation.item.create / session.update 等）不携带
			//     per-response metadata，不应覆盖前一轮值。
			//   - blocked != nil：该帧不会发送上游，usage metadata 应保持
			//     上一轮值。
			//   - policyErr != nil：异常路径，保持上一轮值。
			//   - 不带 service_tier 的 response.create 会让
			//     extractOpenAIServiceTierFromBody 返回 nil；这里有意
			//     覆盖（Store(nil)），因为 OpenAI 上游对该帧实际不传
			//     service_tier 时按 default 处理，billing 应如实反映。
			if policyErr == nil && blocked == nil &&
				eventType == "response.create" {
				usageMeta.updateFromResponseCreate(out, model, requestModelForThisFrame)
				turnBeginWritePending.Store(true)
				turnBeginLocked = false
			}
			return out, blocked, policyErr
		},
		onBlock: func(blocked *OpenAIFastBlockedError) {
			MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
			// See note above on Conn.Write being synchronous w.r.t. flush;
			// no explicit flush is required to ensure the error event lands
			// before the close frame.
			eventBytes := buildOpenAIFastPolicyBlockedWSEvent(blocked)
			if eventBytes == nil {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, s.openAIWSWriteTimeout())
			_ = clientAttempt.WriteFrame(writeCtx, coderws.MessageText, eventBytes)
			cancel()
		},
	}
	upstreamFirstMessageSent := false
	firstWriteCtx, cancelFirstWrite := context.WithTimeout(ctx, s.openAIWSWriteTimeout())
	firstWriteErr := upstreamFrameConn.WriteFrame(firstWriteCtx, coderws.MessageText, firstClientMessage)
	cancelFirstWrite()
	if firstWriteErr != nil {
		return wrapOpenAIWSIngressTurnError(
			"write_upstream",
			fmt.Errorf("write first upstream websocket request: %w", firstWriteErr),
			false,
		)
	}
	upstreamFirstMessageSent = true

	readNextClientFrame := func(readCtx context.Context, conn openaiwsv2.FrameConn) (coderws.MessageType, []byte, error) {
		for {
			msgType, payload, readErr := conn.ReadFrame(readCtx)
			if readErr != nil {
				return msgType, payload, readErr
			}
			if (msgType == coderws.MessageText || msgType == coderws.MessageBinary) &&
				strings.TrimSpace(gjson.GetBytes(payload, "type").String()) == "response.create" {
				return msgType, payload, nil
			}
			if writeErr := upstreamFrameConn.WriteFrame(readCtx, msgType, payload); writeErr != nil {
				return msgType, payload, writeErr
			}
		}
	}

	relayResult, relayExit := openaiwsv2.RunEntry(openaiwsv2.EntryInput{
		Ctx:                ctx,
		ClientConn:         policyClientConn,
		UpstreamConn:       upstreamFrameConn,
		FirstClientMessage: firstClientMessage,
		Options: openaiwsv2.RelayOptions{
			WriteTimeout:                    s.openAIWSWriteTimeout(),
			IdleTimeout:                     s.openAIWSPassthroughIdleTimeout(),
			FirstMessageType:                coderws.MessageText,
			FirstMessageSent:                upstreamFirstMessageSent,
			StartClientAfterFirstDownstream: false,
			ReadClientFrame:                 readNextClientFrame,
			AfterWriteUpstream: func(_ coderws.MessageType, _ []byte, writeErr error) {
				if !turnBeginWritePending.CompareAndSwap(true, false) {
					return
				}
				if writeErr == nil {
					unwrittenClientTurn.Store(false)
				}
				turnLifecycleMu.Unlock()
			},
			HasUnwrittenClientTurn: func() bool {
				return unwrittenClientTurn.Load()
			},
			OnUsageParseFailure: func(eventType string, usageRaw string) {
				logOpenAIWSV2Passthrough(
					"usage_parse_failed event_type=%s usage_raw=%s",
					truncateOpenAIWSLogValue(eventType, openAIWSLogValueMaxLen),
					truncateOpenAIWSLogValue(usageRaw, openAIWSLogValueMaxLen),
				)
			},
			OnTurnComplete: func(turn openaiwsv2.RelayTurnResult) {
				turnLifecycleMu.Lock()
				defer turnLifecycleMu.Unlock()
				responseID := strings.TrimSpace(turn.RequestID)
				if responseID != "" {
					if _, completed := completedResponseIDs[responseID]; completed {
						return
					}
					completedResponseIDs[responseID] = struct{}{}
				}
				if !turnActive.Load() {
					return
				}
				turnNo := int(completedTurns.Add(1))
				turnResult := &OpenAIForwardResult{
					RequestID: turn.RequestID,
					Usage: OpenAIUsage{
						InputTokens:              turn.Usage.InputTokens,
						OutputTokens:             turn.Usage.OutputTokens,
						CacheCreationInputTokens: turn.Usage.CacheCreationInputTokens,
						CacheReadInputTokens:     turn.Usage.CacheReadInputTokens,
						ImageOutputTokens:        turn.Usage.ImageOutputTokens,
					},
					Model:                 turn.RequestModel,
					ServiceTier:           usageMeta.serviceTier.Load(),
					ReasoningEffort:       usageMeta.reasoningEffort.Load(),
					Stream:                true,
					OpenAIWSMode:          true,
					UpstreamTerminalEvent: normalizeOpenAIWSTerminalEvent(turn.TerminalEventType),
					ResponseHeaders:       cloneHeader(handshakeHeaders),
					Duration:              turn.Duration,
					FirstTokenMs:          turn.FirstTokenMs,
				}
				logOpenAIWSV2Passthrough(
					"relay_turn_completed account_id=%d turn=%d request_id=%s terminal_event=%s duration_ms=%d first_token_ms=%d input_tokens=%d output_tokens=%d cache_read_tokens=%d",
					account.ID,
					turnNo,
					truncateOpenAIWSLogValue(turnResult.RequestID, openAIWSIDValueMaxLen),
					truncateOpenAIWSLogValue(turn.TerminalEventType, openAIWSLogValueMaxLen),
					turnResult.Duration.Milliseconds(),
					openAIWSFirstTokenMsForLog(turnResult.FirstTokenMs),
					turnResult.Usage.InputTokens,
					turnResult.Usage.OutputTokens,
					turnResult.Usage.CacheReadInputTokens,
				)
				if hooks != nil && hooks.AfterTurn != nil {
					hooks.AfterTurn(turnNo, turnResult, nil)
				}
				turnActive.Store(false)
			},
			BeforeWriteClient: func(msgType coderws.MessageType, payload []byte, wroteDownstream bool) error {
				if msgType != coderws.MessageText {
					return nil
				}
				eventType, _, _ := parseOpenAIWSEventEnvelope(payload)
				cyberHit := observeOpenAIWSCyberPolicy(c, payload, http.StatusOK, nil)
				if cyberHit {
					cyberBlocked.Store(true)
				}
				if isOpenAIWSTerminalEvent(eventType) {
					if !cyberHit {
						s.handleOpenAIWSTerminalTransientFailure(ctx, account, lockedPolicyModel, handshakeHeaders, payload)
					}
				}
				if eventType == "error" {
					if !cyberHit {
						s.handleOpenAIWSErrorEventTransientFailure(ctx, account, lockedPolicyModel, handshakeHeaders, payload)
					}
				}
				if wroteDownstream || eventType != "error" {
					return nil
				}
				errCodeRaw, errTypeRaw, errMsgRaw := parseOpenAIWSErrorEventFields(payload)
				if !isOpenAIWSRateLimitError(errCodeRaw, errTypeRaw, errMsgRaw) {
					return nil
				}
				s.persistOpenAIWSRateLimitSignal(ctx, account, handshakeHeaders, payload, errCodeRaw, errTypeRaw, errMsgRaw)
				logOpenAIWSV2Passthrough(
					"relay_rate_limit_failover account_id=%d err_code=%s err_type=%s err_message=%s",
					account.ID,
					truncateOpenAIWSLogValue(errCodeRaw, openAIWSLogValueMaxLen),
					truncateOpenAIWSLogValue(errTypeRaw, openAIWSLogValueMaxLen),
					truncateOpenAIWSLogValue(errMsgRaw, openAIWSLogValueMaxLen),
				)
				return &UpstreamFailoverError{
					StatusCode:      http.StatusTooManyRequests,
					ResponseBody:    append([]byte(nil), payload...),
					ResponseHeaders: cloneHeader(handshakeHeaders),
				}
			},
			OnTrace: func(event openaiwsv2.RelayTraceEvent) {
				logOpenAIWSV2Passthrough(
					"relay_trace account_id=%d stage=%s direction=%s msg_type=%s bytes=%d graceful=%v wrote_downstream=%v err=%s",
					account.ID,
					truncateOpenAIWSLogValue(event.Stage, openAIWSLogValueMaxLen),
					truncateOpenAIWSLogValue(event.Direction, openAIWSLogValueMaxLen),
					truncateOpenAIWSLogValue(event.MessageType, openAIWSLogValueMaxLen),
					event.PayloadBytes,
					event.Graceful,
					event.WroteDownstream,
					truncateOpenAIWSLogValue(event.Error, openAIWSLogValueMaxLen),
				)
			},
		},
	})

	result := &OpenAIForwardResult{
		RequestID: relayResult.RequestID,
		Usage: OpenAIUsage{
			InputTokens:              relayResult.Usage.InputTokens,
			OutputTokens:             relayResult.Usage.OutputTokens,
			CacheCreationInputTokens: relayResult.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     relayResult.Usage.CacheReadInputTokens,
			ImageOutputTokens:        relayResult.Usage.ImageOutputTokens,
		},
		Model:                 relayResult.RequestModel,
		ServiceTier:           usageMeta.serviceTier.Load(),
		ReasoningEffort:       usageMeta.reasoningEffort.Load(),
		Stream:                true,
		OpenAIWSMode:          true,
		UpstreamTerminalEvent: normalizeOpenAIWSTerminalEvent(relayResult.TerminalEventType),
		ResponseHeaders:       cloneHeader(handshakeHeaders),
		Duration:              relayResult.Duration,
		FirstTokenMs:          relayResult.FirstTokenMs,
	}

	turnCount := int(completedTurns.Load())
	if relayExit == nil {
		logOpenAIWSV2Passthrough(
			"relay_completed account_id=%d request_id=%s terminal_event=%s exit_source=%s duration_ms=%d c2u_frames=%d u2c_frames=%d dropped_frames=%d turns=%d",
			account.ID,
			truncateOpenAIWSLogValue(result.RequestID, openAIWSIDValueMaxLen),
			truncateOpenAIWSLogValue(relayResult.TerminalEventType, openAIWSLogValueMaxLen),
			relayResult.ExitSource,
			result.Duration.Milliseconds(),
			relayResult.ClientToUpstreamFrames,
			relayResult.UpstreamToClientFrames,
			relayResult.DroppedDownstreamFrames,
			turnCount,
		)
		if relayResult.ExitSource == openaiwsv2.RelayExitSourceClientDisconnect {
			result.ClientDisconnect = true
			turnLifecycleMu.Lock()
			if turnActive.Load() {
				if hooks != nil && hooks.AfterTurn != nil {
					hooks.AfterTurn(int(completedTurns.Load())+1, result, nil)
				}
				turnActive.Store(false)
			}
			turnLifecycleMu.Unlock()
			return nil
		}
		if relayResult.ExitSource != openaiwsv2.RelayExitSourceTerminal {
			turnErr := wrapOpenAIWSIngressTurnError(
				"relay_exit",
				errors.New("websocket relay ended without a terminal event"),
				false,
			)
			turnLifecycleMu.Lock()
			if turnActive.Load() {
				if hooks != nil && hooks.AfterTurn != nil {
					hooks.AfterTurn(int(completedTurns.Load())+1, nil, turnErr)
				}
				turnActive.Store(false)
			}
			turnLifecycleMu.Unlock()
			return turnErr
		}
		// RunEntry uses a bounded goroutine drain. The active check makes the
		// terminal callback and the response-id-less terminal fallback single-winner.
		turnLifecycleMu.Lock()
		if turnActive.Load() {
			if hooks != nil && hooks.AfterTurn != nil {
				hooks.AfterTurn(int(completedTurns.Load())+1, result, nil)
			}
			turnActive.Store(false)
		}
		turnLifecycleMu.Unlock()
		return nil
	}
	logOpenAIWSV2Passthrough(
		"relay_failed account_id=%d stage=%s wrote_downstream=%v err=%s duration_ms=%d c2u_frames=%d u2c_frames=%d dropped_frames=%d turns=%d",
		account.ID,
		truncateOpenAIWSLogValue(relayExit.Stage, openAIWSLogValueMaxLen),
		relayExit.WroteDownstream,
		truncateOpenAIWSLogValue(relayErrorText(relayExit.Err), openAIWSLogValueMaxLen),
		result.Duration.Milliseconds(),
		relayResult.ClientToUpstreamFrames,
		relayResult.UpstreamToClientFrames,
		relayResult.DroppedDownstreamFrames,
		turnCount,
	)

	relayErr := relayExit.Err
	if relayExit.Stage == "idle_timeout" {
		relayErr = NewOpenAIWSClientCloseError(
			coderws.StatusPolicyViolation,
			"client websocket idle timeout",
			relayErr,
		)
	}
	turnErr := wrapOpenAIWSIngressTurnError(
		relayExit.Stage,
		relayErr,
		relayExit.WroteDownstream,
	)
	turnLifecycleMu.Lock()
	if turnActive.Load() {
		if hooks != nil && hooks.AfterTurn != nil {
			hooks.AfterTurn(int(completedTurns.Load())+1, nil, turnErr)
		}
		turnActive.Store(false)
	}
	turnLifecycleMu.Unlock()
	return turnErr
}

func (s *OpenAIGatewayService) mapOpenAIWSPassthroughDialError(
	err error,
	statusCode int,
	handshakeHeaders http.Header,
) error {
	if err == nil {
		return nil
	}
	wrappedErr := err
	var dialErr *openAIWSDialError
	if !errors.As(err, &dialErr) {
		var handshakeErr *openAIWSHandshakeError
		var responseBody []byte
		if errors.As(err, &handshakeErr) && handshakeErr != nil {
			responseBody = append([]byte(nil), handshakeErr.Body...)
		}
		wrappedErr = &openAIWSDialError{
			StatusCode:      statusCode,
			ResponseHeaders: cloneHeader(handshakeHeaders),
			ResponseBody:    responseBody,
			Err:             err,
		}
	}

	if errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return NewOpenAIWSClientCloseError(
			coderws.StatusTryAgainLater,
			"upstream websocket connect timeout",
			wrappedErr,
		)
	}
	if statusCode == http.StatusTooManyRequests {
		return NewOpenAIWSClientCloseError(
			coderws.StatusTryAgainLater,
			"upstream websocket is busy, please retry later",
			wrappedErr,
		)
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return NewOpenAIWSClientCloseError(
			coderws.StatusPolicyViolation,
			"upstream websocket authentication failed",
			wrappedErr,
		)
	}
	if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		return NewOpenAIWSClientCloseError(
			coderws.StatusPolicyViolation,
			"upstream websocket handshake rejected",
			wrappedErr,
		)
	}
	return fmt.Errorf("openai ws passthrough dial: %w", wrappedErr)
}

func openaiwsv2RelayMessageTypeName(msgType coderws.MessageType) string {
	switch msgType {
	case coderws.MessageText:
		return "text"
	case coderws.MessageBinary:
		return "binary"
	default:
		return fmt.Sprintf("unknown(%d)", msgType)
	}
}

func relayErrorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func openAIWSFirstTokenMsForLog(firstTokenMs *int) int {
	if firstTokenMs == nil {
		return -1
	}
	return *firstTokenMs
}

func logOpenAIWSV2Passthrough(format string, args ...any) {
	logger.LegacyPrintf(
		"service.openai_ws_v2",
		"[OpenAI WS v2 passthrough] %s "+format,
		append([]any{openaiWSV2PassthroughModeFields}, args...)...,
	)
}
