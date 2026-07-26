package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type adminChatHistoryServiceRepositoryStub struct {
	listQuery AdminChatConversationListQuery
	viewQuery AdminChatContentAccessQuery
}

func (s *adminChatHistoryServiceRepositoryStub) ListUserConversationRefs(
	_ context.Context,
	query AdminChatConversationListQuery,
) (*AdminChatConversationRefPage, error) {
	s.listQuery = query
	return &AdminChatConversationRefPage{}, nil
}

func (s *adminChatHistoryServiceRepositoryStub) ViewConversationPageAndRecordAccess(
	_ context.Context,
	query AdminChatContentAccessQuery,
) (*AdminChatConversationPage, error) {
	s.viewQuery = query
	return &AdminChatConversationPage{}, nil
}

func TestAdminChatHistoryServiceNormalizesLimitsAndAuditMetadata(t *testing.T) {
	repo := &adminChatHistoryServiceRepositoryStub{}
	svc := NewAdminChatHistoryService(repo)

	_, err := svc.ListUserConversationRefs(
		context.Background(),
		AdminChatConversationListQuery{
			TargetUserID: 42,
			Limit:        MaxAdminChatPageLimit + 1,
		},
	)
	require.NoError(t, err)
	require.Equal(t, MaxAdminChatPageLimit, repo.listQuery.Limit)

	_, err = svc.ViewConversationPage(
		context.Background(),
		AdminChatContentAccessQuery{
			AdminID:              7,
			TargetUserID:         42,
			ConversationPublicID: " conversation-1 ",
			Limit:                0,
			RequestID:            " " + strings.Repeat("r", 80) + " ",
			ClientIP:             strings.Repeat("i", 80),
			UserAgent:            strings.Repeat("u", 600),
		},
	)
	require.NoError(t, err)
	require.Equal(t, "conversation-1", repo.viewQuery.ConversationPublicID)
	require.Equal(t, DefaultAdminChatPageLimit, repo.viewQuery.Limit)
	require.Len(t, repo.viewQuery.RequestID, 64)
	require.Len(t, repo.viewQuery.ClientIP, 64)
	require.Len(t, repo.viewQuery.UserAgent, 512)
}

func TestAdminChatHistoryServiceRejectsInvalidOwnershipInputs(t *testing.T) {
	repo := &adminChatHistoryServiceRepositoryStub{}
	svc := NewAdminChatHistoryService(repo)

	_, err := svc.ListUserConversationRefs(
		context.Background(),
		AdminChatConversationListQuery{},
	)
	require.ErrorIs(t, err, ErrAdminChatHistoryInvalid)

	_, err = svc.ViewConversationPage(
		context.Background(),
		AdminChatContentAccessQuery{
			AdminID:              7,
			TargetUserID:         42,
			ConversationPublicID: "",
		},
	)
	require.ErrorIs(t, err, ErrAdminChatHistoryInvalid)
}

func TestAdminChatHistoryServiceRequiresRepository(t *testing.T) {
	var svc *AdminChatHistoryService
	_, err := svc.ViewConversationPage(
		context.Background(),
		AdminChatContentAccessQuery{},
	)
	require.ErrorIs(t, err, ErrAdminChatContentAuditUnavailable)
}
