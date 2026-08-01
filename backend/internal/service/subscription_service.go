package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/dgraph-io/ristretto"
	"golang.org/x/sync/singleflight"
)

// MaxExpiresAt is the maximum allowed expiration date (year 2099)
// This prevents time.Time JSON serialization errors (RFC 3339 requires year <= 9999)
var MaxExpiresAt = time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

// MaxValidityDays is the maximum allowed validity days for subscriptions (100 years)
const MaxValidityDays = 36500

var (
	ErrSubscriptionNotFound        = infraerrors.NotFound("SUBSCRIPTION_NOT_FOUND", "subscription not found")
	ErrSubscriptionExpired         = infraerrors.Forbidden("SUBSCRIPTION_EXPIRED", "subscription has expired")
	ErrSubscriptionSuspended       = infraerrors.Forbidden("SUBSCRIPTION_SUSPENDED", "subscription is suspended")
	ErrSubscriptionAlreadyExists   = infraerrors.Conflict("SUBSCRIPTION_ALREADY_EXISTS", "subscription already exists for this user and group")
	ErrSubscriptionAssignConflict  = infraerrors.Conflict("SUBSCRIPTION_ASSIGN_CONFLICT", "subscription exists but request conflicts with existing assignment semantics")
	ErrSubscriptionNotRevoked      = infraerrors.Conflict("SUBSCRIPTION_NOT_REVOKED", "subscription is not revoked")
	ErrSubscriptionRestoreConflict = infraerrors.Conflict("SUBSCRIPTION_RESTORE_CONFLICT", "subscription already exists for this user and group")
	ErrGroupNotSubscriptionType    = infraerrors.BadRequest("GROUP_NOT_SUBSCRIPTION_TYPE", "group is not a subscription type")
	ErrInvalidInput                = infraerrors.BadRequest("INVALID_INPUT", "at least one of resetDaily, resetWeekly, or resetMonthly must be true")
	ErrDailyLimitExceeded          = infraerrors.TooManyRequests("DAILY_LIMIT_EXCEEDED", "daily usage limit exceeded")
	ErrWeeklyLimitExceeded         = infraerrors.TooManyRequests("WEEKLY_LIMIT_EXCEEDED", "weekly usage limit exceeded")
	ErrMonthlyLimitExceeded        = infraerrors.TooManyRequests("MONTHLY_LIMIT_EXCEEDED", "monthly usage limit exceeded")
	ErrSubscriptionNilInput        = infraerrors.BadRequest("SUBSCRIPTION_NIL_INPUT", "subscription input cannot be nil")
	ErrAdjustWouldExpire           = infraerrors.BadRequest("ADJUST_WOULD_EXPIRE", "adjustment would result in expired subscription (remaining days must be > 0)")
)

// SubscriptionService 订阅服务
type SubscriptionService struct {
	groupRepo           GroupRepository
	userSubRepo         UserSubscriptionRepository
	billingCacheService *BillingCacheService
	entClient           *dbent.Client
	cfg                 *config.Config

	// L1 缓存：加速中间件热路径的订阅查询
	subCacheL1     *ristretto.Cache
	subCacheGroup  singleflight.Group
	subCacheTTL    time.Duration
	subCacheJitter int // 抖动百分比

	maintenanceQueue *SubscriptionMaintenanceQueue
	subscriberMu     sync.Mutex
	subscriberCancel context.CancelFunc
	subscriberWG     sync.WaitGroup
	startOnce        sync.Once
	stopOnce         sync.Once
	cacheCloseOnce   sync.Once
}

// NewSubscriptionService 创建订阅服务
func NewSubscriptionService(groupRepo GroupRepository, userSubRepo UserSubscriptionRepository, billingCacheService *BillingCacheService, entClient *dbent.Client, cfg *config.Config) *SubscriptionService {
	svc := &SubscriptionService{
		groupRepo:           groupRepo,
		userSubRepo:         userSubRepo,
		billingCacheService: billingCacheService,
		entClient:           entClient,
		cfg:                 cfg,
	}
	svc.initMaintenanceQueue(cfg)
	return svc
}

func (s *SubscriptionService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		s.initSubCache(s.cfg)
		if s.maintenanceQueue != nil {
			s.maintenanceQueue.Start()
		}
		s.StartSubCacheInvalidationSubscriber(context.Background())
	})
}

func (s *SubscriptionService) initMaintenanceQueue(cfg *config.Config) {
	if cfg == nil {
		return
	}
	mc := cfg.SubscriptionMaintenance
	if mc.WorkerCount <= 0 || mc.QueueSize <= 0 {
		return
	}
	s.maintenanceQueue = NewSubscriptionMaintenanceQueue(mc.WorkerCount, mc.QueueSize)
}

// Stop stops the maintenance worker pool.
func (s *SubscriptionService) Stop() {
	_ = s.StopContext(context.Background())
}

func (s *SubscriptionService) StopContext(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.stopOnce.Do(func() {
		s.subscriberMu.Lock()
		cancel := s.subscriberCancel
		s.subscriberCancel = nil
		s.subscriberMu.Unlock()
		if cancel != nil {
			cancel()
		}
	})
	if s.maintenanceQueue != nil {
		if err := s.maintenanceQueue.StopContext(ctx); err != nil {
			return err
		}
	}
	done := make(chan struct{})
	go func() {
		s.subscriberWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		s.cacheCloseOnce.Do(func() {
			if s.subCacheL1 != nil {
				s.subCacheL1.Close()
			}
		})
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// initSubCache 初始化订阅 L1 缓存
func (s *SubscriptionService) initSubCache(cfg *config.Config) {
	if cfg == nil {
		return
	}
	sc := cfg.SubscriptionCache
	if sc.L1Size <= 0 || sc.L1TTLSeconds <= 0 {
		return
	}
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(sc.L1Size) * 10,
		MaxCost:     int64(sc.L1Size),
		BufferItems: 64,
	})
	if err != nil {
		log.Printf("Warning: failed to init subscription L1 cache: %v", err)
		return
	}
	s.subCacheL1 = cache
	s.subCacheTTL = time.Duration(sc.L1TTLSeconds) * time.Second
	s.subCacheJitter = sc.JitterPercent
}

