package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/ent/authidentitychannel"
	dbgroup "github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/identityadoptiondecision"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/ent/userallowedgroup"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"

	entsql "entgo.io/ent/dialect/sql"
)

type userRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

var _ service.RedeemUserAdjustmentRepository = (*userRepository)(nil)

func NewUserRepository(client *dbent.Client, sqlDB *sql.DB) service.UserRepository {
	return newUserRepositoryWithSQL(client, sqlDB)
}

func newUserRepositoryWithSQL(client *dbent.Client, sqlq sqlExecutor) *userRepository {
	return &userRepository{client: client, sql: sqlq}
}

func (r *userRepository) Create(ctx context.Context, userIn *service.User) error {
	return r.create(ctx, userIn, false)
}

// CreateWithEmailAliasGuard is the registration-only variant of Create. It
// serializes aliases that resolve to the same inbox and rechecks the alias
// identity while holding the repository lock.
func (r *userRepository) CreateWithEmailAliasGuard(ctx context.Context, userIn *service.User) error {
	return r.create(ctx, userIn, true)
}

func (r *userRepository) create(ctx context.Context, userIn *service.User, guardEmailAlias bool) error {
	if userIn == nil {
		return nil
	}

	// 统一使用 ent 的事务：保证用户与允许分组的更新原子化，
	// 并避免基于 *sql.Tx 手动构造 ent client 导致的 ExecQuerier 断言错误。
	var txClient *dbent.Client
	txCtx := ctx
	var ownedTx *dbent.Tx
	// ent.Client.Tx does not inspect TxFromContext, so explicitly reuse an
	// outer transaction (registration needs user creation and invitation claim
	// to commit or roll back together).
	if existingTx := dbent.TxFromContext(ctx); existingTx != nil {
		txClient = existingTx.Client()
	} else {
		tx, err := r.client.Tx(ctx)
		if errors.Is(err, dbent.ErrTxStarted) {
			// Some callers construct a repository with tx.Client() but cannot
			// propagate the matching Tx context.  Ent reports ErrTxStarted in
			// that compatibility mode; the repository must use the supplied
			// transaction client and leave ownership/commit to the caller.
			txClient = r.client
		} else if err != nil {
			return err
		} else {
			ownedTx = tx
			defer func() { _ = ownedTx.Rollback() }()
			txClient = tx.Client()
			txCtx = dbent.NewTxContext(ctx, tx)
		}
	}

	lockKeys := []string{normalizedEmailUniquenessLockKey(userIn.Email)}
	if guardEmailAlias {
		lockKeys = append(lockKeys, emailAliasUniquenessLockKey(userIn.Email))
	}

	releaseEmailLock, err := lockRepositoryScopedKeys(
		txCtx,
		txClient,
		txAwareSQLExecutor(txCtx, r.sql, r.client),
		lockKeys...,
	)
	if err != nil {
		return err
	}
	defer releaseEmailLock()

	if err := ensureNormalizedEmailAvailableWithClient(txCtx, txClient, 0, userIn.Email); err != nil {
		return err
	}
	if guardEmailAlias {
		aliasExists, err := existsByEmailAliasWithClient(txCtx, txClient, userIn.Email)
		if err != nil {
			return err
		}
		if aliasExists {
			return service.ErrEmailExists
		}
	}

	created, err := txClient.User.Create().
		SetEmail(userIn.Email).
		SetUsername(userIn.Username).
		SetNotes(userIn.Notes).
		SetPasswordHash(userIn.PasswordHash).
		SetRole(userIn.Role).
		SetBalance(userIn.Balance).
		SetConcurrency(userIn.Concurrency).
		SetStatus(userIn.Status).
		SetSignupSource(userSignupSourceOrDefault(userIn.SignupSource)).
		SetNillableLastLoginAt(userIn.LastLoginAt).
		SetNillableLastActiveAt(userIn.LastActiveAt).
		SetRpmLimit(userIn.RPMLimit).
		Save(txCtx)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrEmailExists)
	}

	if err := r.syncUserAllowedGroupsWithClient(txCtx, txClient, created.ID, userIn.AllowedGroups); err != nil {
		return err
	}
	if err := ensureEmailAuthIdentityWithClient(txCtx, txClient, created.ID, created.Email, "user_repo_create"); err != nil {
		return err
	}

	if ownedTx != nil {
		if err := ownedTx.Commit(); err != nil {
			return err
		}
	}

	applyUserEntityToService(userIn, created)
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*service.User, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.User.Query().Where(dbuser.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}

	out := userEntityToService(m)
	groups, err := r.loadAllowedGroups(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	if v, ok := groups[id]; ok {
		out.AllowedGroups = v
	}
	return out, nil
}

