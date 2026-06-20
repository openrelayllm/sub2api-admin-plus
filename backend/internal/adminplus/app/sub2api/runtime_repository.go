package sub2api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/redis/go-redis/v9"
)

const (
	accountConcurrencyKeyPrefix = "concurrency:account:"
	accountWaitKeyPrefix        = "wait:account:"
	tempUnschedAccountPrefix    = "temp_unsched:account:"
	defaultRuntimeLimit         = 100
	maxRuntimeLimit             = 500
)

type RuntimeFilter struct {
	AccountID int64
	Query     string
	Limit     int
}

type RuntimeRepository struct {
	db    *sql.DB
	redis Sub2APIRedis
	now   func() time.Time
}

type runtimeAccountRow struct {
	accountID               int64
	accountName             string
	accountPlatform         string
	accountType             string
	status                  string
	schedulable             bool
	configuredLimit         int
	errorMessage            string
	rateLimitResetAt        *time.Time
	overloadUntil           *time.Time
	tempUnschedulableUntil  *time.Time
	tempUnschedulableReason string
	lastUsedAt              *time.Time
}

type tempUnschedState struct {
	UntilUnix       int64  `json:"until_unix"`
	TriggeredAtUnix int64  `json:"triggered_at_unix"`
	StatusCode      int    `json:"status_code"`
	MatchedKeyword  string `json:"matched_keyword"`
	RuleIndex       int    `json:"rule_index"`
	ErrorMessage    string `json:"error_message"`
}

func NewRuntimeRepository(db ReadDB, redis Sub2APIRedis) *RuntimeRepository {
	return &RuntimeRepository{
		db:    db.DB,
		redis: redis,
		now:   time.Now,
	}
}

func (r *RuntimeRepository) ListAccountRuntime(ctx context.Context, filter RuntimeFilter) ([]*adminplusdomain.LocalAccountRuntime, error) {
	if r == nil || r.db == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "SUB2API_READ_DB_NOT_CONFIGURED", "sub2api read database is not configured")
	}
	normalized := normalizeRuntimeFilter(filter)
	accounts, err := r.listRuntimeAccounts(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return []*adminplusdomain.LocalAccountRuntime{}, nil
	}

	collectedAt := r.now().UTC()
	redisValues, redisErr := r.readRedisRuntime(ctx, accounts, collectedAt)
	if redisErr != nil {
		return nil, redisErr
	}

	items := make([]*adminplusdomain.LocalAccountRuntime, 0, len(accounts))
	for _, account := range accounts {
		item := account.toRuntime(collectedAt, r.redis.Configured)
		if redisValue, ok := redisValues[account.accountID]; ok {
			item.CurrentConcurrency = redisValue.currentConcurrency
			item.WaitingCount = redisValue.waitingCount
			if redisValue.tempUnschedUntil != nil {
				item.TempUnschedUntil = redisValue.tempUnschedUntil
			}
			if redisValue.tempUnschedReason != "" {
				item.TempUnschedReason = redisValue.tempUnschedReason
			}
		}
		if item.ConfiguredLimit > 0 {
			item.LoadPercent = float64(item.CurrentConcurrency+item.WaitingCount) / float64(item.ConfiguredLimit) * 100
		}
		item.SwitchEligible, item.BlockedReason = resolveSwitchEligibility(item, collectedAt)
		items = append(items, item)
	}
	return items, nil
}

