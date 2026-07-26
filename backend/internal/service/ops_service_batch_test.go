package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpsServiceRecordErrorBatch_SanitizesAndBatches(t *testing.T) {
	t.Parallel()

	var captured []*OpsInsertErrorLogInput
	repo := &opsRepoMock{
		BatchInsertErrorLogsFn: func(ctx context.Context, inputs []*OpsInsertErrorLogInput) (int64, error) {
			captured = append(captured, inputs...)
			return int64(len(inputs)), nil
		},
	}
	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	msg := " upstream failed: https://example.com?access_token=secret-value "
	detail := `{"authorization":"Bearer secret-token"}`
	entries := []*OpsInsertErrorLogInput{
		{
			ErrorBody:            `{"error":"bad","access_token":"secret"}`,
			UpstreamStatusCode:   intPtr(-10),
			UpstreamErrorMessage: strPtr(msg),
			UpstreamErrorDetail:  strPtr(detail),
			UpstreamErrors: []*OpsUpstreamErrorEvent{
				{
					AccountID:          -2,
					UpstreamStatusCode: 429,
					Message:            " token leaked ",
					Detail:             `{"refresh_token":"secret"}`,
				},
			},
		},
		{
			ErrorPhase: "upstream",
			ErrorType:  "upstream_error",
			CreatedAt:  time.Now().UTC(),
		},
	}

	require.NoError(t, svc.RecordErrorBatch(context.Background(), entries))
	require.Len(t, captured, 2)

	first := captured[0]
	require.Equal(t, "internal", first.ErrorPhase)
	require.Equal(t, "api_error", first.ErrorType)
	require.Nil(t, first.UpstreamStatusCode)
	require.NotNil(t, first.UpstreamErrorMessage)
	require.NotContains(t, *first.UpstreamErrorMessage, "secret-value")
	require.Contains(t, *first.UpstreamErrorMessage, "access_token=***")
	require.NotNil(t, first.UpstreamErrorDetail)
	require.NotContains(t, *first.UpstreamErrorDetail, "secret-token")
	require.NotContains(t, first.ErrorBody, "secret")
	require.Nil(t, first.UpstreamErrors)
	require.NotNil(t, first.UpstreamErrorsJSON)
	require.NotContains(t, *first.UpstreamErrorsJSON, "secret")
	require.Contains(t, *first.UpstreamErrorsJSON, "[REDACTED]")

	second := captured[1]
	require.Equal(t, "upstream", second.ErrorPhase)
	require.Equal(t, "upstream_error", second.ErrorType)
	require.False(t, second.CreatedAt.IsZero())
}