func (r *userRepository) GetByIDIncludeDeleted(ctx context.Context, id int64) (*service.User, error) {
	ctx = mixins.SkipSoftDelete(ctx)
	client := clientFromContext(ctx, r.client)
	m, err := client.User.Query().Where(dbuser.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	out := userEntityToService(m)
	groups, err := r.loadAllowedGroups(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	if v, ok := groups[id]; ok {
		out.AllowedGroups = v
	}
	return out, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*service.User, error) {
	client := clientFromContext(ctx, r.client)
	matches, err := client.User.Query().
		Where(userEmailLookupPredicate(email)).
		Order(dbent.Asc(dbuser.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, service.ErrUserNotFound
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("normalized email lookup matched multiple users for %q", strings.TrimSpace(email))
	}
	m := matches[0]

	out := userEntityToService(m)
	groups, err := r.loadAllowedGroups(ctx, []int64{m.ID})
	if err != nil {
		return nil, err
	}
	if v, ok := groups[m.ID]; ok {
		out.AllowedGroups = v
	}
	return out, nil
}

func (r *userRepository) Update(ctx context.Context, userIn *service.User, fields service.UserUpdateFields) error {
	if userIn == nil {
		return nil
	}
	if fields.IsEmpty() {
		return nil
	}

	// 使用 ent 事务包裹用户更新与 allowed_groups 同步，避免跨层事务不一致。
	// Ent 的 Client.Tx 不读取 context 中的事务；显式复用外层事务，否则在
	// UpdateProfile 等调用链中会开启独立事务，破坏头像/身份与用户更新的原子性。
	var txClient *dbent.Client
	txCtx := ctx
	var ownedTx *dbent.Tx
	if existingTx := dbent.TxFromContext(ctx); existingTx != nil {
		txClient = existingTx.Client()
	} else {
		tx, err := r.client.Tx(ctx)
		if errors.Is(err, dbent.ErrTxStarted) {
			// See create(): a tx.Client()-backed repository may arrive without
			// TxFromContext.  Do not open/commit a nested transaction.
			txClient = r.client
		} else if err != nil {
			return err
		} else {
			ownedTx = tx
			defer func() { _ = ownedTx.Rollback() }()
			txClient = tx.Client()
			txCtx = dbent.NewTxContext(ctx, tx)
		}
	}

	if fields.Email {
		releaseEmailLock, err := lockRepositoryScopedKeys(
			txCtx,
			txClient,
			txAwareSQLExecutor(txCtx, r.sql, r.client),
			normalizedEmailUniquenessLockKey(userIn.Email),
			emailAliasUniquenessLockKey(userIn.Email),
			userEmailIdentityLockKey(userIn.ID),
		)
		if err != nil {
			return err
		}
		defer releaseEmailLock()
		if err := ensureNormalizedEmailAvailableWithClient(txCtx, txClient, userIn.ID, userIn.Email); err != nil {
			return err
		}
		// Generic profile/admin email updates must use the same inbox-identity
		// guard as explicit email binding.  The service-layer precheck is only a
		// fast path; this locked recheck closes the concurrent alias race.
		ownerID, aliasExists, err := emailAliasOwnerIDWithClient(txCtx, txClient, userIn.Email, userIn.ID)
		if err != nil {
			return err
		}
		if aliasExists && ownerID != userIn.ID {
			return service.ErrEmailExists
		}
	}

	existing, err := clientFromContext(txCtx, txClient).User.Get(txCtx, userIn.ID)
	if err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	oldEmail := existing.Email

	// UpdateOneID does not inherit the soft-delete interceptor (it only applies
	// to queries and delete mutations). Keep the liveness predicate on the
	// write itself so a concurrent soft-delete cannot be overwritten by a stale
	// profile snapshot.
	updateOp := txClient.User.UpdateOneID(userIn.ID).Where(dbuser.DeletedAtIsNil())
	hasColumnUpdate := false
	if fields.Email {
		updateOp = updateOp.SetEmail(userIn.Email)
		hasColumnUpdate = true
	}
	if fields.Username {
		updateOp = updateOp.SetUsername(userIn.Username)
		hasColumnUpdate = true
	}
	if fields.Notes {
		updateOp = updateOp.SetNotes(userIn.Notes)
		hasColumnUpdate = true
	}
	if fields.PasswordHash {
		updateOp = updateOp.SetPasswordHash(userIn.PasswordHash)
		hasColumnUpdate = true
	}
	if fields.Role {
		updateOp = updateOp.SetRole(userIn.Role)
		hasColumnUpdate = true
	}
	if fields.Concurrency {
		updateOp = updateOp.SetConcurrency(userIn.Concurrency)
		hasColumnUpdate = true
	}
	if fields.Status {
		updateOp = updateOp.SetStatus(userIn.Status)
		hasColumnUpdate = true
	}
	if fields.RPMLimit {
		updateOp = updateOp.SetRpmLimit(userIn.RPMLimit)
		hasColumnUpdate = true
	}
	if fields.BalanceNotifySettings {
		updateOp = updateOp.SetBalanceNotifyEnabled(userIn.BalanceNotifyEnabled).
			SetBalanceNotifyThresholdType(userIn.BalanceNotifyThresholdType).
			SetNillableBalanceNotifyThreshold(userIn.BalanceNotifyThreshold)
		hasColumnUpdate = true
		if userIn.BalanceNotifyThreshold == nil {
			updateOp = updateOp.ClearBalanceNotifyThreshold()
		}
	}
	if fields.BalanceNotifyExtraEmails {
		updateOp = updateOp.SetBalanceNotifyExtraEmails(marshalExtraEmails(userIn.BalanceNotifyExtraEmails))
		hasColumnUpdate = true
	}
	if fields.SignupSource && userIn.SignupSource != "" {
		updateOp = updateOp.SetSignupSource(userIn.SignupSource)
		hasColumnUpdate = true
	}
	if fields.LastLoginAt && userIn.LastLoginAt != nil {
		updateOp = updateOp.SetLastLoginAt(*userIn.LastLoginAt)
		hasColumnUpdate = true
	}
	if fields.LastActiveAt && userIn.LastActiveAt != nil {
		updateOp = updateOp.SetLastActiveAt(*userIn.LastActiveAt)
		hasColumnUpdate = true
	}
	// AllowedGroups is a relation-only update. Ent rejects UpdateOneID.Save
	// when no scalar field was set, so keep the initial row for that case and
	// apply the join-table synchronization below without issuing an empty UPDATE.
	updated := existing
	if hasColumnUpdate {
		updated, err = updateOp.Save(txCtx)
		if err != nil {
			return translatePersistenceError(err, service.ErrUserNotFound, service.ErrEmailExists)
		}
	}

	if fields.AllowedGroups {
		if err := r.syncUserAllowedGroupsWithClient(txCtx, txClient, updated.ID, userIn.AllowedGroups); err != nil {
			return err
		}
	}
	if fields.Email {
		if err := replaceEmailAuthIdentityWithClient(txCtx, txClient, updated.ID, oldEmail, updated.Email, "user_repo_update"); err != nil {
			return err
		}
	}

	if ownedTx != nil {
		if err := ownedTx.Commit(); err != nil {
			return err
		}
	}

	userIn.UpdatedAt = updated.UpdatedAt
	return nil
}

func ensureEmailAuthIdentityWithClient(ctx context.Context, client *dbent.Client, userID int64, email string, source string) error {
	client = clientFromContext(ctx, client)
	if client == nil || userID <= 0 {
		return nil
	}

	subject := normalizeEmailAuthIdentitySubject(email)
	if subject == "" {
		return nil
	}

	if err := client.AuthIdentity.Create().
		SetUserID(userID).
		SetProviderType("email").
		SetProviderKey("email").
		SetProviderSubject(subject).
		SetVerifiedAt(time.Now().UTC()).
		SetMetadata(map[string]any{"source": source}).
		OnConflictColumns(
			authidentity.FieldProviderType,
			authidentity.FieldProviderKey,
			authidentity.FieldProviderSubject,
		).
		DoNothing().
		Exec(ctx); err != nil {
		if !isSQLNoRowsError(err) {
			return err
		}
	}

	identity, err := client.AuthIdentity.Query().
		Where(
			authidentity.ProviderTypeEQ("email"),
			authidentity.ProviderKeyEQ("email"),
			authidentity.ProviderSubjectEQ(subject),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil
		}
		return err
	}
	if identity.UserID != userID {
		return ErrAuthIdentityOwnershipConflict
	}
	return nil
}

func replaceEmailAuthIdentityWithClient(ctx context.Context, client *dbent.Client, userID int64, oldEmail, newEmail string, source string) error {
	newSubject := normalizeEmailAuthIdentitySubject(newEmail)
	if err := ensureEmailAuthIdentityWithClient(ctx, client, userID, newEmail, source); err != nil {
		return err
	}

	// A stale caller snapshot can make two concurrent email updates each try to
	// remove only its own observed old subject. Keep the new primary subject and
	// remove every other email identity so a stale update cannot leave multiple
	// local-email login identities attached to one user. The oldEmail argument
	// remains part of the helper contract for callers that track the previous
	// value; cleanup intentionally does not rely on it.
	_ = oldEmail
	deleteQuery := clientFromContext(ctx, client).AuthIdentity.Delete().Where(
		authidentity.UserIDEQ(userID),
		authidentity.ProviderTypeEQ("email"),
		authidentity.ProviderKeyEQ("email"),
	)
	if newSubject != "" {
		deleteQuery = deleteQuery.Where(authidentity.ProviderSubjectNEQ(newSubject))
	}
	_, err := deleteQuery.Exec(ctx)
	return err
}

func normalizeEmailAuthIdentitySubject(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return ""
	}
	if strings.HasSuffix(normalized, service.LinuxDoConnectSyntheticEmailDomain) ||
		strings.HasSuffix(normalized, service.OIDCConnectSyntheticEmailDomain) ||
		strings.HasSuffix(normalized, service.WeChatConnectSyntheticEmailDomain) ||
		strings.HasSuffix(normalized, service.DingTalkConnectSyntheticEmailDomain) {
		return ""
	}
	return normalized
}

func (r *userRepository) Delete(ctx context.Context, id int64) error {
	// 复用 context 中已存在的事务（如 AdminService.DeleteUser 把删 Key 与删 User 包在同一事务中），
	// 由调用方负责提交/回滚，保证两者的原子性。
	if existingTx := dbent.TxFromContext(ctx); existingTx != nil {
		return r.deleteUser(ctx, existingTx.Client(), id)
	}

	tx, err := r.client.Tx(ctx)
	if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	exec := r.client
	if err == nil {
		defer func() { _ = tx.Rollback() }()
		exec = tx.Client()
	}
	// err == dbent.ErrTxStarted 时复用当前事务（exec = r.client）。

	if err := r.deleteUser(ctx, exec, id); err != nil {
		return err
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return translatePersistenceError(err, service.ErrUserNotFound, nil)
		}
	}
	return nil
}

