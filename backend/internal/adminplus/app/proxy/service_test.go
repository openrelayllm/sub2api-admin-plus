package proxy

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestServiceRequestAssignmentEnforcesTargetRateLimit(t *testing.T) {
	repo := newProxyServiceTestRepository()
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{})
	ctx := context.Background()

	first, err := svc.RequestAssignment(ctx, RequestAssignmentInput{
		TaskType:   "site_discovery",
		TaskID:     "run-1",
		PolicyID:   repo.policy.ID,
		TargetHost: "example.com",
		Purpose:    adminplusdomain.ProxyPurposeSiteDiscovery,
		Method:     "GET",
	})
	require.NoError(t, err)
	require.NotZero(t, first.ID)

	_, err = svc.RequestAssignment(ctx, RequestAssignmentInput{
		TaskType:   "site_discovery",
		TaskID:     "run-2",
		PolicyID:   repo.policy.ID,
		TargetHost: "example.com",
		Purpose:    adminplusdomain.ProxyPurposeSiteDiscovery,
		Method:     "GET",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "rate limit")
	require.True(t, repo.hasAudit("policy_rate_limited"))
}

func TestServiceSwitchAssignmentEnforcesSwitchBudget(t *testing.T) {
	repo := newProxyServiceTestRepository()
	repo.policy.MaxSwitchesPerTask = 1
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{})
	ctx := context.Background()

	assignment, err := svc.RequestAssignment(ctx, RequestAssignmentInput{
		TaskType:   "site_discovery",
		TaskID:     "run-1",
		PolicyID:   repo.policy.ID,
		TargetHost: "example.com",
		Purpose:    adminplusdomain.ProxyPurposeSiteDiscovery,
		Method:     "GET",
	})
	require.NoError(t, err)

	updated, err := svc.SwitchAssignment(ctx, assignment.ID, SwitchAssignmentInput{NodeID: repo.secondNode.ID})
	require.NoError(t, err)
	require.Equal(t, 1, updated.SwitchCount)

	_, err = svc.SwitchAssignment(ctx, assignment.ID, SwitchAssignmentInput{NodeID: repo.firstNode.ID})
	require.Error(t, err)
	require.Contains(t, err.Error(), "switch budget")
	require.True(t, repo.hasAudit("node_switch_denied"))
}

func TestServiceDisabledRejectsRuntimeOperations(t *testing.T) {
	repo := newProxyServiceTestRepository()
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{Enabled: false, MaxSlots: 2})
	ctx := context.Background()

	status, err := svc.CenterStatus(ctx)
	require.NoError(t, err)
	require.False(t, status.ProxyEnabled)

	_, err = svc.RequestAssignment(ctx, RequestAssignmentInput{
		TaskType:   "site_discovery",
		TaskID:     "run-1",
		PolicyID:   repo.policy.ID,
		TargetHost: "example.com",
		Purpose:    adminplusdomain.ProxyPurposeSiteDiscovery,
		Method:     "GET",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "disabled")

	_, err = svc.CheckNode(ctx, repo.firstNode.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "disabled")

	assignment := repo.seedActiveAssignment()
	_, err = svc.SwitchAssignment(ctx, assignment.ID, SwitchAssignmentInput{NodeID: repo.secondNode.ID})
	require.Error(t, err)
	require.Contains(t, err.Error(), "disabled")
	require.True(t, repo.hasAudit("proxy_disabled"))
}

func TestServiceReportFailureAutoSwitchesReplacementNode(t *testing.T) {
	repo := newProxyServiceTestRepository()
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{})
	ctx := context.Background()

	assignment, err := svc.RequestAssignment(ctx, RequestAssignmentInput{
		TaskType:   "site_discovery",
		TaskID:     "run-1",
		PolicyID:   repo.policy.ID,
		TargetHost: "example.com",
		Purpose:    adminplusdomain.ProxyPurposeSiteDiscovery,
		Method:     "GET",
	})
	require.NoError(t, err)
	require.Equal(t, repo.firstNode.ID, assignment.NodeID)

	updated, err := svc.ReportFailure(ctx, assignment.ID, ReportFailureInput{
		ErrorCode:    "SITE_DISCOVERY_PROXY_NETWORK_FAILED",
		ErrorMessage: "dial tcp timeout",
	})
	require.NoError(t, err)
	require.Equal(t, repo.secondNode.ID, updated.NodeID)
	require.Equal(t, 1, updated.SwitchCount)
	require.Equal(t, adminplusdomain.ProxyNodeHealthSuspect, repo.firstNode.HealthStatus)
	require.True(t, repo.hasAudit("node_failure_reported"))
	require.True(t, repo.hasAudit("node_auto_switched"))
}

