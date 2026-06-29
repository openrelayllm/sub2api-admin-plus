package purity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProbeBillingMultiplierFromUsageDelta(t *testing.T) {
	before := &usageCostSnapshot{StandardCost: 10, ActualCost: 1}
	after := &usageCostSnapshot{StandardCost: 11, ActualCost: 1.07}

	probe := probeBillingMultiplierFromUsageDelta(before, after)

	require.NotNil(t, probe.Multiplier)
	require.InDelta(t, 0.07, *probe.Multiplier, 0.0001)
	require.Equal(t, "usage_delta", probe.Source)
}

func TestParseUsageCostSnapshotUsesSub2APIUsageShape(t *testing.T) {
	body := []byte(`{
		"mode": "unrestricted",
		"usage": {
			"total": {
				"cost": 12.5,
				"actual_cost": 0.875
			}
		}
	}`)

	snapshot, ok := parseUsageCostSnapshot(body)

	require.True(t, ok)
	require.Equal(t, 12.5, snapshot.StandardCost)
	require.Equal(t, 0.875, snapshot.ActualCost)
}

func TestParseUsageCostSnapshotUsesSub2APITotalShape(t *testing.T) {
	body := []byte(`{
		"mode": "unrestricted",
		"usage": {
			"today": {
				"cost": 0.02,
				"actual_cost": 0.0014
			},
			"total": {
				"cost": 12.5,
				"actual_cost": 0.875
			}
		}
	}`)

	snapshot, ok := parseUsageCostSnapshot(body)

	require.True(t, ok)
	require.Equal(t, 12.5, snapshot.StandardCost)
	require.Equal(t, 0.875, snapshot.ActualCost)
}

func TestParseUsageCostSnapshotUsesSub2APIAccountSummaryShape(t *testing.T) {
	body := []byte(`{
		"summary": {
			"total_standard_cost": 12.5,
			"total_cost": 1.375,
			"total_user_cost": 1.5
		}
	}`)

	snapshot, ok := parseUsageCostSnapshot(body)

	require.True(t, ok)
	require.Equal(t, 12.5, snapshot.StandardCost)
	require.Equal(t, 1.375, snapshot.ActualCost)
}

func TestBillingUsageHeadersForGeminiSupportsGatewayAuthVariants(t *testing.T) {
	headers := billingUsageHeaders(ProviderGemini, "test-key")

	require.Equal(t, "Bearer test-key", headers["Authorization"])
	require.Equal(t, "test-key", headers["x-goog-api-key"])
	require.Equal(t, "application/json", headers["Accept"])
}

func TestProbeBillingMultiplierFromUsageDeltaRequiresPositiveStandardDelta(t *testing.T) {
	probe := probeBillingMultiplierFromUsageDelta(
		&usageCostSnapshot{StandardCost: 10, ActualCost: 1},
		&usageCostSnapshot{StandardCost: 10, ActualCost: 1.07},
	)

	require.Nil(t, probe.Multiplier)
	require.Empty(t, probe.Source)
}

func TestProbeBillingMultiplierFromUsageSnapshotUsesCumulativeRatio(t *testing.T) {
	probe := probeBillingMultiplierFromUsageSnapshot(&usageCostSnapshot{StandardCost: 12.5, ActualCost: 1.375})

	require.NotNil(t, probe.Multiplier)
	require.InDelta(t, 0.11, *probe.Multiplier, 0.0001)
	require.Equal(t, "usage_snapshot_ratio", probe.Source)
}

func TestProbeBillingMultiplierAfterAuditRetriesUntilUsageDeltaAppears(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/usage", r.URL.Path)
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		attempt++
		payload := map[string]any{
			"usage": map[string]any{
				"total": map[string]any{
					"cost":        10,
					"actual_cost": 1,
				},
			},
		}
		if attempt >= 2 {
			usage, ok := payload["usage"].(map[string]any)
			require.True(t, ok)
			usage["total"] = map[string]any{
				"cost":        11,
				"actual_cost": 1.11,
			}
		}
		require.NoError(t, json.NewEncoder(w).Encode(payload))
	}))
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()

	probe := service.probeBillingMultiplierAfterAudit(context.Background(), server.Client(), ProviderOpenAI, server.URL, "sk-test", &usageCostSnapshot{StandardCost: 10, ActualCost: 1})

	require.NotNil(t, probe.Multiplier)
	require.InDelta(t, 0.11, *probe.Multiplier, 0.0001)
	require.Equal(t, "usage_delta", probe.Source)
	require.GreaterOrEqual(t, attempt, 2)
}

func TestProbeBillingMultiplierAfterAuditFallsBackToSnapshotRatio(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/usage", r.URL.Path)
		attempt++
		payload := map[string]any{
			"usage": map[string]any{
				"total": map[string]any{
					"cost":        12.5,
					"actual_cost": 1.375,
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(payload))
	}))
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()

	probe := service.probeBillingMultiplierAfterAudit(context.Background(), server.Client(), ProviderGemini, server.URL, "test-key", &usageCostSnapshot{StandardCost: 12.5, ActualCost: 1.375})

	require.NotNil(t, probe.Multiplier)
	require.InDelta(t, 0.11, *probe.Multiplier, 0.0001)
	require.Equal(t, "usage_snapshot_ratio", probe.Source)
	require.GreaterOrEqual(t, attempt, 8)
}