// deleteUser 在给定 client（可能是外部事务 client）上删除用户及其身份关联记录，自身不开启/提交事务。
func (r *userRepository) deleteUser(ctx context.Context, exec *dbent.Client, id int64) error {
	identityIDs, err := exec.AuthIdentity.Query().
		Where(authidentity.UserIDEQ(id)).
		IDs(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	if len(identityIDs) > 0 {
		if _, err := exec.IdentityAdoptionDecision.Update().
			Where(identityadoptiondecision.IdentityIDIn(identityIDs...)).
			ClearIdentityID().
			Save(ctx); err != nil {
			return translatePersistenceError(err, service.ErrUserNotFound, nil)
		}
		if _, err := exec.AuthIdentityChannel.Delete().
			Where(authidentitychannel.IdentityIDIn(identityIDs...)).
			Exec(ctx); err != nil {
			return translatePersistenceError(err, service.ErrUserNotFound, nil)
		}
		if _, err := exec.AuthIdentity.Delete().
			Where(authidentity.UserIDEQ(id)).
			Exec(ctx); err != nil {
			return translatePersistenceError(err, service.ErrUserNotFound, nil)
		}
	}

	affected, err := exec.User.Delete().Where(dbuser.IDEQ(id)).Exec(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	if affected == 0 {
		return service.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	return r.ListWithFilters(ctx, params, service.UserListFilters{})
}

func (r *userRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	// SkipSoftDelete 仅作用于 User 身份解析（下方 Count/All）；订阅、分组等关联实体沿用原始 ctx，避免穿透到这些同样带软删除的实体而带出已删除行。
	userCtx := ctx
	if filters.IncludeDeleted {
		userCtx = mixins.SkipSoftDelete(ctx)
	}

	client := clientFromContext(ctx, r.client)
	q := client.User.Query()

	if filters.Status != "" {
		q = q.Where(dbuser.StatusEQ(filters.Status))
	}
	if filters.Role != "" {
		q = q.Where(dbuser.RoleEQ(filters.Role))
	}
	if filters.Search != "" {
		q = q.Where(
			dbuser.Or(
				dbuser.EmailContainsFold(filters.Search),
				dbuser.UsernameContainsFold(filters.Search),
				dbuser.NotesContainsFold(filters.Search),
				dbuser.HasAPIKeysWith(
					apikey.KeyContainsFold(filters.Search),
					apikey.PurposeEQ(service.APIKeyPurposeUser),
				),
			),
		)
	}

	if filters.GroupName != "" {
		q = q.Where(dbuser.HasAllowedGroupsWith(
			dbgroup.NameContainsFold(filters.GroupName),
		))
	}

	if filters.APIKeyGroupID > 0 {
		// 按"API Key 实际绑定的分组"过滤：用户只要有任意一个未软删除的 API Key
		// 绑定到该分组即命中（EXISTS 语义）。
		// 注意：SoftDeleteMixin 的拦截器不会自动下沉到 HasAPIKeysWith 子查询，
		// 必须显式加 apikey.DeletedAtIsNil()，否则已软删除的 key 会污染过滤结果。
		q = q.Where(dbuser.HasAPIKeysWith(
			apikey.GroupIDEQ(filters.APIKeyGroupID),
			apikey.DeletedAtIsNil(),
			apikey.PurposeEQ(service.APIKeyPurposeUser),
		))
	}

	// If attribute filters are specified, we need to filter by user IDs first
	var allowedUserIDs []int64
	if len(filters.Attributes) > 0 {
		var attrErr error
		allowedUserIDs, attrErr = r.filterUsersByAttributes(ctx, filters.Attributes)
		if attrErr != nil {
			return nil, nil, attrErr
		}
		if len(allowedUserIDs) == 0 {
			// No users match the attribute filters
			return []service.User{}, paginationResultFromTotal(0, params), nil
		}
		q = q.Where(dbuser.IDIn(allowedUserIDs...))
	}

	total, err := q.Clone().Count(userCtx)
	if err != nil {
		return nil, nil, err
	}

	usersQuery := q.
		Offset(params.Offset()).
		Limit(params.Limit())
	for _, order := range userListOrder(params) {
		usersQuery = usersQuery.Order(order)
	}

	users, err := usersQuery.All(userCtx)
	if err != nil {
		return nil, nil, err
	}

	outUsers := make([]service.User, 0, len(users))
	if len(users) == 0 {
		return outUsers, paginationResultFromTotal(int64(total), params), nil
	}

	userIDs := make([]int64, 0, len(users))
	userMap := make(map[int64]*service.User, len(users))
	for i := range users {
		userIDs = append(userIDs, users[i].ID)
		u := userEntityToService(users[i])
		outUsers = append(outUsers, *u)
		userMap[u.ID] = &outUsers[len(outUsers)-1]
	}

	shouldLoadSubscriptions := filters.IncludeSubscriptions == nil || *filters.IncludeSubscriptions
	if shouldLoadSubscriptions {
		// Batch load active subscriptions with groups to avoid N+1.
		subs, err := client.UserSubscription.Query().
			Where(
				usersubscription.UserIDIn(userIDs...),
				usersubscription.StatusEQ(service.SubscriptionStatusActive),
			).
			WithGroup().
			All(ctx)
		if err != nil {
			return nil, nil, err
		}

		for i := range subs {
			if u, ok := userMap[subs[i].UserID]; ok {
				u.Subscriptions = append(u.Subscriptions, *userSubscriptionEntityToService(subs[i]))
			}
		}
	}

	allowedGroupsByUser, err := r.loadAllowedGroups(ctx, userIDs)
	if err != nil {
		return nil, nil, err
	}
	for id, u := range userMap {
		if groups, ok := allowedGroupsByUser[id]; ok {
			u.AllowedGroups = groups
		}
	}

	return outUsers, paginationResultFromTotal(int64(total), params), nil
}

func userListOrder(params pagination.PaginationParams) []func(*entsql.Selector) {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)

	if sortBy == "last_used_at" {
		return userLastUsedAtOrder(sortOrder)
	}

	var field string
	defaultField := true
	nullsLastField := false
	switch sortBy {
	case "email":
		field = dbuser.FieldEmail
		defaultField = false
	case "username":
		field = dbuser.FieldUsername
		defaultField = false
	case "role":
		field = dbuser.FieldRole
		defaultField = false
	case "balance":
		field = dbuser.FieldBalance
		defaultField = false
	case "concurrency":
		field = dbuser.FieldConcurrency
		defaultField = false
	case "status":
		field = dbuser.FieldStatus
		defaultField = false
	case "created_at":
		field = dbuser.FieldCreatedAt
		defaultField = false
	case "last_active_at":
		field = dbuser.FieldLastActiveAt
		defaultField = false
		nullsLastField = true
	default:
		field = dbuser.FieldID
	}

	if sortOrder == pagination.SortOrderAsc {
		if defaultField && field == dbuser.FieldID {
			return []func(*entsql.Selector){dbent.Asc(dbuser.FieldID)}
		}
		if nullsLastField {
			return []func(*entsql.Selector){
				entsql.OrderByField(field, entsql.OrderNullsLast()).ToFunc(),
				dbent.Asc(dbuser.FieldID),
			}
		}
		return []func(*entsql.Selector){dbent.Asc(field), dbent.Asc(dbuser.FieldID)}
	}
	if defaultField && field == dbuser.FieldID {
		return []func(*entsql.Selector){dbent.Desc(dbuser.FieldID)}
	}
	if nullsLastField {
		return []func(*entsql.Selector){
			entsql.OrderByField(field, entsql.OrderDesc(), entsql.OrderNullsLast()).ToFunc(),
			dbent.Desc(dbuser.FieldID),
		}
	}
	return []func(*entsql.Selector){dbent.Desc(field), dbent.Desc(dbuser.FieldID)}
}

func (r *userRepository) GetLatestUsedAtByUserIDs(ctx context.Context, userIDs []int64) (map[int64]*time.Time, error) {
	result := make(map[int64]*time.Time, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT user_id, MAX(created_at) AS last_used_at
		FROM usage_logs
		WHERE user_id = ANY($1)
		GROUP BY user_id
	`

	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return nil, fmt.Errorf("sql executor is not configured")
	}
	rows, err := exec.QueryContext(ctx, query, pq.Array(userIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			userID     int64
			lastUsedAt time.Time
		)
		if scanErr := rows.Scan(&userID, &lastUsedAt); scanErr != nil {
			return nil, scanErr
		}
		ts := lastUsedAt.UTC()
		result[userID] = &ts
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *userRepository) GetLatestUsedAtByUserID(ctx context.Context, userID int64) (*time.Time, error) {
	latestByUserID, err := r.GetLatestUsedAtByUserIDs(ctx, []int64{userID})
	if err != nil {
		return nil, err
	}
	return latestByUserID[userID], nil
}

func userLastUsedAtOrder(sortOrder string) []func(*entsql.Selector) {
	orderExpr := func(direction, nulls string, tieOrder func(string) string) func(*entsql.Selector) {
		return func(s *entsql.Selector) {
			subquery := fmt.Sprintf("(SELECT MAX(created_at) FROM usage_logs WHERE user_id = %s)", s.C(dbuser.FieldID))
			s.OrderExpr(entsql.Expr(subquery + " " + direction + " NULLS " + nulls))
			s.OrderBy(tieOrder(s.C(dbuser.FieldID)))
		}
	}

	if sortOrder == pagination.SortOrderAsc {
		return []func(*entsql.Selector){
			orderExpr("ASC", "FIRST", entsql.Asc),
		}
	}
	return []func(*entsql.Selector){
		orderExpr("DESC", "LAST", entsql.Desc),
	}
}

// filterUsersByAttributes returns user IDs that match ALL the given attribute filters
func (r *userRepository) filterUsersByAttributes(ctx context.Context, attrs map[int64]string) ([]int64, error) {
	if len(attrs) == 0 {
		return nil, nil
	}

	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return nil, fmt.Errorf("sql executor is not configured")
	}

	clauses := make([]string, 0, len(attrs))
	args := make([]any, 0, len(attrs)*2+1)
	argIndex := 1
	for attrID, value := range attrs {
		clauses = append(clauses, fmt.Sprintf("(attribute_id = $%d AND value ILIKE $%d)", argIndex, argIndex+1))
		args = append(args, attrID, "%"+value+"%")
		argIndex += 2
	}

	query := fmt.Sprintf(
		`SELECT user_id
		 FROM user_attribute_values
		 WHERE %s
		 GROUP BY user_id
		 HAVING COUNT(DISTINCT attribute_id) = $%d`,
		strings.Join(clauses, " OR "),
		argIndex,
	)
	args = append(args, len(attrs))

	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make([]int64, 0)
	for rows.Next() {
		var userID int64
		if scanErr := rows.Scan(&userID); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *userRepository) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	client := clientFromContext(ctx, r.client)
	update := client.User.Update().Where(dbuser.IDEQ(id), dbuser.DeletedAtIsNil()).AddBalance(amount)
	// Track cumulative recharge amount for percentage-based notifications
	if amount > 0 {
		update = update.AddTotalRecharged(amount)
	}
	n, err := update.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	if n == 0 {
		return service.ErrUserNotFound
	}
	return nil
}

// AdjustBalance atomically applies delta and rejects a negative resulting balance.
func (r *userRepository) AdjustBalance(ctx context.Context, id int64, delta float64) (service.BalanceChange, error) {
	const q = `
		UPDATE users
		SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance + $1 >= 0
		RETURNING balance - $1, balance`
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, q, delta, id)
	if err != nil {
		return service.BalanceChange{}, err
	}
	defer rows.Close()
	if rows.Next() {
		var c service.BalanceChange
		if err := rows.Scan(&c.Old, &c.New); err != nil {
			return service.BalanceChange{}, err
		}
		return c, rows.Err()
	}
	if err := rows.Err(); err != nil {
		return service.BalanceChange{}, err
	}
	rows2, err := clientFromContext(ctx, r.client).QueryContext(ctx, "SELECT balance FROM users WHERE id = $1 AND deleted_at IS NULL", id)
	if err != nil {
		return service.BalanceChange{}, err
	}
	defer rows2.Close()
	if !rows2.Next() {
		if rowsErr := rows2.Err(); rowsErr != nil {
			return service.BalanceChange{}, rowsErr
		}
		return service.BalanceChange{}, service.ErrUserNotFound
	}
	var current float64
	if err := rows2.Scan(&current); err != nil {
		return service.BalanceChange{}, err
	}
	return service.BalanceChange{Old: current, New: current + delta}, service.ErrBalanceNegative
}

// SetBalance atomically sets a non-negative balance and returns before/after values.
func (r *userRepository) SetBalance(ctx context.Context, id int64, value float64) (service.BalanceChange, error) {
	if value < 0 {
		return service.BalanceChange{}, service.ErrBalanceNegative
	}
	const q = `
		UPDATE users AS u
		SET balance = $1, updated_at = NOW()
		FROM (SELECT id, balance FROM users WHERE id = $2 AND deleted_at IS NULL) AS prev
		WHERE u.id = prev.id AND u.deleted_at IS NULL
		RETURNING prev.balance, u.balance`
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, q, value, id)
	if err != nil {
		return service.BalanceChange{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return service.BalanceChange{}, err
		}
		return service.BalanceChange{}, service.ErrUserNotFound
	}
	var c service.BalanceChange
	if err := rows.Scan(&c.Old, &c.New); err != nil {
		return service.BalanceChange{}, err
	}
	return c, rows.Err()
}

func (r *userRepository) ApplyRedeemBalanceAdjustment(ctx context.Context, id int64, delta float64) error {
	const updateSQL = `
		UPDATE users
		SET balance = GREATEST(balance + $1, 0), updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`
	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx, updateSQL, delta, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrUserNotFound
	}
	return nil
}

// DeductBalance 扣除用户余额
// 透支策略：允许余额变为负数，确保当前请求能够完成
// 中间件会阻止余额 <= 0 的用户发起后续请求
func (r *userRepository) DeductBalance(ctx context.Context, id int64, amount float64) error {
	client := clientFromContext(ctx, r.client)
	n, err := client.User.Update().
		Where(dbuser.IDEQ(id), dbuser.DeletedAtIsNil(), dbuser.BalanceGTE(amount)).
		AddBalance(-amount).
		Save(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	n, err = client.User.Update().
		Where(dbuser.IDEQ(id), dbuser.DeletedAtIsNil()).
		AddBalance(-amount).
		Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return service.ErrUserNotFound
	}
	return nil
}

// DeductAvailableBalance atomically deducts min(amount, max(balance, 0)).
// Refunds use this operation after the gateway has accepted the refund so a
// concurrent spend cannot turn a refund into an unintended overdraft.
func (r *userRepository) DeductAvailableBalance(ctx context.Context, id int64, amount float64) (deducted float64, err error) {
	if amount < 0 {
		return 0, fmt.Errorf("deduction amount must be nonnegative")
	}
	const updateSQL = `
		WITH target AS (
			SELECT id, balance
			FROM users
			WHERE id = $2 AND deleted_at IS NULL
			FOR UPDATE
		), updated AS (
			UPDATE users AS u
			SET balance = target.balance - LEAST($1, GREATEST(target.balance, 0)), updated_at = NOW()
			FROM target
			WHERE u.id = target.id AND u.deleted_at IS NULL
			RETURNING target.balance - u.balance AS deducted
		)
		SELECT deducted FROM updated
	`
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, updateSQL, amount, id)
	if err != nil {
		return 0, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	if !rows.Next() {
		if rowsErr := rows.Err(); rowsErr != nil {
			return 0, rowsErr
		}
		return 0, service.ErrUserNotFound
	}
	if err := rows.Scan(&deducted); err != nil {
		return 0, err
	}
	return deducted, rows.Err()
}

func (r *userRepository) UpdateConcurrency(ctx context.Context, id int64, amount int) error {
	client := clientFromContext(ctx, r.client)
	n, err := client.User.Update().Where(dbuser.IDEQ(id), dbuser.DeletedAtIsNil()).AddConcurrency(amount).Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	if n == 0 {
		return service.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) ApplyRedeemConcurrencyAdjustment(ctx context.Context, id int64, delta int) error {
	const updateSQL = `
		UPDATE users
		SET concurrency = GREATEST(concurrency + $1, 0), updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`
	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx, updateSQL, delta, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) BatchSetConcurrency(ctx context.Context, userIDs []int64, value int) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}
	if value < 0 {
		value = 0
	}
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return 0, fmt.Errorf("sql executor is not configured")
	}
	res, err := exec.ExecContext(ctx,
		"UPDATE users SET concurrency = $1, updated_at = NOW() WHERE id = ANY($2) AND deleted_at IS NULL",
		value, pq.Array(userIDs))
	if err != nil {
		return 0, fmt.Errorf("batch set concurrency: %w", err)
	}
	affected, _ := res.RowsAffected()
	return int(affected), nil
}

func (r *userRepository) BatchAddConcurrency(ctx context.Context, userIDs []int64, delta int) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return 0, fmt.Errorf("sql executor is not configured")
	}
	res, err := exec.ExecContext(ctx,
		"UPDATE users SET concurrency = GREATEST(concurrency + $1, 0), updated_at = NOW() WHERE id = ANY($2) AND deleted_at IS NULL",
		delta, pq.Array(userIDs))
	if err != nil {
		return 0, fmt.Errorf("batch add concurrency: %w", err)
	}
	affected, _ := res.RowsAffected()
	return int(affected), nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	client := clientFromContext(ctx, r.client)
	return client.User.Query().Where(userEmailLookupPredicate(email)).Exist(ctx)
}

// ExistsByEmailAlias checks whether another active user already owns the same
// provider inbox identity.  The predicates are built from the same
// normalization rules as the service layer, so the database can return only
// the first conflicting owner instead of loading an arbitrary, truncated
// candidate set into memory.
func (r *userRepository) ExistsByEmailAlias(ctx context.Context, email string) (bool, error) {
	_, exists, err := emailAliasOwnerIDWithClient(ctx, clientFromContext(ctx, r.client), email, 0)
	return exists, err
}

func existsByEmailAliasWithClient(ctx context.Context, client *dbent.Client, email string) (bool, error) {
	_, exists, err := emailAliasOwnerIDWithClient(ctx, client, email, 0)
	return exists, err
}

func emailAliasOwnerIDWithClient(ctx context.Context, client *dbent.Client, email string, currentUserID int64) (int64, bool, error) {
	if client == nil {
		return 0, false, nil
	}
	match, ok := emailAliasMatchPredicate(email)
	if !ok {
		return 0, false, nil
	}
	otherQuery := client.User.Query().Where(match)
	if currentUserID > 0 {
		otherQuery = otherQuery.Where(dbuser.IDNEQ(currentUserID))
	}
	other, err := otherQuery.Order(dbent.Asc(dbuser.FieldID)).Select(dbuser.FieldID).First(ctx)
	if err == nil {
		return other.ID, true, nil
	}
	if !dbent.IsNotFound(err) {
		return 0, false, err
	}
	if currentUserID <= 0 {
		return 0, false, nil
	}
	selfExists, err := client.User.Query().Where(match, dbuser.IDEQ(currentUserID)).Exist(ctx)
	if err != nil {
		return 0, false, err
	}
	if selfExists {
		return currentUserID, true, nil
	}
	return 0, false, nil
}

// emailAliasMatchPredicate renders an exact SQL representation of
// NormalizeEmailForAliasDedup.  Gmail/Googlemail uses the indexed
// dot-stripped expression, constrained by an exact domain suffix; other
// providers retain significant local-part dots and only match the canonical
// address plus its plus-suffix variants.
func emailAliasMatchPredicate(email string) (predicate.User, bool) {
	identity := service.NormalizeEmailForAliasDedup(email)
	local, domain, ok := strings.Cut(identity, "@")
	if !ok || local == "" || domain == "" {
		return nil, false
	}

	if domain == "gmail.com" {
		probes := service.EmailAliasDedupProbes(email)
		if len(probes) == 0 {
			return nil, false
		}
		// The service normalizer trims every trailing root dot.  Keep the SQL
		// predicate in lock-step so a legacy `user@gmail.com..` row cannot
		// evade the alias guard when the canonical address is registered.
		domainPredicate := emailDomainSuffixPredicate("gmail.com", "googlemail.com")
		preds := make([]predicate.User, 0, len(probes)*2)
		for _, probe := range probes {
			// Keep the domain predicate alongside the indexed expression:
			// dot-stripping the whole address is intentionally broader than
			// Gmail semantics for legacy rows with unusual domain dots.
			preds = append(preds, dbuser.And(dotStrippedEmailEQ(probe.Local+"@"+probe.Domain), domainPredicate))
			// A leading '+' is intentionally not treated as a plus-address
			// separator by the service normalizer (e.g. `+alice` and
			// `+alice+tag` remain distinct identities).  Skip the suffix
			// probe in that case to avoid a false-positive lockout.
			if !strings.HasPrefix(local, "+") {
				preds = append(preds, dbuser.And(
					dotStrippedEmailLike(escapeLikeWildcards(probe.Local)+"+%@"+escapeLikeWildcards(probe.Domain)),
					domainPredicate,
				))
			}
		}
		return dbuser.Or(preds...), true
	}

	// RTRIM in normalizedEmailEQ/Like folds any number of root dots while
	// retaining significant local-part dots for non-Gmail providers.
	address := local + "@" + domain
	preds := make([]predicate.User, 0, 2)
	preds = append(preds, normalizedEmailEQ(address))
	if !strings.HasPrefix(local, "+") {
		preds = append(preds, normalizedEmailLike(
			escapeLikeWildcards(local)+"+%"+escapeLikeWildcards("@"+domain),
		))
	}
	return dbuser.Or(preds...), true
}

func emailDomainSuffixPredicate(domains ...string) predicate.User {
	return predicate.User(func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString("(")
			for i, domain := range domains {
				if i > 0 {
					b.WriteString(" OR ")
				}
				b.WriteString("RTRIM(LOWER(TRIM(").Ident(s.C(dbuser.FieldEmail)).WriteString(")), '.') LIKE ").Arg("%@" + domain)
			}
			b.WriteString(")")
		}))
	})
}

func normalizedEmailEQ(value string) predicate.User {
	return predicate.User(func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString("RTRIM(LOWER(TRIM(").Ident(s.C(dbuser.FieldEmail)).WriteString(")), '.') = ").Arg(value)
		}))
	})
}

func normalizedEmailLike(pattern string) predicate.User {
	return predicate.User(func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString("RTRIM(LOWER(TRIM(").Ident(s.C(dbuser.FieldEmail)).WriteString(")), '.') LIKE ").Arg(pattern).WriteString(` ESCAPE '\'`)
		}))
	})
}