func TestServiceReportFailureDoesNotAutoSwitchFixedPolicy(t *testing.T) {
	repo := newProxyServiceTestRepository()
	repo.policy.Config = map[string]any{
		"selection_mode": "fixed",
		"fixed_node_id":  repo.firstNode.ID,
	}
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{})
	ctx := context.Background()

	assignment, err := svc.RequestAssignment(ctx, RequestAssignmentInput{
		TaskType:   "site_discovery",
		TaskID:     "run-1",
		PolicyID:   repo.policy.ID,
		TargetHost: "example.com",
		Purpose:    adminplusdomain.ProxyPurposeSiteDiscovery,
		Method:     "GET",
	})
	require.NoError(t, err)

	_, err = svc.ReportFailure(ctx, assignment.ID, ReportFailureInput{
		ErrorCode:    "SITE_DISCOVERY_PROXY_NETWORK_FAILED",
		ErrorMessage: "dial tcp timeout",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "automatic switching")
	require.Equal(t, adminplusdomain.ProxyNodeHealthSuspect, repo.firstNode.HealthStatus)
	require.False(t, repo.hasAudit("node_auto_switched"))
	require.True(t, repo.hasAudit("node_switch_denied"))
}

func TestServiceRotateRuntimeSlotSecret(t *testing.T) {
	repo := newProxyServiceTestRepository()
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{})
	ctx := context.Background()
	oldSecret := repo.slotSecret

	slot, err := svc.RotateRuntimeSlotSecret(ctx, repo.slot.ID)
	require.NoError(t, err)
	require.Equal(t, repo.slot.ID, slot.ID)
	require.NotEmpty(t, repo.slotSecret)
	require.NotEqual(t, oldSecret, repo.slotSecret)
	require.True(t, repo.hasAudit("slot_secret_rotated"))
}

func TestServiceRotateRuntimeSlotSecretRejectsAssignedSlot(t *testing.T) {
	repo := newProxyServiceTestRepository()
	repo.slot.Status = adminplusdomain.ProxyRuntimeSlotAssigned
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{})

	_, err := svc.RotateRuntimeSlotSecret(context.Background(), repo.slot.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "assigned")
	require.Equal(t, "secret", repo.slotSecret)
}

func TestServiceCheckNodeCachesAndListsCachedResult(t *testing.T) {
	repo := newProxyServiceTestRepository()
	cache := newMemoryNodeCheckCache()
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{Enabled: true, BinaryPath: "/usr/local/bin/mihomo"}).WithNodeCheckCache(cache)
	ctx := context.Background()

	checked, err := svc.CheckNode(ctx, repo.firstNode.ID)
	require.NoError(t, err)
	require.Equal(t, "203.0.113.10", checked.LastEgressIP)
	require.NotNil(t, checked.LastLatencyMS)
	require.NotNil(t, checked.LastCheckedAt)

	repo.firstNode.LastLatencyMS = nil
	repo.firstNode.LastEgressIP = ""
	repo.firstNode.LastCheckedAt = nil

	nodes, err := svc.ListNodes(ctx, NodeFilter{})
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	require.Equal(t, "203.0.113.10", nodes[0].LastEgressIP)
	require.NotNil(t, nodes[0].LastLatencyMS)
	require.NotNil(t, nodes[0].LastCheckedAt)
}

