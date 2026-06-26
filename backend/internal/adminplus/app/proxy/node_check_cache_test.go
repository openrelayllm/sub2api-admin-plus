package proxy

import (
	"context"
	"testing"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisNodeCheckCacheRoundTrip(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })
	cache := NewRedisNodeCheckCache(rdb)
	ctx := context.Background()
	now := time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)
	latency := 123

	err := cache.SetNodeCheck(ctx, &adminplusdomain.ProxyNode{
		ID:            9,
		HealthStatus:  adminplusdomain.ProxyNodeHealthHealthy,
		LastLatencyMS: &latency,
		LastEgressIP:  "203.0.113.9",
		LastCheckedAt: &now,
	}, time.Hour)
	require.NoError(t, err)

	got, err := cache.GetNodeChecks(ctx, []int64{9, 404})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, adminplusdomain.ProxyNodeHealthHealthy, got[9].HealthStatus)
	require.Equal(t, "203.0.113.9", got[9].LastEgressIP)
	require.NotNil(t, got[9].LastLatencyMS)
	require.Equal(t, latency, *got[9].LastLatencyMS)
	require.NotNil(t, got[9].LastCheckedAt)
}