func dotStrippedEmailExpr(b *entsql.Builder, s *entsql.Selector) *entsql.Builder {
	return b.WriteString("REPLACE(LOWER(TRIM(").Ident(s.C(dbuser.FieldEmail)).WriteString(")), '.', '')")
}

func dotStrippedEmailEQ(value string) predicate.User {
	return predicate.User(func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) { dotStrippedEmailExpr(b, s).WriteString(" = ").Arg(value) }))
	})
}

func dotStrippedEmailLike(pattern string) predicate.User {
	return predicate.User(func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			dotStrippedEmailExpr(b, s).WriteString(" LIKE ").Arg(pattern).WriteString(` ESCAPE '\'`)
		}))
	})
}

var likeWildcardEscaper = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)

func escapeLikeWildcards(value string) string { return likeWildcardEscaper.Replace(value) }

// UpdateEmailWithAliasGuard performs the primary-email update inside the
// caller's transaction, holding both exact-email and inbox-identity locks.
func (r *userRepository) UpdateEmailWithAliasGuard(ctx context.Context, userID int64, email, passwordHash string) error {
	if userID <= 0 {
		return service.ErrUserNotFound
	}
	if strings.TrimSpace(email) == "" || passwordHash == "" {
		return fmt.Errorf("email identity update requires email and password hash")
	}
	tx := dbent.TxFromContext(ctx)
	if tx == nil {
		return fmt.Errorf("email identity update requires a transaction")
	}
	client := tx.Client()
	release, err := lockRepositoryScopedKeys(
		ctx,
		client,
		txAwareSQLExecutor(ctx, r.sql, r.client),
		normalizedEmailUniquenessLockKey(email),
		emailAliasUniquenessLockKey(email),
		userEmailIdentityLockKey(userID),
	)
	if err != nil {
		return err
	}
	defer release()
	ownerID, exists, err := emailAliasOwnerIDWithClient(ctx, client, email, userID)
	if err != nil {
		return err
	}
	if exists && ownerID != userID {
		return service.ErrEmailExists
	}
	_, err = client.User.UpdateOneID(userID).
		Where(dbuser.DeletedAtIsNil()).
		SetEmail(email).
		SetPasswordHash(passwordHash).
		Save(ctx)
	return translatePersistenceError(err, service.ErrUserNotFound, service.ErrEmailExists)
}

