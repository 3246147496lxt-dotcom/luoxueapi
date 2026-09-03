package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type chatHistoryRepositoryStub struct {
	createInput     *CreateChatHistoryConversationInput
	listCursor      string
	listLimit       int
	listSearch      string
	syncVersion     int64
	syncLimit       int
	deleteID        string
	deleteRevision  int64
	prepareInput    *PrepareChatCompletionInput
	checkpointInput *CheckpointChatCompletionInput
	finalizeInput   *FinalizeChatCompletionInput
	stopUserID      int64
	stopAttemptID   string
}

func (s *chatHistoryRepositoryStub) CreateConversation(
	_ context.Context,
	_ int64,
	input *CreateChatHistoryConversationInput,
) (*ChatHistoryConversation, error) {
	s.createInput = input
	return &ChatHistoryConversation{ID: input.ID}, nil
}

func (s *chatHistoryRepositoryStub) ListConversations(
	_ context.Context,
	_ int64,
	cursor string,
	limit int,
	search string,
) (*ChatHistoryConversationList, error) {
	s.listCursor = cursor
	s.listLimit = limit
	s.listSearch = search
	return &ChatHistoryConversationList{}, nil
}

func (*chatHistoryRepositoryStub) GetConversation(context.Context, int64, string) (*ChatHistoryConversation, error) {
	return &ChatHistoryConversation{}, nil
}

func (*chatHistoryRepositoryStub) ListMessages(context.Context, int64, string, *int64, int) (*ChatHistoryMessagePage, error) {
	return &ChatHistoryMessagePage{}, nil
}

func (*chatHistoryRepositoryStub) UpdateConversation(context.Context, int64, *UpdateChatHistoryConversationInput) (*ChatHistoryConversation, error) {
	return &ChatHistoryConversation{}, nil
}

func (s *chatHistoryRepositoryStub) DeleteConversation(
	_ context.Context,
	_ int64,
	id string,
	revision int64,
) error {
	s.deleteID = id
	s.deleteRevision = revision
	return nil
}

func (s *chatHistoryRepositoryStub) Sync(
	_ context.Context,
	_, afterVersion int64,
	limit int,
) (*ChatHistorySyncPage, error) {
	s.syncVersion = afterVersion
	s.syncLimit = limit
	return &ChatHistorySyncPage{}, nil
}

func (*chatHistoryRepositoryStub) GetAttempt(context.Context, int64, string) (*ChatHistoryAttempt, error) {
	return &ChatHistoryAttempt{}, nil
}

func (s *chatHistoryRepositoryStub) PrepareCompletion(
	_ context.Context,
	_ int64,
	input *PrepareChatCompletionInput,
) (*PreparedChatCompletion, error) {
	s.prepareInput = input
	return &PreparedChatCompletion{
		Claimed:            true,
		ClientRequestID:    input.ClientRequestID,
		AttemptStatus:      ChatAttemptStatusProcessing,
		ConversationID:     input.ConversationID,
		AssistantMessageID: input.AssistantMessageID,
	}, nil
}

func (s *chatHistoryRepositoryStub) FinalizeCompletion(
	_ context.Context,
	_ int64,
	input *FinalizeChatCompletionInput,
) error {
	s.finalizeInput = input
	return nil
}

func (s *chatHistoryRepositoryStub) CheckpointCompletion(
	_ context.Context,
	_ int64,
	input *CheckpointChatCompletionInput,
) error {
	s.checkpointInput = input
	return nil
}

func (s *chatHistoryRepositoryStub) StopCompletion(
	_ context.Context,
	userID int64,
	attemptID string,
) (*StopChatCompletionResult, error) {
	s.stopUserID = userID
	s.stopAttemptID = attemptID
	now := time.Now().UTC()
	return &StopChatCompletionResult{
		AttemptID:      attemptID,
		Accepted:       true,
		AttemptStatus:  ChatAttemptStatusInterrupted,
		DeliveryStatus: ChatMessageDeliveryStopped,
		StoppedAt:      &now,
	}, nil
}