func TestServiceCheckNodeFailsFastWhenServerIsLiteralReservedIP(t *testing.T) {
	repo := newProxyServiceTestRepository()
	repo.firstNode.RawMetadata = map[string]any{
		"server": "127.127.127.5",
		"port":   19273,
	}
	cache := newMemoryNodeCheckCache()
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{Enabled: true, BinaryPath: "/usr/local/bin/mihomo"}).WithNodeCheckCache(cache)

	checked, err := svc.CheckNode(context.Background(), repo.firstNode.ID)

	require.Error(t, err)
	require.Equal(t, "PROXY_NODE_SERVER_RESOLVES_RESERVED_IP", infraerrors.Reason(err))
	require.Equal(t, adminplusdomain.ProxyNodeHealthUnhealthy, checked.HealthStatus)
	require.Equal(t, "PROXY_NODE_SERVER_RESOLVES_RESERVED_IP", checked.LastErrorCode)
	require.Contains(t, checked.LastErrorMessage, "127.127.127.5:19273")
	require.NotNil(t, checked.LastCheckedAt)
	require.Contains(t, cache.items[repo.firstNode.ID].LastErrorMessage, "127.127.127.5:19273")
}

func TestServiceCheckNodeDoesNotResolveHostnameDuringPreflight(t *testing.T) {
	repo := newProxyServiceTestRepository()
	repo.firstNode.RawMetadata = map[string]any{
		"server": "node.example.test",
		"port":   19273,
	}
	cache := newMemoryNodeCheckCache()
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{Enabled: true, BinaryPath: "/usr/local/bin/mihomo"}).WithNodeCheckCache(cache)

	checked, err := svc.CheckNode(context.Background(), repo.firstNode.ID)

	require.NoError(t, err)
	require.Equal(t, adminplusdomain.ProxyNodeHealthHealthy, checked.HealthStatus)
	require.Equal(t, "203.0.113.10", checked.LastEgressIP)
	require.Empty(t, checked.LastErrorCode)
}

func TestServiceCheckNodesChecksAllNodes(t *testing.T) {
	repo := newProxyServiceTestRepository()
	cache := newMemoryNodeCheckCache()
	svc := NewService(repo, testSecretCipher{}, nil, testRuntime{}, RuntimeConfig{Enabled: true, BinaryPath: "/usr/local/bin/mihomo"}).WithNodeCheckCache(cache)

	result, err := svc.CheckNodes(context.Background(), NodeFilter{})

	require.NoError(t, err)
	require.Equal(t, 2, result.Total)
	require.Equal(t, 2, result.Succeeded)
	require.Equal(t, 0, result.Failed)
	require.Len(t, result.Results, 2)
	require.Len(t, cache.items, 2)
}

type testSecretCipher struct{}

func (testSecretCipher) Encrypt(plaintext string) (string, error)  { return plaintext, nil }
func (testSecretCipher) Decrypt(ciphertext string) (string, error) { return ciphertext, nil }

type testRuntime struct{}

func (testRuntime) ConfigureSlot(ctx context.Context, slot *adminplusdomain.ProxyRuntimeSlot, node *adminplusdomain.ProxyNode, mihomoYAML []byte, controllerSecret string) (*RuntimeSlotResult, error) {
	now := time.Now()
	return &RuntimeSlotResult{ConfigPath: "/tmp/proxy-test.yaml", StartedAt: &now}, nil
}

func (testRuntime) SwitchNode(ctx context.Context, slot *adminplusdomain.ProxyRuntimeSlot, nodeName string, controllerSecret string) error {
	return nil
}

func (testRuntime) VerifyEgress(ctx context.Context, mixedPort int) (string, error) {
	return "203.0.113.10", nil
}

func (testRuntime) RestartSlot(ctx context.Context, slot *adminplusdomain.ProxyRuntimeSlot) error {
	return nil
}

type memoryNodeCheckCache struct {
	items map[int64]*NodeCheckSnapshot
}

func newMemoryNodeCheckCache() *memoryNodeCheckCache {
	return &memoryNodeCheckCache{items: make(map[int64]*NodeCheckSnapshot)}
}