func TestOpsServiceRecordErrorBatch_FallsBackToSingleInsert(t *testing.T) {
	t.Parallel()

	var (
		batchCalls  int
		singleCalls int
	)
	repo := &opsRepoMock{
		BatchInsertErrorLogsFn: func(ctx context.Context, inputs []*OpsInsertErrorLogInput) (int64, error) {
			batchCalls++
			return 0, errors.New("batch failed")
		},
		InsertErrorLogFn: func(ctx context.Context, input *OpsInsertErrorLogInput) (int64, error) {
			singleCalls++
			return int64(singleCalls), nil
		},
	}
	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	err := svc.RecordErrorBatch(context.Background(), []*OpsInsertErrorLogInput{
		{ErrorMessage: "first"},
		{ErrorMessage: "second"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, batchCalls)
	require.Equal(t, 2, singleCalls)
}

func TestOpsServiceRecordErrorPersistsExplicitAccountAuthStatusZero(t *testing.T) {
	t.Parallel()

	var captured *OpsInsertErrorLogInput
	repo := &opsRepoMock{
		InsertErrorLogFn: func(_ context.Context, input *OpsInsertErrorLogInput) (int64, error) {
			captured = input
			return 1, nil
		},
	}
	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	staleStatus := 403
	staleMessage := "stale inference message"
	staleDetail := "stale inference detail"

	err := svc.RecordError(context.Background(), &OpsInsertErrorLogInput{
		ErrorPhase:           "upstream",
		ErrorType:            "upstream_error",
		ErrorOwner:           "provider",
		ErrorSource:          "upstream_http",
		UpstreamStatusCode:   &staleStatus,
		UpstreamErrorMessage: &staleMessage,
		UpstreamErrorDetail:  &staleDetail,
		UpstreamErrors: []*OpsUpstreamErrorEvent{
			{Stage: string(GatewayFailureStageInference), UpstreamStatusCode: 403, Message: staleMessage, Detail: staleDetail},
			{
				Stage: string(GatewayFailureStageAccountAuth), Scope: string(GatewayFailureScopeAccount),
				Reason: string(GrokCredentialReasonRevoked), Message: "Grok OAuth credentials require account action",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, captured)
	require.Equal(t, "account_auth", captured.ErrorPhase)
	require.Equal(t, "provider", captured.ErrorOwner)
	require.Equal(t, "gateway", captured.ErrorSource)
	require.NotNil(t, captured.UpstreamStatusCode)
	require.Zero(t, *captured.UpstreamStatusCode)
	require.NotNil(t, captured.UpstreamErrorMessage)
	require.Equal(t, "Grok OAuth credentials require account action", *captured.UpstreamErrorMessage)
	require.Nil(t, captured.UpstreamErrorDetail)
	require.Nil(t, captured.UpstreamErrors)
	require.NotNil(t, captured.UpstreamErrorsJSON)
	require.Contains(t, *captured.UpstreamErrorsJSON, `"upstream_status_code":403`)
	require.Contains(t, *captured.UpstreamErrorsJSON, `"stage":"account_auth"`)
}

func TestOpsServiceRecordErrorPreservesMetadataOnlyUpstreamEvent(t *testing.T) {
	t.Parallel()

	var captured *OpsInsertErrorLogInput
	repo := &opsRepoMock{
		InsertErrorLogFn: func(_ context.Context, input *OpsInsertErrorLogInput) (int64, error) {
			captured = input
			return 1, nil
		},
	}
	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	err := svc.RecordError(context.Background(), &OpsInsertErrorLogInput{
		ErrorPhase:   string(GatewayFailureStageAccountAuth),
		ErrorType:    "upstream_error",
		ErrorMessage: "Web chat upstream request failed",
		UpstreamErrors: []*OpsUpstreamErrorEvent{
			{
				Passthrough:          true,
				Platform:             "openai",
				AccountID:            91,
				AccountName:          "fallback-account",
				UpstreamRequestID:    "upstream-request-metadata-only",
				UpstreamURL:          "https://api.openai.com/v1/responses",
				Kind:                 "failover",
				Stage:                string(GatewayFailureStageAccountAuth),
				Scope:                string(GatewayFailureScopeAccount),
				Reason:               string(GrokCredentialReasonRevoked),
				Message:              "",
				Detail:               "",
				UpstreamResponseBody: "",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, captured)
	require.Nil(t, captured.UpstreamErrors)
	require.NotNil(t, captured.UpstreamErrorsJSON)

	var stored []*OpsUpstreamErrorEvent
	require.NoError(t, json.Unmarshal([]byte(*captured.UpstreamErrorsJSON), &stored))
	require.Len(t, stored, 1)
	require.True(t, stored[0].Passthrough)
	require.Equal(t, "openai", stored[0].Platform)
	require.Equal(t, int64(91), stored[0].AccountID)
	require.Equal(t, "fallback-account", stored[0].AccountName)
	require.Equal(t, "upstream-request-metadata-only", stored[0].UpstreamRequestID)
	require.Equal(t, "https://api.openai.com/v1/responses", stored[0].UpstreamURL)
	require.Equal(t, "failover", stored[0].Kind)
	require.Equal(t, string(GatewayFailureStageAccountAuth), stored[0].Stage)
	require.Equal(t, string(GatewayFailureScopeAccount), stored[0].Scope)
	require.Equal(t, string(GrokCredentialReasonRevoked), stored[0].Reason)
	require.Empty(t, stored[0].Message)
	require.Empty(t, stored[0].Detail)
	require.Empty(t, stored[0].UpstreamResponseBody)
}

func strPtr(v string) *string {
	return &v
}
