package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/redis/go-redis/v9"
)

const nodeCheckCacheTTL = 24 * time.Hour

type NodeCheckCache interface {
	SetNodeCheck(ctx context.Context, node *adminplusdomain.ProxyNode, ttl time.Duration) error
	GetNodeChecks(ctx context.Context, nodeIDs []int64) (map[int64]*NodeCheckSnapshot, error)
}

type NodeCheckSnapshot struct {
	NodeID           int64                                 `json:"node_id"`
	HealthStatus     adminplusdomain.ProxyNodeHealthStatus `json:"health_status"`
	LastLatencyMS    *int                                  `json:"last_latency_ms,omitempty"`
	LastEgressIP     string                                `json:"last_egress_ip,omitempty"`
	LastErrorCode    string                                `json:"last_error_code,omitempty"`
	LastErrorMessage string                                `json:"last_error_message,omitempty"`
	LastCheckedAt    *time.Time                            `json:"last_checked_at,omitempty"`
}

type redisNodeCheckCache struct {
	rdb *redis.Client
}

func NewRedisNodeCheckCache(rdb *redis.Client) NodeCheckCache {
	if rdb == nil {
		return nil
	}
	return &redisNodeCheckCache{rdb: rdb}
}

func (c *redisNodeCheckCache) SetNodeCheck(ctx context.Context, node *adminplusdomain.ProxyNode, ttl time.Duration) error {
	if c == nil || c.rdb == nil || node == nil || node.ID <= 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = nodeCheckCacheTTL
	}
	payload, err := json.Marshal(NodeCheckSnapshotFromNode(node))
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, proxyNodeCheckCacheKey(node.ID), payload, ttl).Err()
}

func (c *redisNodeCheckCache) GetNodeChecks(ctx context.Context, nodeIDs []int64) (map[int64]*NodeCheckSnapshot, error) {
	results := make(map[int64]*NodeCheckSnapshot)
	if c == nil || c.rdb == nil || len(nodeIDs) == 0 {
		return results, nil
	}
	ids := compactNodeCheckIDs(nodeIDs)
	if len(ids) == 0 {
		return results, nil
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, proxyNodeCheckCacheKey(id))
	}
	values, err := c.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return results, err
	}
	for i, raw := range values {
		if raw == nil {
			continue
		}
		payload, ok := redisPayloadBytes(raw)
		if !ok {
			continue
		}
		var snapshot NodeCheckSnapshot
		if err := json.Unmarshal(payload, &snapshot); err != nil {
			continue
		}
		if snapshot.NodeID == 0 {
			snapshot.NodeID = ids[i]
		}
		results[ids[i]] = &snapshot
	}
	return results, nil
}

func proxyNodeCheckCacheKey(nodeID int64) string {
	return fmt.Sprintf("admin_plus:proxy:node_check:%d", nodeID)
}

func compactNodeCheckIDs(nodeIDs []int64) []int64 {
	seen := make(map[int64]struct{}, len(nodeIDs))
	ids := make([]int64, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func redisPayloadBytes(raw any) ([]byte, bool) {
	switch value := raw.(type) {
	case string:
		return []byte(value), true
	case []byte:
		return value, true
	default:
		return nil, false
	}
}

func NodeCheckSnapshotFromNode(node *adminplusdomain.ProxyNode) *NodeCheckSnapshot {
	if node == nil {
		return nil
	}
	var latency *int
	if node.LastLatencyMS != nil {
		value := *node.LastLatencyMS
		latency = &value
	}
	var checkedAt *time.Time
	if node.LastCheckedAt != nil {
		value := *node.LastCheckedAt
		checkedAt = &value
	}
	return &NodeCheckSnapshot{
		NodeID:           node.ID,
		HealthStatus:     node.HealthStatus,
		LastLatencyMS:    latency,
		LastEgressIP:     node.LastEgressIP,
		LastErrorCode:    node.LastErrorCode,
		LastErrorMessage: node.LastErrorMessage,
		LastCheckedAt:    checkedAt,
	}
}

func applyNodeCheckSnapshot(node *adminplusdomain.ProxyNode, snapshot *NodeCheckSnapshot) {
	if node == nil || snapshot == nil || snapshot.NodeID != node.ID {
		return
	}
	if node.HealthStatus == adminplusdomain.ProxyNodeHealthDisabled {
		return
	}
	if snapshot.LastCheckedAt != nil && node.LastCheckedAt != nil && snapshot.LastCheckedAt.Before(*node.LastCheckedAt) {
		return
	}
	if snapshot.HealthStatus != "" {
		node.HealthStatus = snapshot.HealthStatus
	}
	node.LastLatencyMS = snapshot.LastLatencyMS
	node.LastEgressIP = snapshot.LastEgressIP
	node.LastErrorCode = snapshot.LastErrorCode
	node.LastErrorMessage = snapshot.LastErrorMessage
	node.LastCheckedAt = snapshot.LastCheckedAt
}
