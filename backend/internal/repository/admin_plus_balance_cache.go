package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	balancesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/balances"
	"github.com/redis/go-redis/v9"
)

const adminPlusBalanceCurrentKeyFormat = "admin_plus:supplier:%d:balance:current"

type adminPlusBalanceCache struct {
	rdb *redis.Client
}

func NewAdminPlusBalanceCache(rdb *redis.Client) balancesapp.BalanceCache {
	return &adminPlusBalanceCache{rdb: rdb}
}

func (c *adminPlusBalanceCache) GetCurrent(ctx context.Context, supplierID int64) (*balancesapp.CurrentBalance, error) {
	if c == nil || c.rdb == nil {
		return nil, balancesapp.ErrCurrentBalanceCacheMiss
	}
	raw, err := c.rdb.Get(ctx, adminPlusBalanceCurrentKey(supplierID)).Result()
	if err == redis.Nil {
		return nil, balancesapp.ErrCurrentBalanceCacheMiss
	}
	if err != nil {
		return nil, err
	}
	var current balancesapp.CurrentBalance
	if err := json.Unmarshal([]byte(raw), &current); err != nil {
		return nil, err
	}
	return &current, nil
}

func (c *adminPlusBalanceCache) SetCurrent(ctx context.Context, current *balancesapp.CurrentBalance, ttl time.Duration) error {
	if c == nil || c.rdb == nil || current == nil {
		return nil
	}
	if ttl <= 0 {
		return nil
	}
	payload, err := json.Marshal(current)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, adminPlusBalanceCurrentKey(current.SupplierID), payload, ttl).Err()
}

func adminPlusBalanceCurrentKey(supplierID int64) string {
	return fmt.Sprintf(adminPlusBalanceCurrentKeyFormat, supplierID)
}
