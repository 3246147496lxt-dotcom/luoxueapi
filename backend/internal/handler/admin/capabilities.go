package admin

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// AccountAdminUseCases is the account handler's actual dependency surface.
// Keep this interface local to the consumer: adding an operation to the legacy
// AdminService must not silently make it available to HTTP handlers.
type AccountAdminUseCases interface {
	ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode string, sortBy, sortOrder string) ([]service.Account, int64, error)
	ListAccountsForSchedulerScoreFilter(ctx context.Context, platform, accountType, status, search string, groupID int64, privacyMode string) ([]service.Account, error)
	ListOpenAISchedulableAccountsForSchedulerScore(ctx context.Context, groupID *int64) ([]service.Account, error)
	GetAccount(ctx context.Context, id int64) (*service.Account, error)
	GetAccountsByIDs(ctx context.Context, ids []int64) ([]*service.Account, error)
	CreateAccount(ctx context.Context, input *service.CreateAccountInput) (*service.Account, error)
	DuplicateAccount(ctx context.Context, id int64, actorScope, operationKey string) (*service.Account, error)
	RecoverDuplicateAccount(ctx context.Context, id int64, actorScope, operationKey string) (*service.Account, error)
	UpdateAccount(ctx context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error)
	UpdateAccountExtra(ctx context.Context, id int64, updates map[string]any) error
	DeleteAccount(ctx context.Context, id int64) error
	ClearAccountError(ctx context.Context, id int64) (*service.Account, error)
	EnsureOpenAIPrivacy(ctx context.Context, account *service.Account) string
	EnsureAntigravityPrivacy(ctx context.Context, account *service.Account) string
	ForceOpenAIPrivacy(ctx context.Context, account *service.Account) string
	ForceAntigravityPrivacy(ctx context.Context, account *service.Account) string
	SetAccountSchedulable(ctx context.Context, id int64, schedulable bool) (*service.Account, error)
	BulkUpdateAccounts(ctx context.Context, input *service.BulkUpdateAccountsInput) (*service.BulkUpdateAccountsResult, error)
	CheckMixedChannelRisk(ctx context.Context, currentAccountID int64, currentAccountPlatform string, groupIDs []int64) error
	RevertAccountProxyFallback(ctx context.Context, id int64) error
	ResetAccountQuota(ctx context.Context, id int64) error

	// Account creation/import can resolve or persist an inline proxy, so only
	// the proxy operations used by those flows are included here.
	ListProxies(ctx context.Context, page, pageSize int, protocol, status, search string, sortBy, sortOrder string) ([]service.Proxy, int64, error)
	GetProxy(ctx context.Context, id int64) (*service.Proxy, error)
	GetProxiesByIDs(ctx context.Context, ids []int64) ([]service.Proxy, error)
	CreateProxy(ctx context.Context, input *service.CreateProxyInput) (*service.Proxy, error)
	UpdateProxy(ctx context.Context, id int64, input *service.UpdateProxyInput) (*service.Proxy, error)
}

type GroupAdminUseCases interface {
	ListGroups(ctx context.Context, page, pageSize int, platform, status, search string, isExclusive *bool, sortBy, sortOrder string) ([]service.Group, int64, error)
	GetAllGroups(ctx context.Context) ([]service.Group, error)
	GetAllGroupsByPlatform(ctx context.Context, platform string) ([]service.Group, error)
	GetAllGroupsIncludingInactive(ctx context.Context) ([]service.Group, error)
	GetGroup(ctx context.Context, id int64) (*service.Group, error)
	GetGroupModelsListCandidates(ctx context.Context, id int64, platform string) ([]string, error)
	CreateGroup(ctx context.Context, input *service.CreateGroupInput) (*service.Group, error)
	UpdateGroup(ctx context.Context, id int64, input *service.UpdateGroupInput) (*service.Group, error)
	DeleteGroup(ctx context.Context, id int64) error
	GetGroupAPIKeys(ctx context.Context, groupID int64, page, pageSize int) ([]service.APIKey, int64, error)
	GetGroupRateMultipliers(ctx context.Context, groupID int64) ([]service.UserGroupRateEntry, error)
	ClearGroupRateMultipliers(ctx context.Context, groupID int64) error
	BatchSetGroupRateMultipliers(ctx context.Context, groupID int64, entries []service.GroupRateMultiplierInput) error
	ClearGroupRPMOverrides(ctx context.Context, groupID int64) error
	BatchSetGroupRPMOverrides(ctx context.Context, groupID int64, entries []service.GroupRPMOverrideInput) error
	UpdateGroupSortOrders(ctx context.Context, updates []service.GroupSortOrderUpdate) error
}