// subCacheKey 生成订阅缓存 key（热路径，避免 fmt.Sprintf 开销）
func subCacheKey(userID, groupID int64) string {
	return "sub:" + strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(groupID, 10)
}

// jitteredTTL 为 TTL 添加抖动，避免集中过期
func (s *SubscriptionService) jitteredTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 || s.subCacheJitter <= 0 {
		return ttl
	}
	pct := s.subCacheJitter
	if pct > 100 {
		pct = 100
	}
	delta := float64(pct) / 100
	factor := 1 - delta + rand.Float64()*(2*delta)
	if factor <= 0 {
		return ttl
	}
	return time.Duration(float64(ttl) * factor)
}

// InvalidateSubCache 失效指定用户+分组的订阅 L1 缓存
func (s *SubscriptionService) InvalidateSubCache(userID, groupID int64) {
	if s.subCacheL1 == nil {
		return
	}
	s.subCacheL1.Del(subCacheKey(userID, groupID))
}

// InvalidateSubCacheSync 失效订阅 L1 缓存并等待 Ristretto 删除操作生效。
func (s *SubscriptionService) InvalidateSubCacheSync(userID, groupID int64) {
	s.invalidateSubCacheKeySync(subCacheKey(userID, groupID))
}

func (s *SubscriptionService) invalidateSubCacheKeySync(key string) {
	if s.subCacheL1 == nil {
		return
	}
	s.subCacheL1.Del(key)
	s.subCacheL1.Wait()
}

// StartSubCacheInvalidationSubscriber 启动跨实例订阅 L1 缓存失效订阅。
func (s *SubscriptionService) StartSubCacheInvalidationSubscriber(ctx context.Context) {
	if s.billingCacheService == nil || s.subCacheL1 == nil {
		return
	}
	if ctx != nil && ctx.Err() != nil {
		return
	}
	s.subscriberMu.Lock()
	if s.subscriberCancel != nil {
		s.subscriberMu.Unlock()
		return
	}
	subscriberCtx, cancel := context.WithCancel(context.Background())
	s.subscriberCancel = cancel
	s.subscriberWG.Add(1)
	s.subscriberMu.Unlock()
	go func() {
		defer s.subscriberWG.Done()
		if err := s.billingCacheService.SubscribeSubscriptionCacheInvalidation(subscriberCtx, func(cacheKey string) {
			s.invalidateSubCacheKeySync(cacheKey)
		}); err != nil && subscriberCtx.Err() == nil {
			log.Printf("Warning: failed to start subscription cache invalidation subscriber: %v", err)
		}
	}()
}

func (s *SubscriptionService) invalidateSubscriptionCaches(userID, groupID int64) error {
	s.InvalidateSubCacheSync(userID, groupID)
	if s.billingCacheService == nil {
		return nil
	}

	cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID); err != nil {
		return fmt.Errorf("invalidate billing subscription cache: %w", err)
	}
	if err := s.billingCacheService.PublishSubscriptionCacheInvalidation(cacheCtx, subCacheKey(userID, groupID)); err != nil {
		return fmt.Errorf("publish subscription cache invalidation: %w", err)
	}
	return nil
}

// AssignSubscriptionInput 分配订阅输入
type AssignSubscriptionInput struct {
	UserID       int64
	GroupID      int64
	ValidityDays int
	AssignedBy   int64
	Notes        string
}

// AssignSubscription 分配订阅给用户（不允许重复分配）
func (s *SubscriptionService) AssignSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, error) {
	sub, _, err := s.assignSubscriptionWithReuse(ctx, input)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// AssignOrExtendSubscription 分配或续期订阅（用于兑换码等场景）
// 如果用户已有同分组的订阅：
//   - 未过期：从当前过期时间累加天数
//   - 已过期：从当前时间开始计算新的过期时间，并激活订阅
//
// 如果没有订阅：创建新订阅
func (s *SubscriptionService) AssignOrExtendSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	return s.assignOrExtendSubscription(ctx, input, false)
}

func (s *SubscriptionService) assignOrExtendSubscription(ctx context.Context, input *AssignSubscriptionInput, deferCacheInvalidation bool) (*UserSubscription, bool, error) {
	// 检查分组是否存在且为订阅类型
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, false, fmt.Errorf("group not found: %w", err)
	}
	if !group.IsSubscriptionType() {
		return nil, false, ErrGroupNotSubscriptionType
	}

	// 查询是否已有订阅
	existingSub, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
	if err != nil {
		if !errors.Is(err, ErrSubscriptionNotFound) {
			return nil, false, fmt.Errorf("get existing subscription: %w", err)
		}
		// 不存在记录是正常情况，后续创建新订阅。
		existingSub = nil
	}

	validityDays := input.ValidityDays
	if validityDays <= 0 {
		validityDays = 30
	}
	if validityDays > MaxValidityDays {
		validityDays = MaxValidityDays
	}

	// 已有订阅，执行续期（在事务中完成所有更新）
	if existingSub != nil {
		updatedSub, err := s.updateExistingSubscriptionTerm(ctx, existingSub.ID, input.Notes, validityDays, true)
		if err != nil {
			return nil, false, err
		}

		// 失效订阅缓存
		s.maybeInvalidateAssignmentCaches(input.UserID, input.GroupID, deferCacheInvalidation)

		return updatedSub, true, nil // true 表示是续期
	}

	// 没有订阅，创建新订阅
	sub, err := s.createSubscription(ctx, input)
	if err != nil {
		return nil, false, err
	}

	// 失效订阅缓存
	s.maybeInvalidateAssignmentCaches(input.UserID, input.GroupID, deferCacheInvalidation)

	return sub, false, nil // false 表示是新建
}