func ensureNormalizedEmailAvailableWithClient(ctx context.Context, client *dbent.Client, userID int64, email string) error {
	client = clientFromContext(ctx, client)
	if client == nil {
		return nil
	}

	matches, err := client.User.Query().
		Where(userEmailLookupPredicate(email)).
		All(ctx)
	if err != nil {
		return err
	}
	for _, match := range matches {
		if match.ID != userID {
			return service.ErrEmailExists
		}
	}
	return nil
}

func userEmailLookupPredicate(email string) predicate.User {
	normalized := normalizeEmailLookupValue(email)
	if normalized == "" {
		return dbuser.EmailEQ(email)
	}
	return predicate.User(func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString("LOWER(TRIM(").
				Ident(s.C(dbuser.FieldEmail)).
				WriteString(")) = ").
				Arg(normalized)
		}))
	})
}

func normalizeEmailLookupValue(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizedEmailUniquenessLockKey(email string) string {
	normalized := normalizeEmailLookupValue(email)
	if normalized == "" {
		return ""
	}
	return "users:normalized-email:" + normalized
}

func emailAliasUniquenessLockKey(email string) string {
	identity := service.NormalizeEmailForAliasDedup(email)
	if identity == "" {
		return ""
	}
	return "users:email-alias-identity:" + identity
}

// userEmailIdentityLockKey serializes primary-email replacement for one user.
// It is acquired together with the target-address locks, in normalized key
// order by lockRepositoryScopedKeys. PostgreSQL gets a transaction-scoped
// advisory lock; the in-process registry covers SQLite/unit-test clients.
func userEmailIdentityLockKey(userID int64) string {
	if userID <= 0 {
		return ""
	}
	return fmt.Sprintf("users:email-identity-user:%d", userID)
}

func (r *userRepository) AddGroupToAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	client := clientFromContext(ctx, r.client)
	err := client.UserAllowedGroup.Create().
		SetUserID(userID).
		SetGroupID(groupID).
		OnConflictColumns(userallowedgroup.FieldUserID, userallowedgroup.FieldGroupID).
		DoNothing().
		Exec(ctx)
	if isSQLNoRowsError(err) {
		return nil
	}
	return err
}