func (c *memoryNodeCheckCache) SetNodeCheck(ctx context.Context, node *adminplusdomain.ProxyNode, ttl time.Duration) error {
	c.items[node.ID] = NodeCheckSnapshotFromNode(node)
	return nil
}

func (c *memoryNodeCheckCache) GetNodeChecks(ctx context.Context, nodeIDs []int64) (map[int64]*NodeCheckSnapshot, error) {
	out := make(map[int64]*NodeCheckSnapshot)
	for _, id := range nodeIDs {
		if item := c.items[id]; item != nil {
			clone := *item
			out[id] = &clone
		}
	}
	return out, nil
}

type proxyServiceTestRepository struct {
	mu          sync.Mutex
	nextID      int64
	policy      *adminplusdomain.ProxyPolicy
	target      *adminplusdomain.ProxyTargetPolicy
	firstNode   *adminplusdomain.ProxyNode
	secondNode  *adminplusdomain.ProxyNode
	slot        *adminplusdomain.ProxyRuntimeSlot
	slotSecret  string
	assignments map[int64]*adminplusdomain.ProxyAssignment
	audits      []*adminplusdomain.ProxyAuditEvent
}

func newProxyServiceTestRepository() *proxyServiceTestRepository {
	return &proxyServiceTestRepository{
		nextID: 100,
		policy: &adminplusdomain.ProxyPolicy{
			ID:                    1,
			Name:                  "test policy",
			Enabled:               true,
			SubscriptionIDs:       []int64{10},
			PreferredRegions:      []string{"HK"},
			MaxConcurrency:        2,
			MaxSwitchesPerTask:    2,
			ConnectTimeoutMS:      1000,
			RequestTimeoutMS:      3000,
			Config:                map[string]any{},
			HealthyNodesAvailable: 2,
		},
		target: &adminplusdomain.ProxyTargetPolicy{
			ID:                 2,
			PolicyID:           1,
			TargetHost:         "example.com",
			Purpose:            adminplusdomain.ProxyPurposeSiteDiscovery,
			AllowedMethods:     []string{"GET"},
			RateLimitPerMinute: 1,
			Enabled:            true,
		},
		firstNode: &adminplusdomain.ProxyNode{
			ID:             3,
			SubscriptionID: 10,
			ConfigVersion:  "v1",
			DisplayName:    "HK-1",
			Protocol:       "ss",
			Region:         "HK",
			HealthStatus:   adminplusdomain.ProxyNodeHealthHealthy,
		},
		secondNode: &adminplusdomain.ProxyNode{
			ID:             4,
			SubscriptionID: 10,
			ConfigVersion:  "v1",
			DisplayName:    "HK-2",
			Protocol:       "ss",
			Region:         "HK",
			HealthStatus:   adminplusdomain.ProxyNodeHealthHealthy,
		},
		slot: &adminplusdomain.ProxyRuntimeSlot{
			ID:             5,
			SlotKey:        "slot-1",
			Status:         adminplusdomain.ProxyRuntimeSlotIdle,
			MixedPort:      17890,
			ControllerPort: 19090,
		},
		slotSecret:  "secret",
		assignments: make(map[int64]*adminplusdomain.ProxyAssignment),
	}
}

func (r *proxyServiceTestRepository) CenterStatus(ctx context.Context) (*adminplusdomain.ProxyCenterStatus, error) {
	return &adminplusdomain.ProxyCenterStatus{}, nil
}

func (r *proxyServiceTestRepository) CreateSubscription(ctx context.Context, subscription *adminplusdomain.ProxySubscription, urlCiphertext string) (*adminplusdomain.ProxySubscription, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) UpdateSubscription(ctx context.Context, id int64, input UpdateSubscriptionInput) (*adminplusdomain.ProxySubscription, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) DeleteSubscription(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) GetSubscription(ctx context.Context, id int64) (*adminplusdomain.ProxySubscription, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) GetSubscriptionSecret(ctx context.Context, id int64) (*adminplusdomain.ProxySubscription, string, error) {
	return nil, "", fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) ListSubscriptions(ctx context.Context, filter SubscriptionFilter) ([]*adminplusdomain.ProxySubscription, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) SaveConfigVersion(ctx context.Context, subscriptionID int64, configVersion string, mihomoYAML []byte, generatedAt time.Time) error {
	return nil
}