func TestChatHistoryCreateNormalizesLegacyImportAndHashesCanonicalContent(t *testing.T) {
	t.Parallel()
	repo := &chatHistoryRepositoryStub{}
	svc := NewChatHistoryService(repo)

	_, err := svc.CreateConversation(context.Background(), 42, &CreateChatHistoryConversationInput{
		ID:       "conversation-12345678",
		Title:    " Legacy ",
		Model:    " gpt-5.5 ",
		Imported: true,
		ImportedMessages: []ChatHistoryImportedMessage{
			{
				ID:        "message-12345678",
				Role:      "user",
				Content:   "hello",
				Status:    "complete",
				CreatedAt: time.UnixMilli(1_700_000_000_000),
			},
			{
				ID:        "message-87654321",
				Role:      "assistant",
				Content:   "partial",
				Status:    "stopped",
				CreatedAt: time.UnixMilli(1_700_000_001_000),
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "Legacy", repo.createInput.Title)
	require.Equal(t, "gpt-5.5", repo.createInput.Model)
	require.Equal(t, ChatMessageDeliveryCompleted, repo.createInput.ImportedMessages[0].Status)
	require.Equal(t, ChatMessageDeliveryStopped, repo.createInput.ImportedMessages[1].Status)
	require.Len(t, repo.createInput.CreateHash, 64)
}

func TestChatHistoryStopCompletionValidatesAndScopesIntentToUser(t *testing.T) {
	t.Parallel()
	repo := &chatHistoryRepositoryStub{}
	svc := NewChatHistoryService(repo)

	result, err := svc.StopCompletion(context.Background(), 42, " attempt-12345678 ")
	require.NoError(t, err)
	require.True(t, result.Accepted)
	require.Equal(t, int64(42), repo.stopUserID)
	require.Equal(t, "attempt-12345678", repo.stopAttemptID)

	_, err = svc.StopCompletion(context.Background(), 42, "bad")
	require.ErrorIs(t, err, ErrChatAttemptIDInvalid)
}

func TestTerminalizeChatMessageActivitiesPreservesPartialSummary(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	activities := TerminalizeChatMessageActivities([]ChatMessageActivity{{
		Source:       ChatMessageActivitySourceOpenAIResponses,
		ActivityType: ChatMessageActivityTypeReasoningSummary,
		ItemID:       "rs_1",
		Status:       ChatMessageActivityStatusInProgress,
		Text:         "kept partial summary",
		Metadata:     []byte(`{"provider":"openai"}`),
	}}, ChatMessageActivityStatusStopped, "user_stopped", now)

	require.Len(t, activities, 1)
	require.Equal(t, "kept partial summary", activities[0].Text)
	require.Equal(t, ChatMessageActivityStatusStopped, activities[0].Status)
	require.Equal(t, now, *activities[0].CompletedAt)
	require.JSONEq(t, `{"provider":"openai","last_event":"user_stopped"}`, string(activities[0].Metadata))
}

func TestChatHistoryCreateRejectsEmptyImportedContent(t *testing.T) {
	t.Parallel()
	svc := NewChatHistoryService(&chatHistoryRepositoryStub{})

	_, err := svc.CreateConversation(context.Background(), 42, &CreateChatHistoryConversationInput{
		ID:       "conversation-12345678",
		Title:    "Legacy",
		Model:    "gpt-5.5",
		Imported: true,
		ImportedMessages: []ChatHistoryImportedMessage{{
			ID:      "message-12345678",
			Role:    "user",
			Content: "   ",
			Status:  "completed",
		}},
	})

	require.ErrorIs(t, err, ErrChatHistoryInvalid)
}

func TestChatHistoryListUsesOpaqueCursorContractAndClampsLimit(t *testing.T) {
	t.Parallel()
	repo := &chatHistoryRepositoryStub{}
	svc := NewChatHistoryService(repo)

	_, err := svc.ListConversations(context.Background(), 42, " opaque ", 999, " query ")

	require.NoError(t, err)
	require.Equal(t, "opaque", repo.listCursor)
	require.Equal(t, maxChatHistoryPageSize, repo.listLimit)
	require.Equal(t, "query", repo.listSearch)
}

func TestChatHistorySyncParsesStringVersionCursor(t *testing.T) {
	t.Parallel()
	repo := &chatHistoryRepositoryStub{}
	svc := NewChatHistoryService(repo)

	_, err := svc.Sync(context.Background(), 42, "17", 0)
	require.NoError(t, err)
	require.Equal(t, int64(17), repo.syncVersion)
	require.Equal(t, 100, repo.syncLimit)

	_, err = svc.Sync(context.Background(), 42, "not-a-version", 20)
	require.ErrorIs(t, err, ErrChatHistoryInvalid)
}

func TestChatHistoryPrepareCompletionCanonicalizesEnvelopeAndHashesIt(t *testing.T) {
	t.Parallel()
	repo := &chatHistoryRepositoryStub{}
	svc := NewChatHistoryService(repo)
	expectedHead := " message-head-12345678 "

	result, err := svc.PrepareCompletion(
		context.Background(),
		42,
		" attempt-12345678 ",
		" client-request-12345678 ",
		&PrepareChatCompletionInput{
			ConversationID:        " conversation-12345678 ",
			Model:                 " gpt-5.5 ",
			ReasoningMode:         " PRO ",
			ReasoningEffort:       " high ",
			ExpectedHeadMessageID: &expectedHead,
			UserMessage: &ChatCompletionHistoryUserMessage{
				ID:      " message-user-12345678 ",
				Content: "  preserve prompt spacing  ",
			},
			AssistantMessageID: " message-assistant-12345678 ",
		},
	)

	require.NoError(t, err)
	require.True(t, result.Claimed)
	require.Equal(t, "conversation-12345678", repo.prepareInput.ConversationID)
	require.Equal(t, "pro", repo.prepareInput.ReasoningMode)
	require.Equal(t, "high", repo.prepareInput.ReasoningEffort)
	require.Equal(t, "message-head-12345678", *repo.prepareInput.ExpectedHeadMessageID)
	require.Equal(t, "  preserve prompt spacing  ", repo.prepareInput.UserMessage.Content)
	require.Equal(t, maxChatCompletionContextMessages, repo.prepareInput.ContextMessageLimit)
	require.Len(t, repo.prepareInput.RequestHash, 64)
}

func TestChatHistoryPrepareCompletionHashesReasoningMode(t *testing.T) {
	t.Parallel()
	hashForMode := func(mode string) string {
		repo := &chatHistoryRepositoryStub{}
		svc := NewChatHistoryService(repo)
		_, err := svc.PrepareCompletion(
			context.Background(),
			42,
			"attempt-12345678",
			"client-request-12345678",
			&PrepareChatCompletionInput{
				ConversationID:  "conversation-12345678",
				Model:           "gpt-5.6-sol",
				ReasoningMode:   mode,
				ReasoningEffort: "medium",
				UserMessage: &ChatCompletionHistoryUserMessage{
					ID:      "message-user-12345678",
					Content: "same prompt",
				},
				AssistantMessageID: "message-assistant-12345678",
			},
		)
		require.NoError(t, err)
		return repo.prepareInput.RequestHash
	}

	standardHash := hashForMode(" standard ")
	proHash := hashForMode("pro")
	require.NotEqual(t, standardHash, proHash, "standard and Pro are distinct idempotent requests")
	require.Equal(t, proHash, hashForMode(" PRO "), "mode normalization must be stable")
}

func TestChatHistoryPrepareCompletionAllowsAttachmentOnlyAndHashesOnlyOrderedIdentity(t *testing.T) {
	expires := time.Now().Add(24 * time.Hour)
	attachmentRepo := &chatAttachmentRepoFake{attachments: map[string]ChatAttachment{
		"att_12345678": {ID: "att_12345678", Name: "first.png", Kind: ChatAttachmentKindImage, Size: 10, Status: ChatAttachmentStatusReady, ExpiresAt: expires, Digest: strings.Repeat("a", 64)},
	}}
	prepare := func(repo *chatHistoryRepositoryStub, attempt, assistant string) string {
		svc := NewChatHistoryService(repo)
		svc.attachments = attachmentRepo
		_, err := svc.PrepareCompletion(context.Background(), 42, attempt, "client-request-12345678", &PrepareChatCompletionInput{
			ConversationID: "conversation-12345678", Model: "gpt-5.5",
			UserMessage:        &ChatCompletionHistoryUserMessage{ID: "message-user-12345678", AttachmentIDs: []string{"att_12345678"}},
			AssistantMessageID: assistant,
		})
		require.NoError(t, err)
		return repo.prepareInput.RequestHash
	}
	firstRepo := &chatHistoryRepositoryStub{}
	firstHash := prepare(firstRepo, "attempt-12345678", "message-assistant-12345678")
	a := attachmentRepo.attachments["att_12345678"]
	a.Name = "renamed.png"
	a.ExpiresAt = expires.Add(time.Hour)
	attachmentRepo.attachments[a.ID] = a
	secondRepo := &chatHistoryRepositoryStub{}
	secondHash := prepare(secondRepo, "attempt-87654321", "message-assistant-12345678")
	require.Equal(t, firstHash, secondHash)
	require.Empty(t, firstRepo.prepareInput.UserMessage.Content)
	require.Len(t, firstRepo.prepareInput.UserMessage.Attachments, 1)
}

func TestChatHistoryCompletionWritesRequireMonotonicSequence(t *testing.T) {
	t.Parallel()
	svc := NewChatHistoryService(&chatHistoryRepositoryStub{})

	err := svc.CheckpointCompletion(
		context.Background(),
		42,
		&CheckpointChatCompletionInput{
			AttemptID:          "attempt-12345678",
			AssistantMessageID: "message-assistant-12345678",
			CheckpointSeq:      0,
		},
	)
	require.ErrorIs(t, err, ErrChatHistoryInvalid)

	err = svc.FinalizeCompletion(
		context.Background(),
		42,
		&FinalizeChatCompletionInput{
			AttemptID:          "attempt-12345678",
			AssistantMessageID: "message-assistant-12345678",
			CheckpointSeq:      0,
			DeliveryStatus:     ChatMessageDeliveryCompleted,
			AttemptStatus:      ChatAttemptStatusCompleted,
		},
	)
	require.ErrorIs(t, err, ErrChatHistoryInvalid)
}

func TestChatHistoryCompletionAllowsSettlementFailureAfterDeliveredMessage(t *testing.T) {
	t.Parallel()
	repo := &chatHistoryRepositoryStub{}
	svc := NewChatHistoryService(repo)

	err := svc.FinalizeCompletion(
		context.Background(),
		42,
		&FinalizeChatCompletionInput{
			AttemptID:          "attempt-12345678",
			AssistantMessageID: "message-assistant-12345678",
			Content:            "complete answer",
			CheckpointSeq:      3,
			DeliveryStatus:     ChatMessageDeliveryCompleted,
			AttemptStatus:      ChatAttemptStatusFailed,
			HTTPStatus:         200,
			FinishReason:       "stop",
			ErrorCode:          ChatAttemptFailureCodeSettlement,
			ErrorMessage:       "Chat usage settlement could not be completed",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, repo.finalizeInput)
	require.Equal(t, ChatMessageDeliveryCompleted, repo.finalizeInput.DeliveryStatus)
	require.Equal(t, ChatAttemptStatusFailed, repo.finalizeInput.AttemptStatus)
	require.Equal(t, ChatAttemptFailureCodeSettlement, repo.finalizeInput.ErrorCode)
}

func TestChatHistoryCheckpointNormalizesActivitySnapshot(t *testing.T) {
	t.Parallel()
	repo := &chatHistoryRepositoryStub{}
	svc := NewChatHistoryService(repo)

	err := svc.CheckpointCompletion(context.Background(), 42, &CheckpointChatCompletionInput{
		AttemptID:          " attempt-12345678 ",
		AssistantMessageID: " message-assistant-12345678 ",
		Content:            "answer",
		CheckpointSeq:      3,
		Activities: []ChatMessageActivity{{
			ResponseID:      " resp_1 ",
			Source:          " openai_responses ",
			ActivityType:    " reasoning_summary ",
			ItemID:          " rs_1 ",
			OutputIndex:     0,
			SummaryIndex:    0,
			SortOrder:       1,
			Status:          ChatMessageActivityStatusInProgress,
			Text:            "partial",
			SequenceStart:   1,
			SequenceEnd:     2,
			ReasoningMode:   " pro ",
			ReasoningEffort: " medium ",
		}},
	})

	require.NoError(t, err)
	require.NotNil(t, repo.checkpointInput)
	require.Len(t, repo.checkpointInput.Activities, 1)
	activity := repo.checkpointInput.Activities[0]
	require.Equal(t, "resp_1", activity.ResponseID)
	require.Equal(t, "rs_1", activity.ItemID)
	require.Equal(t, "pro", activity.ReasoningMode)
	require.Equal(t, "medium", activity.ReasoningEffort)
	require.False(t, activity.StartedAt.IsZero())
	require.JSONEq(t, `{}`, string(activity.Metadata))
}

func TestChatHistoryCompletionRejectsInvalidOrDuplicateActivities(t *testing.T) {
	t.Parallel()
	valid := ChatMessageActivity{
		ResponseID:    "resp_1",
		Source:        ChatMessageActivitySourceOpenAIResponses,
		ActivityType:  ChatMessageActivityTypeReasoningSummary,
		ItemID:        "rs_1",
		OutputIndex:   0,
		SummaryIndex:  0,
		SortOrder:     1,
		Status:        ChatMessageActivityStatusCompleted,
		Text:          "summary",
		SequenceStart: 1,
		SequenceEnd:   4,
		Metadata:      []byte(`{}`),
	}
	tests := []struct {
		name       string
		activities []ChatMessageActivity
	}{
		{
			name: "untrusted source",
			activities: []ChatMessageActivity{func() ChatMessageActivity {
				activity := valid
				activity.Source = "chat_completions"
				return activity
			}()},
		},
		{
			name:       "duplicate stable key",
			activities: []ChatMessageActivity{valid, valid},
		},
		{
			name: "invalid metadata",
			activities: []ChatMessageActivity{func() ChatMessageActivity {
				activity := valid
				activity.Metadata = []byte(`[]`)
				return activity
			}()},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := NewChatHistoryService(&chatHistoryRepositoryStub{})
			err := svc.CheckpointCompletion(context.Background(), 42, &CheckpointChatCompletionInput{
				AttemptID:          "attempt-12345678",
				AssistantMessageID: "message-assistant-12345678",
				CheckpointSeq:      1,
				Activities:         test.activities,
			})
			require.ErrorIs(t, err, ErrChatHistoryInvalid)
		})
	}
}