func (r *userRepository) RemoveGroupFromAllowedGroups(ctx context.Context, groupID int64) (int64, error) {
	// 仅操作 user_allowed_groups 联接表，legacy users.allowed_groups 列已弃用。
	client := clientFromContext(ctx, r.client)
	affected, err := client.UserAllowedGroup.Delete().
		Where(userallowedgroup.GroupIDEQ(groupID)).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int64(affected), nil
}

// RemoveGroupFromUserAllowedGroups 移除单个用户的指定分组权限
func (r *userRepository) RemoveGroupFromUserAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.UserAllowedGroup.Delete().
		Where(userallowedgroup.UserIDEQ(userID), userallowedgroup.GroupIDEQ(groupID)).
		Exec(ctx)
	return err
}

func (r *userRepository) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.User.Query().
		Where(
			dbuser.RoleEQ(service.RoleAdmin),
			dbuser.StatusEQ(service.StatusActive),
		).
		Order(dbent.Asc(dbuser.FieldID)).
		First(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}

	out := userEntityToService(m)
	groups, err := r.loadAllowedGroups(ctx, []int64{m.ID})
	if err != nil {
		return nil, err
	}
	if v, ok := groups[m.ID]; ok {
		out.AllowedGroups = v
	}
	return out, nil
}

func (r *userRepository) loadAllowedGroups(ctx context.Context, userIDs []int64) (map[int64][]int64, error) {
	out := make(map[int64][]int64, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}

	client := clientFromContext(ctx, r.client)
	rows, err := client.UserAllowedGroup.Query().
		Where(userallowedgroup.UserIDIn(userIDs...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	for i := range rows {
		out[rows[i].UserID] = append(out[rows[i].UserID], rows[i].GroupID)
	}

	for userID := range out {
		sort.Slice(out[userID], func(i, j int) bool { return out[userID][i] < out[userID][j] })
	}

	return out, nil
}

// syncUserAllowedGroupsWithClient 在 ent client/事务内同步用户允许分组：
// 仅操作 user_allowed_groups 联接表，legacy users.allowed_groups 列已弃用。
func (r *userRepository) syncUserAllowedGroupsWithClient(ctx context.Context, client *dbent.Client, userID int64, groupIDs []int64) error {
	if client == nil {
		return nil
	}

	// Keep join table as the source of truth for reads.
	if _, err := client.UserAllowedGroup.Delete().Where(userallowedgroup.UserIDEQ(userID)).Exec(ctx); err != nil {
		return err
	}

	unique := make(map[int64]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		if id <= 0 {
			continue
		}
		unique[id] = struct{}{}
	}

	if len(unique) > 0 {
		creates := make([]*dbent.UserAllowedGroupCreate, 0, len(unique))
		for groupID := range unique {
			creates = append(creates, client.UserAllowedGroup.Create().SetUserID(userID).SetGroupID(groupID))
		}
		if err := client.UserAllowedGroup.
			CreateBulk(creates...).
			OnConflictColumns(userallowedgroup.FieldUserID, userallowedgroup.FieldGroupID).
			DoNothing().
			Exec(ctx); err != nil {
			if isSQLNoRowsError(err) {
				return nil
			}
			return err
		}
	}

	return nil
}

func applyUserEntityToService(dst *service.User, src *dbent.User) {
	if dst == nil || src == nil {
		return
	}
	dst.ID = src.ID
	dst.SignupSource = src.SignupSource
	dst.LastLoginAt = src.LastLoginAt
	dst.LastActiveAt = src.LastActiveAt
	dst.CreatedAt = src.CreatedAt
	dst.UpdatedAt = src.UpdatedAt
}

func userSignupSourceOrDefault(signupSource string) string {
	switch strings.TrimSpace(strings.ToLower(signupSource)) {
	case "", "email":
		return "email"
	case "linuxdo", "wechat", "oidc", "dingtalk":
		return strings.TrimSpace(strings.ToLower(signupSource))
	default:
		return "email"
	}
}

// marshalExtraEmails serializes notify email entries to JSON for storage.
func marshalExtraEmails(entries []service.NotifyEmailEntry) string {
	return service.MarshalNotifyEmails(entries)
}

// UpdateTotpSecret 更新用户的 TOTP 加密密钥
func (r *userRepository) UpdateTotpSecret(ctx context.Context, userID int64, encryptedSecret *string) error {
	client := clientFromContext(ctx, r.client)
	update := client.User.UpdateOneID(userID).Where(dbuser.DeletedAtIsNil())
	if encryptedSecret == nil {
		update = update.ClearTotpSecretEncrypted()
	} else {
		update = update.SetTotpSecretEncrypted(*encryptedSecret)
	}
	_, err := update.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	return nil
}

// EnableTotp 启用用户的 TOTP 双因素认证
func (r *userRepository) EnableTotp(ctx context.Context, userID int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.User.UpdateOneID(userID).
		Where(dbuser.DeletedAtIsNil()).
		SetTotpEnabled(true).
		SetTotpEnabledAt(time.Now()).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	return nil
}

// DisableTotp 禁用用户的 TOTP 双因素认证
func (r *userRepository) DisableTotp(ctx context.Context, userID int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.User.UpdateOneID(userID).
		Where(dbuser.DeletedAtIsNil()).
		SetTotpEnabled(false).
		ClearTotpEnabledAt().
		ClearTotpSecretEncrypted().
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	return nil
}