func (r *proxyServiceTestRepository) GetConfigVersion(ctx context.Context, subscriptionID int64, configVersion string) ([]byte, error) {
	return []byte("proxies: []"), nil
}

func (r *proxyServiceTestRepository) ReplaceNodes(ctx context.Context, subscriptionID int64, configVersion string, nodes []*adminplusdomain.ProxyNode) error {
	return nil
}

func (r *proxyServiceTestRepository) UpdateSubscriptionRefresh(ctx context.Context, id int64, status adminplusdomain.ProxyRefreshStatus, refreshError string, activeConfigVersion string, nodeCount int, refreshedAt *time.Time) (*adminplusdomain.ProxySubscription, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) ListNodes(ctx context.Context, filter NodeFilter) ([]*adminplusdomain.ProxyNode, error) {
	return []*adminplusdomain.ProxyNode{cloneProxyNode(r.firstNode), cloneProxyNode(r.secondNode)}, nil
}

func (r *proxyServiceTestRepository) GetNode(ctx context.Context, id int64) (*adminplusdomain.ProxyNode, error) {
	if id == r.firstNode.ID {
		return cloneProxyNode(r.firstNode), nil
	}
	if id == r.secondNode.ID {
		return cloneProxyNode(r.secondNode), nil
	}
	return nil, notFound("PROXY_NODE_NOT_FOUND", "proxy node not found")
}

func (r *proxyServiceTestRepository) UpdateNodeHealth(ctx context.Context, id int64, status adminplusdomain.ProxyNodeHealthStatus, latencyMS *int, egressIP string, errorCode string, errorMessage string, checkedAt *time.Time) (*adminplusdomain.ProxyNode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var node *adminplusdomain.ProxyNode
	if id == r.firstNode.ID {
		node = cloneProxyNode(r.firstNode)
	} else if id == r.secondNode.ID {
		node = cloneProxyNode(r.secondNode)
	} else {
		return nil, notFound("PROXY_NODE_NOT_FOUND", "proxy node not found")
	}
	node.HealthStatus = status
	node.LastLatencyMS = latencyMS
	node.LastEgressIP = egressIP
	node.LastErrorCode = errorCode
	node.LastErrorMessage = errorMessage
	node.LastCheckedAt = checkedAt
	if id == r.firstNode.ID {
		r.firstNode = node
	} else {
		r.secondNode = node
	}
	return node, nil
}

func (r *proxyServiceTestRepository) UpdateNodeDisabled(ctx context.Context, id int64, disabled bool, reason string) (*adminplusdomain.ProxyNode, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) RecordHealthCheck(ctx context.Context, check *adminplusdomain.ProxyHealthCheck) (*adminplusdomain.ProxyHealthCheck, error) {
	return check, nil
}

func (r *proxyServiceTestRepository) ListPolicies(ctx context.Context, filter PolicyFilter) ([]*adminplusdomain.ProxyPolicy, error) {
	return []*adminplusdomain.ProxyPolicy{cloneProxyPolicy(r.policy)}, nil
}

func (r *proxyServiceTestRepository) GetPolicy(ctx context.Context, id int64) (*adminplusdomain.ProxyPolicy, error) {
	if id != r.policy.ID {
		return nil, notFound("PROXY_POLICY_NOT_FOUND", "proxy policy not found")
	}
	return cloneProxyPolicy(r.policy), nil
}

func (r *proxyServiceTestRepository) CreatePolicy(ctx context.Context, policy *adminplusdomain.ProxyPolicy) (*adminplusdomain.ProxyPolicy, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) UpdatePolicy(ctx context.Context, id int64, input UpdatePolicyInput) (*adminplusdomain.ProxyPolicy, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) DeletePolicy(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) ListTargets(ctx context.Context, filter TargetFilter) ([]*adminplusdomain.ProxyTargetPolicy, error) {
	target := cloneProxyTarget(r.target)
	if filter.PolicyID > 0 && filter.PolicyID != target.PolicyID {
		return nil, nil
	}
	if filter.Purpose != "" && filter.Purpose != target.Purpose {
		return nil, nil
	}
	if filter.Enabled != nil && *filter.Enabled != target.Enabled {
		return nil, nil
	}
	return []*adminplusdomain.ProxyTargetPolicy{target}, nil
}

