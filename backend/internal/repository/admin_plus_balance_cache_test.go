package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/balances"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestAdminPlusBalanceCacheSetAndGetCurrent(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })
	cache := NewAdminPlusBalanceCache(rdb)
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	err := cache.SetCurrent(ctx, &balances.CurrentBalance{
		SupplierID:     7,
		RuntimeStatus:  adminplusdomain.SupplierRuntimeStatusCandidate,
		BalanceCents:   183,
		Currency:       "USD",
		SwitchEligible: true,
		Source:         "provider_session",
		CapturedAt:     now,
		RefreshAfter:   now.Add(5 * time.Minute),
		ExpiresAt:      now.Add(15 * time.Minute),
	}, 15*time.Minute)
	require.NoError(t, err)

	got, err := cache.GetCurrent(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, int64(183), got.BalanceCents)
	require.Equal(t, "USD", got.Currency)
	require.Equal(t, adminplusdomain.SupplierRuntimeStatusCandidate, got.RuntimeStatus)
	require.Equal(t, int64(7), got.SupplierID)
}

func TestAdminPlusBalanceCacheMiss(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })
	cache := NewAdminPlusBalanceCache(rdb)

	got, err := cache.GetCurrent(context.Background(), 404)

	require.Nil(t, got)
	require.ErrorIs(t, err, balances.ErrCurrentBalanceCacheMiss)
}