func (s *SubscriptionService) maybeInvalidateAssignmentCaches(userID, groupID int64, deferred bool) {
	// Payment fulfillment owns an outer transaction and performs a synchronous
	// invalidation after commit. Invalidating inside that transaction can reload
	// the pre-commit subscription into cache.
	if deferred {
		return
	}

	s.InvalidateSubCache(userID, groupID)
	if s.billingCacheService != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}
}

func (s *SubscriptionService) updateExistingSubscriptionTerm(
	ctx context.Context,
	subscriptionID int64,
	notes string,
	validityDays int,
	reactivateActive bool,
) (*UserSubscription, error) {
	var updatedSub *UserSubscription
	err := s.withSubscriptionUpdateTx(ctx, func(txCtx context.Context) error {
		// The row lock is held until this transaction commits. Every renewal must
		// recompute from this locked snapshot so concurrent requests accumulate
		// their full validity instead of overwriting one another.
		existingSub, err := s.getSubscriptionByIDForUpdate(txCtx, subscriptionID)
		if err != nil {
			return err
		}

		now := time.Now()
		isExpired := !existingSub.ExpiresAt.After(now)
		if isExpired && validityDays < 0 {
			return infraerrors.BadRequest("CANNOT_SHORTEN_EXPIRED", "cannot shorten an expired subscription")
		}

		var newExpiresAt time.Time
		if isExpired {
			newExpiresAt = now.AddDate(0, 0, validityDays)
		} else {
			newExpiresAt = existingSub.ExpiresAt.AddDate(0, 0, validityDays)
		}
		if newExpiresAt.After(MaxExpiresAt) {
			newExpiresAt = MaxExpiresAt
		}
		if !newExpiresAt.After(now) {
			return ErrAdjustWouldExpire
		}

		if isExpired {
			// An interrupted renewal starts a new term at the exact instant observed
			// while holding the row lock. starts_at remains immutable thereafter.
			renewed := renewedSubscriptionTerm(existingSub, notes, now, newExpiresAt)
			if err := s.userSubRepo.Update(txCtx, renewed); err != nil {
				return fmt.Errorf("renew expired subscription: %w", err)
			}
		} else {
			// An uninterrupted renewal changes expiry only and preserves starts_at,
			// window markers, and accumulated usage from the locked snapshot.
			if err := s.userSubRepo.ExtendExpiry(txCtx, existingSub.ID, newExpiresAt); err != nil {
				return fmt.Errorf("extend subscription: %w", err)
			}

			if reactivateActive && existingSub.Status != SubscriptionStatusActive {
				if err := s.userSubRepo.UpdateStatus(txCtx, existingSub.ID, SubscriptionStatusActive); err != nil {
					return fmt.Errorf("update subscription status: %w", err)
				}
			}

			if notes != "" {
				if err := s.userSubRepo.UpdateNotes(txCtx, existingSub.ID, appendSubscriptionNotes(existingSub.Notes, notes)); err != nil {
					return fmt.Errorf("update subscription notes: %w", err)
				}
			}
		}

		updatedSub, err = s.userSubRepo.GetByID(txCtx, existingSub.ID)
		if err != nil {
			return fmt.Errorf("reload updated subscription: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updatedSub, nil
}

func (s *SubscriptionService) getSubscriptionByIDForUpdate(ctx context.Context, subscriptionID int64) (*UserSubscription, error) {
	if lockingRepo, ok := s.userSubRepo.(userSubscriptionLockingRepository); ok {
		return lockingRepo.GetByIDForUpdate(ctx, subscriptionID)
	}
	// Compatibility path for alternate repositories and existing unit stubs.
	// Production uses userSubscriptionRepository, which implements the row lock.
	return s.userSubRepo.GetByID(ctx, subscriptionID)
}

func (s *SubscriptionService) withSubscriptionUpdateTx(ctx context.Context, fn func(context.Context) error) error {
	if dbent.TxFromContext(ctx) != nil {
		return fn(ctx)
	}
	if s.entClient == nil {
		return fn(ctx)
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func renewedSubscriptionTerm(existingSub *UserSubscription, notes string, startsAt, expiresAt time.Time) *UserSubscription {
	renewed := *existingSub
	windowStart := startsAt
	renewed.StartsAt = startsAt
	renewed.ExpiresAt = expiresAt
	renewed.Status = SubscriptionStatusActive
	renewed.WeeklyWindowStart = &windowStart
	renewed.MonthlyWindowStart = &windowStart
	renewed.DailyWindowStart = nil
	renewed.DailyUsageUSD = 0
	renewed.WeeklyUsageUSD = 0
	renewed.MonthlyUsageUSD = 0
	renewed.Notes = appendSubscriptionNotes(existingSub.Notes, notes)
	return &renewed
}

func appendSubscriptionNotes(existingNotes, newNotes string) string {
	if newNotes == "" {
		return existingNotes
	}
	if existingNotes == "" {
		return newNotes
	}
	return existingNotes + "\n" + newNotes
}

// createSubscription 创建新订阅（内部方法）
func (s *SubscriptionService) createSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, error) {
	validityDays := input.ValidityDays
	if validityDays <= 0 {
		validityDays = 30
	}
	if validityDays > MaxValidityDays {
		validityDays = MaxValidityDays
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, validityDays)
	if expiresAt.After(MaxExpiresAt) {
		expiresAt = MaxExpiresAt
	}

	sub := &UserSubscription{
		UserID:             input.UserID,
		GroupID:            input.GroupID,
		StartsAt:           now,
		ExpiresAt:          expiresAt,
		Status:             SubscriptionStatusActive,
		WeeklyWindowStart:  &now,
		MonthlyWindowStart: &now,
		AssignedAt:         now,
		Notes:              input.Notes,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	// 只有当 AssignedBy > 0 时才设置（0 表示系统分配，如兑换码）
	if input.AssignedBy > 0 {
		sub.AssignedBy = &input.AssignedBy
	}

	if err := s.userSubRepo.Create(ctx, sub); err != nil {
		return nil, err
	}

	// 重新获取完整订阅信息（包含关联）
	return s.userSubRepo.GetByID(ctx, sub.ID)
}

// BulkAssignSubscriptionInput 批量分配订阅输入
type BulkAssignSubscriptionInput struct {
	UserIDs      []int64
	GroupID      int64
	ValidityDays int
	AssignedBy   int64
	Notes        string
}

// BulkAssignResult 批量分配结果
type BulkAssignResult struct {
	SuccessCount  int
	CreatedCount  int
	ReusedCount   int
	FailedCount   int
	Subscriptions []UserSubscription
	Errors        []string
	Statuses      map[int64]string
}

// BulkAssignSubscription 批量分配订阅
func (s *SubscriptionService) BulkAssignSubscription(ctx context.Context, input *BulkAssignSubscriptionInput) (*BulkAssignResult, error) {
	result := &BulkAssignResult{
		Subscriptions: make([]UserSubscription, 0),
		Errors:        make([]string, 0),
		Statuses:      make(map[int64]string),
	}

	for _, userID := range input.UserIDs {
		sub, reused, err := s.assignSubscriptionWithReuse(ctx, &AssignSubscriptionInput{
			UserID:       userID,
			GroupID:      input.GroupID,
			ValidityDays: input.ValidityDays,
			AssignedBy:   input.AssignedBy,
			Notes:        input.Notes,
		})
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("user %d: %v", userID, err))
			result.Statuses[userID] = "failed"
		} else {
			result.SuccessCount++
			result.Subscriptions = append(result.Subscriptions, *sub)
			if reused {
				result.ReusedCount++
				result.Statuses[userID] = "reused"
			} else {
				result.CreatedCount++
				result.Statuses[userID] = "created"
			}
		}
	}

	return result, nil
}

func (s *SubscriptionService) assignSubscriptionWithReuse(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	// 检查分组是否存在且为订阅类型
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, false, fmt.Errorf("group not found: %w", err)
	}
	if !group.IsSubscriptionType() {
		return nil, false, ErrGroupNotSubscriptionType
	}

	// 检查是否已存在订阅；若已存在，则按幂等成功返回现有订阅
	exists, err := s.userSubRepo.ExistsByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
	if err != nil {
		return nil, false, err
	}
	if exists {
		sub, getErr := s.userSubRepo.GetByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
		if getErr != nil {
			return nil, false, getErr
		}
		if conflictReason, conflict := detectAssignSemanticConflict(sub, input); conflict {
			return nil, false, ErrSubscriptionAssignConflict.WithMetadata(map[string]string{
				"conflict_reason": conflictReason,
			})
		}
		return sub, true, nil
	}

	sub, err := s.createSubscription(ctx, input)
	if err != nil {
		return nil, false, err
	}

	// 失效订阅缓存
	s.InvalidateSubCache(input.UserID, input.GroupID)
	if s.billingCacheService != nil {
		userID, groupID := input.UserID, input.GroupID
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}

	return sub, false, nil
}

func detectAssignSemanticConflict(existing *UserSubscription, input *AssignSubscriptionInput) (string, bool) {
	if existing == nil || input == nil {
		return "", false
	}

	normalizedDays := normalizeAssignValidityDays(input.ValidityDays)
	if !existing.StartsAt.IsZero() {
		expectedExpiresAt := existing.StartsAt.AddDate(0, 0, normalizedDays)
		if expectedExpiresAt.After(MaxExpiresAt) {
			expectedExpiresAt = MaxExpiresAt
		}
		if !existing.ExpiresAt.Equal(expectedExpiresAt) {
			return "validity_days_mismatch", true
		}
	}

	existingNotes := strings.TrimSpace(existing.Notes)
	inputNotes := strings.TrimSpace(input.Notes)
	if existingNotes != inputNotes {
		return "notes_mismatch", true
	}

	return "", false
}

func normalizeAssignValidityDays(days int) int {
	if days <= 0 {
		days = 30
	}
	if days > MaxValidityDays {
		days = MaxValidityDays
	}
	return days
}

// RevokeSubscription 撤销订阅
func (s *SubscriptionService) RevokeSubscription(ctx context.Context, subscriptionID int64) error {
	// 先获取订阅信息用于失效缓存
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if err := s.userSubRepo.Delete(ctx, subscriptionID); err != nil {
		return err
	}

	if err := s.invalidateSubscriptionCaches(sub.UserID, sub.GroupID); err != nil {
		return err
	}

	return nil
}

// RestoreSubscription 恢复已撤销订阅
func (s *SubscriptionService) RestoreSubscription(ctx context.Context, subscriptionID int64) (*UserSubscription, error) {
	sub, err := s.userSubRepo.GetByIDIncludeDeleted(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub.DeletedAt == nil {
		return nil, ErrSubscriptionNotRevoked
	}

	exists, err := s.userSubRepo.ExistsActiveByUserIDAndGroupID(ctx, sub.UserID, sub.GroupID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSubscriptionRestoreConflict
	}

	restoredStatus := sub.Status
	now := time.Now()
	if restoredStatus == SubscriptionStatusActive && !sub.ExpiresAt.After(now) {
		restoredStatus = SubscriptionStatusExpired
	}

	restored, err := s.userSubRepo.Restore(ctx, subscriptionID, restoredStatus)
	if err != nil {
		return nil, err
	}

	if err := s.invalidateSubscriptionCaches(restored.UserID, restored.GroupID); err != nil {
		return nil, err
	}
	return restored, nil
}

// ExtendSubscription 调整订阅时长（正数延长，负数缩短）
func (s *SubscriptionService) ExtendSubscription(ctx context.Context, subscriptionID int64, days int) (*UserSubscription, error) {
	// 限制调整天数范围
	if days > MaxValidityDays {
		days = MaxValidityDays
	}
	if days < -MaxValidityDays {
		days = -MaxValidityDays
	}

	updatedSub, err := s.updateExistingSubscriptionTerm(ctx, subscriptionID, "", days, false)
	if err != nil {
		return nil, err
	}

	// 失效订阅缓存
	s.InvalidateSubCache(updatedSub.UserID, updatedSub.GroupID)
	if s.billingCacheService != nil {
		userID, groupID := updatedSub.UserID, updatedSub.GroupID
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}

	return updatedSub, nil
}

// GetByID 根据ID获取订阅
func (s *SubscriptionService) GetByID(ctx context.Context, id int64) (*UserSubscription, error) {
	return s.userSubRepo.GetByID(ctx, id)
}

// GetActiveSubscription 获取用户对特定分组的有效订阅
// 使用 L1 缓存 + singleflight 加速中间件热路径。
// 返回缓存对象的浅拷贝，调用方可安全修改字段而不会污染缓存或触发 data race。
func (s *SubscriptionService) GetActiveSubscription(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	key := subCacheKey(userID, groupID)

	// L1 缓存命中：返回浅拷贝
	if s.subCacheL1 != nil {
		if v, ok := s.subCacheL1.Get(key); ok {
			if sub, ok := v.(*UserSubscription); ok {
				cp := *sub
				return &cp, nil
			}
		}
	}

	// singleflight 防止并发击穿
	value, err, _ := s.subCacheGroup.Do(key, func() (any, error) {
		sub, err := s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, userID, groupID)
		if err != nil {
			return nil, err // 直接透传 repo 已翻译的错误（NotFound → ErrSubscriptionNotFound，其他错误原样返回）
		}
		// 写入 L1 缓存
		if s.subCacheL1 != nil {
			_ = s.subCacheL1.SetWithTTL(key, sub, 1, s.jitteredTTL(s.subCacheTTL))
		}
		return sub, nil
	})
	if err != nil {
		return nil, err
	}
	// singleflight 返回的也是缓存指针，需要浅拷贝
	sub, ok := value.(*UserSubscription)
	if !ok || sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	cp := *sub
	return &cp, nil
}

// ListUserSubscriptions 获取用户的所有订阅
func (s *SubscriptionService) ListUserSubscriptions(ctx context.Context, userID int64) ([]UserSubscription, error) {
	subs, err := s.userSubRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	normalizeExpiredWindows(subs)
	normalizeSubscriptionStatus(subs)
	return subs, nil
}

// ListActiveUserSubscriptions 获取用户的所有有效订阅
func (s *SubscriptionService) ListActiveUserSubscriptions(ctx context.Context, userID int64) ([]UserSubscription, error) {
	subs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	normalizeExpiredWindows(subs)
	return subs, nil
}

// ListGroupSubscriptions 获取分组的所有订阅
func (s *SubscriptionService) ListGroupSubscriptions(ctx context.Context, groupID int64, page, pageSize int) ([]UserSubscription, *pagination.PaginationResult, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	subs, pag, err := s.userSubRepo.ListByGroupID(ctx, groupID, params)
	if err != nil {
		return nil, nil, err
	}
	normalizeExpiredWindows(subs)
	normalizeSubscriptionStatus(subs)
	return subs, pag, nil
}

// List 获取所有订阅（分页，支持筛选和排序）
func (s *SubscriptionService) List(ctx context.Context, page, pageSize int, userID, groupID *int64, status, platform, sortBy, sortOrder string) ([]UserSubscription, *pagination.PaginationResult, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	subs, pag, err := s.userSubRepo.List(ctx, params, userID, groupID, status, platform, sortBy, sortOrder)
	if err != nil {
		return nil, nil, err
	}
	normalizeExpiredWindows(subs)
	normalizeSubscriptionStatus(subs)
	return subs, pag, nil
}

// normalizeExpiredWindows 将已过期窗口的数据清零（仅影响返回数据，不影响数据库）
// 这确保前端显示正确的当前窗口状态，而不是过期窗口的历史数据
func normalizeExpiredWindows(subs []UserSubscription) {
	now := time.Now()
	for i := range subs {
		sub := &subs[i]
		// Daily is legacy-only. Weekly and monthly counters are projected into
		// their current starts_at-anchored periods.
		sub.DailyWindowStart = nil
		sub.DailyUsageUSD = 0
		if expectedStart, _, ok := sub.WeeklyWindowAt(now); ok && sub.NeedsWeeklyResetAt(now) {
			sub.WeeklyWindowStart = &expectedStart
			sub.WeeklyUsageUSD = 0
		}
		if expectedStart, _, ok := sub.MonthlyWindowAt(now); ok && sub.NeedsMonthlyResetAt(now) {
			sub.MonthlyWindowStart = &expectedStart
			sub.MonthlyUsageUSD = 0
		}
	}
}

// normalizeSubscriptionStatus 根据实际过期时间修正状态（仅影响返回数据，不影响数据库）
// 这确保前端显示正确的状态，即使定时任务尚未更新数据库
func normalizeSubscriptionStatus(subs []UserSubscription) {
	now := time.Now()
	for i := range subs {
		sub := &subs[i]
		if sub.Status == SubscriptionStatusActive && !sub.ExpiresAt.After(now) {
			sub.Status = SubscriptionStatusExpired
		}
	}
}

// startOfDay 返回给定时间所在日期的零点（保持原时区）
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// CheckAndActivateWindow 检查并激活窗口（首次使用时）
func (s *SubscriptionService) CheckAndActivateWindow(ctx context.Context, sub *UserSubscription) error {
	// Activation is a reset, not a marker-only relabel. Reusing the legacy
	// ActivateWindows update could attach an elapsed counter to the new period.
	return s.CheckAndResetWindows(ctx, sub)
}

// AdminResetQuota manually clears selected counters. A weekly reset never
// changes the anchored period boundary.
func (s *SubscriptionService) AdminResetQuota(ctx context.Context, subscriptionID int64, resetDaily, resetWeekly, resetMonthly bool) (*UserSubscription, error) {
	if !resetDaily && !resetWeekly && !resetMonthly {
		return nil, ErrInvalidInput
	}
	var sub *UserSubscription
	err := s.withSubscriptionUpdateTx(ctx, func(txCtx context.Context) error {
		// Read starts_at and all current window markers only after acquiring the
		// row lock. This prevents an admin reset from clearing usage that was
		// already moved into a newer term/window by a concurrent request.
		lockedSub, err := s.getSubscriptionByIDForUpdate(txCtx, subscriptionID)
		if err != nil {
			return err
		}
		sub = lockedSub

		now := time.Now()
		weeklyWindowStart, _, ok := sub.WeeklyWindowAt(now)
		if !ok {
			return ErrSubscriptionInvalid
		}
		monthlyWindowStart, _, ok := sub.MonthlyWindowAt(now)
		if !ok {
			return ErrSubscriptionInvalid
		}

		if resetDaily || resetWeekly {
			if err := s.userSubRepo.ResetUsageWindows(
				txCtx,
				sub.ID,
				resetDaily,
				resetWeekly,
				false,
				weeklyWindowStart,
			); err != nil {
				return err
			}
		}
		if resetMonthly {
			// The row lock makes a conditional marker CAS redundant. Use the locked
			// row update so a stale CAS cannot be translated into false success.
			if err := s.userSubRepo.ResetUsageWindows(txCtx, sub.ID, false, false, true, monthlyWindowStart); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Invalidate L1 ristretto cache. Ristretto's Del() is asynchronous by design,
	// so call Wait() immediately after to flush pending operations and guarantee
	// the deleted key is not returned on the very next Get() call.
	s.InvalidateSubCacheSync(sub.UserID, sub.GroupID)
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, sub.GroupID)
	}
	// Return the refreshed subscription from DB
	return s.userSubRepo.GetByID(ctx, subscriptionID)
}

// CheckAndResetWindows 检查并重置过期的窗口
func (s *SubscriptionService) CheckAndResetWindows(ctx context.Context, sub *UserSubscription) error {
	now := time.Now()
	weeklyWindowStart, _, ok := sub.WeeklyWindowAt(now)
	if !ok {
		return ErrSubscriptionInvalid
	}
	monthlyWindowStart, _, ok := sub.MonthlyWindowAt(now)
	if !ok {
		return ErrSubscriptionInvalid
	}
	resetWeekly := sub.NeedsWeeklyResetAt(now)
	resetMonthly := sub.NeedsMonthlyResetAt(now)
	wroteWindow := false

	// The current periods are always derived from StartsAt. Advance both
	// markers in one transaction so a monthly failure cannot commit only the
	// weekly reset. A delayed request may skip any number of periods without
	// shifting either anchor.
	err := s.withSubscriptionUpdateTx(ctx, func(txCtx context.Context) error {
		if resetWeekly {
			expectedWindowStart := sub.WeeklyWindowStart
			if err := s.userSubRepo.ResetWeeklyUsage(txCtx, sub.ID, expectedWindowStart, weeklyWindowStart); err != nil {
				return err
			}
			wroteWindow = true
		}
		if resetMonthly {
			expectedWindowStart := sub.MonthlyWindowStart
			if err := s.userSubRepo.ResetMonthlyUsage(txCtx, sub.ID, expectedWindowStart, monthlyWindowStart); err != nil {
				return err
			}
			wroteWindow = true
		}
		return nil
	})
	if err != nil {
		// Production writes roll back with the transaction; invalidating is
		// still necessary for alternate repository implementations that may
		// have committed a successful first write.
		if resetWeekly || resetMonthly {
			s.InvalidateSubCache(sub.UserID, sub.GroupID)
			if s.billingCacheService != nil {
				_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, sub.GroupID)
			}
		}
		return err
	}
	if resetWeekly {
		sub.WeeklyWindowStart = &weeklyWindowStart
		sub.WeeklyUsageUSD = 0
	}
	if resetMonthly {
		sub.MonthlyWindowStart = &monthlyWindowStart
		sub.MonthlyUsageUSD = 0
	}

	// 如果有窗口被重置，失效缓存以保持一致性
	if wroteWindow {
		s.InvalidateSubCache(sub.UserID, sub.GroupID)
		if s.billingCacheService != nil {
			_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, sub.GroupID)
		}
	}

	return nil
}

// EnsureWindowMaintenance advances expired usage windows before a request is
// allowed to proceed. It returns a fresh database snapshot because a competing
// request may have won one of the conditional resets.
func (s *SubscriptionService) EnsureWindowMaintenance(ctx context.Context, sub *UserSubscription) (*UserSubscription, error) {
	if sub == nil {
		return nil, ErrSubscriptionNilInput
	}
	if err := s.CheckAndResetWindows(ctx, sub); err != nil {
		return nil, err
	}

	// GetByID bypasses the service caches. This prevents a stale loser of the
	// CAS from validating limits against zeroed in-memory usage.
	refreshed, err := s.userSubRepo.GetByID(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	s.InvalidateSubCacheSync(sub.UserID, sub.GroupID)
	return refreshed, nil
}

// CheckUsageLimits 检查使用限额（返回错误如果超限）
// 用于中间件的快速预检查，additionalCost 通常为 0
func (s *SubscriptionService) CheckUsageLimits(ctx context.Context, sub *UserSubscription, group *Group, additionalCost float64) error {
	now := time.Now()
	if !sub.CheckWeeklyLimitAt(group, additionalCost, now) {
		return ErrWeeklyLimitExceeded
	}
	if !sub.CheckMonthlyLimitAt(group, additionalCost, now) {
		return ErrMonthlyLimitExceeded
	}
	return nil
}

// ValidateAndCheckLimits 合并验证+限额检查（中间件热路径专用）
// 仅做内存检查，不触发 DB 写入。调用方必须在放行请求前同步完成窗口维护。
// 返回 needsMaintenance 表示是否需要执行窗口维护并回读数据库快照。
func (s *SubscriptionService) ValidateAndCheckLimits(sub *UserSubscription, group *Group) (needsMaintenance bool, err error) {
	now := time.Now()
	// 1. 验证订阅状态
	if sub.Status == SubscriptionStatusExpired {
		return false, ErrSubscriptionExpired
	}
	if sub.Status == SubscriptionStatusSuspended {
		return false, ErrSubscriptionSuspended
	}
	if sub.IsExpiredAt(now) {
		return false, ErrSubscriptionExpired
	}

	// 2. 内存中修正已跨越的 anchored 7-day 窗口。调用方随后同步推进
	//    DB 窗口，并用回读快照重新校验。
	if sub.NeedsWeeklyResetAt(now) {
		sub.WeeklyUsageUSD = 0
		needsMaintenance = true
	}
	if sub.NeedsMonthlyResetAt(now) {
		sub.MonthlyUsageUSD = 0
		needsMaintenance = true
	}
	// 3. Daily remains legacy-only. Weekly and monthly limits are both
	// authoritative; the monthly limit falls back to four weekly allowances.
	if !sub.CheckWeeklyLimitAt(group, 0, now) {
		return needsMaintenance, ErrWeeklyLimitExceeded
	}
	if !sub.CheckMonthlyLimitAt(group, 0, now) {
		return needsMaintenance, ErrMonthlyLimitExceeded
	}

	return needsMaintenance, nil
}

// DoWindowMaintenance 异步执行窗口维护（激活+重置）
// 使用独立 context，不受请求取消影响。
// 注意：此方法仅在 ValidateAndCheckLimits 返回 needsMaintenance=true 时调用，
// 而 IsExpired()=true 的订阅在 ValidateAndCheckLimits 中已被拦截返回错误，
// 因此进入此方法的订阅一定未过期，无需处理过期状态同步。
func (s *SubscriptionService) DoWindowMaintenance(sub *UserSubscription) {
	if s == nil {
		return
	}
	if s.maintenanceQueue != nil {
		err := s.maintenanceQueue.TryEnqueue(func() {
			s.doWindowMaintenance(sub)
		})
		if err != nil {
			log.Printf("Subscription maintenance enqueue failed: %v", err)
		}
		return
	}

	s.doWindowMaintenance(sub)
}

func (s *SubscriptionService) doWindowMaintenance(sub *UserSubscription) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Activate missing windows and advance elapsed ones.
	if err := s.CheckAndResetWindows(ctx, sub); err != nil {
		log.Printf("Failed to reset subscription windows: %v", err)
	}

	// 失效 L1 缓存，确保后续请求拿到更新后的数据
	s.InvalidateSubCache(sub.UserID, sub.GroupID)
}

// RecordUsage 记录使用量到订阅
func (s *SubscriptionService) RecordUsage(ctx context.Context, subscriptionID int64, costUSD float64) error {
	return s.userSubRepo.IncrementUsage(ctx, subscriptionID, costUSD)
}

// SubscriptionProgress 订阅进度
type SubscriptionProgress struct {
	ID            int64                `json:"id"`
	GroupName     string               `json:"group_name"`
	ExpiresAt     time.Time            `json:"expires_at"`
	ExpiresInDays int                  `json:"expires_in_days"`
	Daily         *UsageWindowProgress `json:"daily,omitempty"`
	Weekly        *UsageWindowProgress `json:"weekly,omitempty"`
	Monthly       *UsageWindowProgress `json:"monthly,omitempty"`
}

// UsageWindowProgress 使用窗口进度
type UsageWindowProgress struct {
	State           string    `json:"state"`
	LimitUSD        float64   `json:"limit_usd"`
	UsedUSD         *float64  `json:"used_usd"`
	RemainingUSD    *float64  `json:"remaining_usd"`
	Percentage      *float64  `json:"percentage"`
	WindowStart     time.Time `json:"window_start"`
	ResetsAt        time.Time `json:"resets_at"`
	ResetsInSeconds int64     `json:"resets_in_seconds"`
}

const (
	usageWindowProgressActive    = "active"
	usageWindowProgressExhausted = "exhausted"
	usageWindowProgressUnknown   = "unknown"
)

// GetSubscriptionProgress 获取订阅使用进度
func (s *SubscriptionService) GetSubscriptionProgress(ctx context.Context, subscriptionID int64) (*SubscriptionProgress, error) {
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	group := sub.Group
	if group == nil {
		group, err = s.groupRepo.GetByID(ctx, sub.GroupID)
		if err != nil {
			return nil, err
		}
	}

	return s.calculateProgress(sub, group), nil
}

// calculateProgress 根据已加载的订阅和分组数据计算使用进度（纯内存计算，无 DB 查询）
func (s *SubscriptionService) calculateProgress(sub *UserSubscription, group *Group) *SubscriptionProgress {
	progress := &SubscriptionProgress{
		ID:            sub.ID,
		GroupName:     group.Name,
		ExpiresAt:     sub.ExpiresAt,
		ExpiresInDays: sub.DaysRemaining(),
	}

	// Expose only counters whose persisted marker matches the exact anchored
	// period. A missing/stale marker is unknown, never an invented zero.
	now := time.Now()
	if group.HasWeeklyLimit() {
		windowStart, resetsAt, ok := sub.WeeklyWindowAt(now)
		if !ok {
			return progress
		}
		progress.Weekly = calculateUsageWindowProgress(
			*group.WeeklyLimitUSD,
			sub.WeeklyUsageUSD,
			sub.WeeklyWindowStart,
			windowStart,
			resetsAt,
			now,
		)
	}

	if monthlyLimit, ok := group.EffectiveMonthlyLimitUSD(); ok {
		windowStart, resetsAt, windowOK := sub.MonthlyWindowAt(now)
		if !windowOK {
			return progress
		}
		progress.Monthly = calculateUsageWindowProgress(
			monthlyLimit,
			sub.MonthlyUsageUSD,
			sub.MonthlyWindowStart,
			windowStart,
			resetsAt,
			now,
		)
	}

	return progress
}

func calculateUsageWindowProgress(
	limit, used float64,
	storedWindowStart *time.Time,
	windowStart, resetsAt, now time.Time,
) *UsageWindowProgress {
	resetsInSeconds := int64(resetsAt.Sub(now).Seconds())
	if resetsInSeconds < 0 {
		resetsInSeconds = 0
	}
	progress := &UsageWindowProgress{
		State:           usageWindowProgressUnknown,
		LimitUSD:        limit,
		WindowStart:     windowStart,
		ResetsAt:        resetsAt,
		ResetsInSeconds: resetsInSeconds,
	}
	if storedWindowStart == nil || !storedWindowStart.Equal(windowStart) {
		return progress
	}

	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}
	percentage := 100.0
	if limit > 0 {
		percentage = (used / limit) * 100
	}
	if percentage > 100 {
		percentage = 100
	}
	state := usageWindowProgressActive
	if limit <= 0 || used >= limit {
		state = usageWindowProgressExhausted
	}
	progress.State = state
	progress.UsedUSD = progressFloat64Pointer(used)
	progress.RemainingUSD = progressFloat64Pointer(remaining)
	progress.Percentage = progressFloat64Pointer(percentage)
	return progress
}

func progressFloat64Pointer(value float64) *float64 {
	return &value
}

// GetUserSubscriptionsWithProgress 获取用户所有订阅及进度
func (s *SubscriptionService) GetUserSubscriptionsWithProgress(ctx context.Context, userID int64) ([]SubscriptionProgress, error) {
	// ListActiveByUserID 已使用 .WithGroup() eager-load Group 关联，1 次查询获取所有数据
	subs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	progresses := make([]SubscriptionProgress, 0, len(subs))
	for i := range subs {
		sub := &subs[i]
		group := sub.Group
		if group == nil {
			continue
		}
		progresses = append(progresses, *s.calculateProgress(sub, group))
	}

	return progresses, nil
}

// ValidateSubscription 验证订阅是否有效
func (s *SubscriptionService) ValidateSubscription(ctx context.Context, sub *UserSubscription) error {
	if sub.Status == SubscriptionStatusExpired {
		return ErrSubscriptionExpired
	}
	if sub.Status == SubscriptionStatusSuspended {
		return ErrSubscriptionSuspended
	}
	if sub.IsExpired() {
		// 更新状态
		_ = s.userSubRepo.UpdateStatus(ctx, sub.ID, SubscriptionStatusExpired)
		return ErrSubscriptionExpired
	}
	return nil
}