func (r *proxyServiceTestRepository) CreateTarget(ctx context.Context, target *adminplusdomain.ProxyTargetPolicy) (*adminplusdomain.ProxyTargetPolicy, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) UpdateTarget(ctx context.Context, policyID int64, targetID int64, input UpdateTargetInput) (*adminplusdomain.ProxyTargetPolicy, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) DeleteTarget(ctx context.Context, policyID int64, targetID int64) error {
	return fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) ListRuntimeSlots(ctx context.Context, filter RuntimeSlotFilter) ([]*adminplusdomain.ProxyRuntimeSlot, error) {
	return []*adminplusdomain.ProxyRuntimeSlot{cloneProxySlot(r.slot)}, nil
}

func (r *proxyServiceTestRepository) CreateRuntimeSlot(ctx context.Context, slot *adminplusdomain.ProxyRuntimeSlot, controllerSecretCiphertext string) (*adminplusdomain.ProxyRuntimeSlot, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) GetRuntimeSlotSecret(ctx context.Context, id int64) (*adminplusdomain.ProxyRuntimeSlot, string, error) {
	if id != r.slot.ID {
		return nil, "", notFound("PROXY_RUNTIME_SLOT_NOT_FOUND", "proxy runtime slot not found")
	}
	return cloneProxySlot(r.slot), r.slotSecret, nil
}

func (r *proxyServiceTestRepository) UpdateRuntimeSlot(ctx context.Context, id int64, input UpdateRuntimeSlotInput) (*adminplusdomain.ProxyRuntimeSlot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id != r.slot.ID {
		return nil, notFound("PROXY_RUNTIME_SLOT_NOT_FOUND", "proxy runtime slot not found")
	}
	slot := cloneProxySlot(r.slot)
	if input.Status != nil {
		slot.Status = *input.Status
	}
	if input.ControllerSecretCiphertext != nil {
		r.slotSecret = *input.ControllerSecretCiphertext
	}
	if input.ProcessID != nil {
		slot.ProcessID = *input.ProcessID
	}
	if input.ConfigPath != nil {
		slot.ConfigPath = *input.ConfigPath
	}
	if input.AssignedTaskType != nil {
		slot.AssignedTaskType = *input.AssignedTaskType
	}
	if input.AssignedTaskID != nil {
		slot.AssignedTaskID = *input.AssignedTaskID
	}
	if input.SelectedNodeID != nil {
		slot.SelectedNodeID = *input.SelectedNodeID
	}
	if input.LastStartedAt != nil {
		slot.LastStartedAt = *input.LastStartedAt
	}
	if input.LastHeartbeatAt != nil {
		slot.LastHeartbeatAt = *input.LastHeartbeatAt
	}
	r.slot = slot
	return cloneProxySlot(r.slot), nil
}

func (r *proxyServiceTestRepository) CreateAssignment(ctx context.Context, assignment *adminplusdomain.ProxyAssignment) (*adminplusdomain.ProxyAssignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	clone := *assignment
	clone.ID = r.nextID
	clone.CreatedAt = time.Now()
	r.assignments[clone.ID] = &clone
	return cloneProxyAssignment(&clone), nil
}

func (r *proxyServiceTestRepository) GetAssignment(ctx context.Context, id int64) (*adminplusdomain.ProxyAssignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	assignment := r.assignments[id]
	if assignment == nil {
		return nil, notFound("PROXY_ASSIGNMENT_NOT_FOUND", "proxy assignment not found")
	}
	out := cloneProxyAssignment(assignment)
	out.MixedPort = r.slot.MixedPort
	return out, nil
}