func (r *RuntimeRepository) listRuntimeAccounts(ctx context.Context, filter RuntimeFilter) ([]runtimeAccountRow, error) {
	where := []string{"deleted_at IS NULL"}
	args := make([]any, 0, 3)
	if filter.AccountID > 0 {
		where = append(where, "id = "+addSQLArg(&args, filter.AccountID))
	}
	if filter.Query != "" {
		queryRef := addSQLArg(&args, "%"+strings.ToLower(filter.Query)+"%")
		where = append(where, "(LOWER(name) LIKE "+queryRef+" OR LOWER(platform) LIKE "+queryRef+" OR CAST(id AS TEXT) = "+addSQLArg(&args, filter.Query)+")")
	}
	limitRef := addSQLArg(&args, filter.Limit)

	query := `
		SELECT
			id,
			COALESCE(name, ''),
			COALESCE(platform, ''),
			COALESCE(type, ''),
			COALESCE(status, ''),
			COALESCE(schedulable, false),
			COALESCE(concurrency, 0),
			COALESCE(error_message, ''),
			rate_limit_reset_at,
			overload_until,
			temp_unschedulable_until,
			COALESCE(temp_unschedulable_reason, ''),
			last_used_at
		FROM accounts
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY id DESC
		LIMIT ` + limitRef

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	accounts := make([]runtimeAccountRow, 0, filter.Limit)
	for rows.Next() {
		var account runtimeAccountRow
		if err := rows.Scan(
			&account.accountID,
			&account.accountName,
			&account.accountPlatform,
			&account.accountType,
			&account.status,
			&account.schedulable,
			&account.configuredLimit,
			&account.errorMessage,
			&account.rateLimitResetAt,
			&account.overloadUntil,
			&account.tempUnschedulableUntil,
			&account.tempUnschedulableReason,
			&account.lastUsedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}

type redisRuntimeValue struct {
	currentConcurrency int
	waitingCount       int
	tempUnschedUntil   *time.Time
	tempUnschedReason  string
}

func (r *RuntimeRepository) readRedisRuntime(ctx context.Context, accounts []runtimeAccountRow, collectedAt time.Time) (map[int64]redisRuntimeValue, error) {
	values := make(map[int64]redisRuntimeValue, len(accounts))
	if r == nil || r.redis.Client == nil {
		return values, nil
	}

	pipe := r.redis.Client.Pipeline()
	type accountCmds struct {
		accountID int64
		zcard     *redis.IntCmd
		wait      *redis.StringCmd
		temp      *redis.StringCmd
	}
	cmds := make([]accountCmds, 0, len(accounts))
	for _, account := range accounts {
		id := strconv.FormatInt(account.accountID, 10)
		cmds = append(cmds, accountCmds{
			accountID: account.accountID,
			zcard:     pipe.ZCard(ctx, accountConcurrencyKeyPrefix+id),
			wait:      pipe.Get(ctx, accountWaitKeyPrefix+id),
			temp:      pipe.Get(ctx, tempUnschedAccountPrefix+id),
		})
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("read sub2api redis runtime: %w", err)
	}

	for _, cmd := range cmds {
		value := redisRuntimeValue{currentConcurrency: int(cmd.zcard.Val())}
		if waiting, err := cmd.wait.Int(); err == nil && waiting > 0 {
			value.waitingCount = waiting
		}
		if raw, err := cmd.temp.Result(); err == nil && strings.TrimSpace(raw) != "" {
			until, reason := parseTempUnsched(raw, collectedAt)
			value.tempUnschedUntil = until
			value.tempUnschedReason = reason
		}
		values[cmd.accountID] = value
	}
	return values, nil
}

func normalizeRuntimeFilter(filter RuntimeFilter) RuntimeFilter {
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultRuntimeLimit
	}
	if limit > maxRuntimeLimit {
		limit = maxRuntimeLimit
	}
	return RuntimeFilter{
		AccountID: filter.AccountID,
		Query:     strings.TrimSpace(filter.Query),
		Limit:     limit,
	}
}

func (a runtimeAccountRow) toRuntime(collectedAt time.Time, redisConfigured bool) *adminplusdomain.LocalAccountRuntime {
	return &adminplusdomain.LocalAccountRuntime{
		AccountID:           a.accountID,
		AccountName:         a.accountName,
		AccountPlatform:     a.accountPlatform,
		AccountType:         a.accountType,
		Status:              a.status,
		Schedulable:         a.schedulable,
		ConfiguredLimit:     a.configuredLimit,
		ErrorMessage:        a.errorMessage,
		RateLimitResetAt:    a.rateLimitResetAt,
		OverloadUntil:       a.overloadUntil,
		TempUnschedUntil:    a.tempUnschedulableUntil,
		TempUnschedReason:   a.tempUnschedulableReason,
		LastUsedAt:          a.lastUsedAt,
		CollectedAt:         collectedAt,
		RedisReadConfigured: redisConfigured,
	}
}

func parseTempUnsched(raw string, now time.Time) (*time.Time, string) {
	var state tempUnschedState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, ""
	}
	var until *time.Time
	if state.UntilUnix > 0 {
		value := time.Unix(state.UntilUnix, 0).UTC()
		if value.After(now) {
			until = &value
		}
	}
	reason := state.ErrorMessage
	if reason == "" {
		reason = state.MatchedKeyword
	}
	if reason == "" && state.StatusCode > 0 {
		reason = fmt.Sprintf("status_code=%d", state.StatusCode)
	}
	return until, reason
}

func resolveSwitchEligibility(item *adminplusdomain.LocalAccountRuntime, now time.Time) (bool, string) {
	if item == nil {
		return false, "unknown"
	}
	if item.Status != "active" {
		return false, "status_" + item.Status
	}
	if !item.Schedulable {
		return false, "not_schedulable"
	}
	if item.RateLimitResetAt != nil && item.RateLimitResetAt.After(now) {
		return false, "rate_limited"
	}
	if item.OverloadUntil != nil && item.OverloadUntil.After(now) {
		return false, "overloaded"
	}
	if item.TempUnschedUntil != nil && item.TempUnschedUntil.After(now) {
		return false, "temp_unschedulable"
	}
	if item.ConfiguredLimit > 0 && item.CurrentConcurrency >= item.ConfiguredLimit {
		return false, "concurrency_full"
	}
	return true, ""
}