type UserAdminUseCases interface {
	ListUsers(ctx context.Context, page, pageSize int, filters service.UserListFilters, sortBy, sortOrder string) ([]service.User, int64, error)
	GetUser(ctx context.Context, id int64) (*service.User, error)
	GetUserIncludeDeleted(ctx context.Context, id int64) (*service.User, error)
	CreateUser(ctx context.Context, input *service.CreateUserInput) (*service.User, error)
	UpdateUser(ctx context.Context, id int64, input *service.UpdateUserInput) (*service.User, error)
	DeleteUser(ctx context.Context, id int64) error
	UpdateUserBalance(ctx context.Context, userID int64, balance float64, operation string, notes string) (*service.User, error)
	BatchUpdateConcurrency(ctx context.Context, userIDs []int64, value int, mode string) (int, error)
	GetUserAPIKeys(ctx context.Context, userID int64, page, pageSize int, sortBy, sortOrder string) ([]service.APIKey, int64, error)
	GetUserUsageStats(ctx context.Context, userID int64, period string) (any, error)
	GetUserRPMStatus(ctx context.Context, userID int64) (*service.UserRPMStatus, error)
	GetUserBalanceHistory(ctx context.Context, userID int64, page, pageSize int, codeType string) ([]service.RedeemCode, int64, float64, error)
	BindUserAuthIdentity(ctx context.Context, userID int64, input service.AdminBindAuthIdentityInput) (*service.AdminBoundAuthIdentity, error)
	ReplaceUserGroup(ctx context.Context, userID, oldGroupID, newGroupID int64) (*service.ReplaceUserGroupResult, error)
}

type ProxyAdminUseCases interface {
	ListProxies(ctx context.Context, page, pageSize int, protocol, status, search string, sortBy, sortOrder string) ([]service.Proxy, int64, error)
	ListProxiesWithAccountCount(ctx context.Context, page, pageSize int, protocol, status, search string, sortBy, sortOrder string) ([]service.ProxyWithAccountCount, int64, error)
	GetAllProxies(ctx context.Context) ([]service.Proxy, error)
	GetAllProxiesWithAccountCount(ctx context.Context) ([]service.ProxyWithAccountCount, error)
	GetProxy(ctx context.Context, id int64) (*service.Proxy, error)
	GetProxiesByIDs(ctx context.Context, ids []int64) ([]service.Proxy, error)
	CreateProxy(ctx context.Context, input *service.CreateProxyInput) (*service.Proxy, error)
	UpdateProxy(ctx context.Context, id int64, input *service.UpdateProxyInput) (*service.Proxy, error)
	DeleteProxy(ctx context.Context, id int64) error
	BatchDeleteProxies(ctx context.Context, ids []int64) (*service.ProxyBatchDeleteResult, error)
	GetProxyAccounts(ctx context.Context, proxyID int64) ([]service.ProxyAccountSummary, error)
	CheckProxyExists(ctx context.Context, host string, port int, username, password string) (bool, error)
	TestProxy(ctx context.Context, id int64) (*service.ProxyTestResult, error)
	CheckProxyQuality(ctx context.Context, id int64) (*service.ProxyQualityCheckResult, error)
}

var (
	_ AccountAdminUseCases = (service.AdminService)(nil)
	_ GroupAdminUseCases   = (service.AdminService)(nil)
	_ UserAdminUseCases    = (service.AdminService)(nil)
	_ ProxyAdminUseCases   = (service.AdminService)(nil)
)