func (r *proxyServiceTestRepository) ListAssignments(ctx context.Context, filter AssignmentFilter) ([]*adminplusdomain.ProxyAssignment, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *proxyServiceTestRepository) UpdateAssignment(ctx context.Context, id int64, input UpdateAssignmentInput) (*adminplusdomain.ProxyAssignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	assignment := r.assignments[id]
	if assignment == nil {
		return nil, notFound("PROXY_ASSIGNMENT_NOT_FOUND", "proxy assignment not found")
	}
	if input.NodeID != nil {
		assignment.NodeID = *input.NodeID
	}
	if input.EgressIP != nil {
		assignment.EgressIP = *input.EgressIP
	}
	if input.Status != nil {
		assignment.Status = *input.Status
	}
	if input.SwitchCount != nil {
		assignment.SwitchCount = *input.SwitchCount
	}
	if input.ErrorCode != nil {
		assignment.ErrorCode = *input.ErrorCode
	}
	if input.ErrorMessage != nil {
		assignment.ErrorMessage = *input.ErrorMessage
	}
	if input.ReleasedAt != nil {
		assignment.ReleasedAt = *input.ReleasedAt
	}
	out := cloneProxyAssignment(assignment)
	out.MixedPort = r.slot.MixedPort
	return out, nil
}

func (r *proxyServiceTestRepository) CreateAuditEvent(ctx context.Context, event *adminplusdomain.ProxyAuditEvent) (*adminplusdomain.ProxyAuditEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	clone := *event
	r.audits = append(r.audits, &clone)
	return &clone, nil
}

func (r *proxyServiceTestRepository) ListAuditEvents(ctx context.Context, filter AuditFilter) ([]*adminplusdomain.ProxyAuditEvent, error) {
	return r.audits, nil
}

func (r *proxyServiceTestRepository) CountPoliciesUsingSubscription(ctx context.Context, subscriptionID int64) (int, error) {
	return 0, nil
}

func (r *proxyServiceTestRepository) CountActiveAssignmentsForSubscription(ctx context.Context, subscriptionID int64) (int, error) {
	return 0, nil
}

func (r *proxyServiceTestRepository) CountActiveAssignmentsForPolicy(ctx context.Context, policyID int64) (int, error) {
	return 0, nil
}

func (r *proxyServiceTestRepository) hasAudit(eventType string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, event := range r.audits {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}

func (r *proxyServiceTestRepository) seedActiveAssignment() *adminplusdomain.ProxyAssignment {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	assignment := &adminplusdomain.ProxyAssignment{
		ID:         r.nextID,
		TaskType:   "site_discovery",
		TaskID:     "run-seeded",
		PolicyID:   r.policy.ID,
		SlotID:     r.slot.ID,
		NodeID:     r.firstNode.ID,
		TargetHost: r.target.TargetHost,
		EgressIP:   "203.0.113.10",
		Status:     adminplusdomain.ProxyAssignmentActive,
		StartedAt:  time.Now(),
		CreatedAt:  time.Now(),
	}
	r.assignments[assignment.ID] = assignment
	return cloneProxyAssignment(assignment)
}

func cloneProxyPolicy(in *adminplusdomain.ProxyPolicy) *adminplusdomain.ProxyPolicy {
	if in == nil {
		return nil
	}
	out := *in
	out.SubscriptionIDs = append([]int64(nil), in.SubscriptionIDs...)
	out.PreferredRegions = append([]string(nil), in.PreferredRegions...)
	out.Config = map[string]any{}
	for key, value := range in.Config {
		out.Config[key] = value
	}
	return &out
}

func cloneProxyTarget(in *adminplusdomain.ProxyTargetPolicy) *adminplusdomain.ProxyTargetPolicy {
	if in == nil {
		return nil
	}
	out := *in
	out.AllowedMethods = append([]string(nil), in.AllowedMethods...)
	return &out
}

func cloneProxyNode(in *adminplusdomain.ProxyNode) *adminplusdomain.ProxyNode {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneProxySlot(in *adminplusdomain.ProxyRuntimeSlot) *adminplusdomain.ProxyRuntimeSlot {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneProxyAssignment(in *adminplusdomain.ProxyAssignment) *adminplusdomain.ProxyAssignment {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
