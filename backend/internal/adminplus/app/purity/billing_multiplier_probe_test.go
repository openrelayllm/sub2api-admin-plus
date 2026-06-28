package purity

import (
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

func TestProbeBillingMultiplierFromUsageDeltaRequiresPositiveStandardDelta(t *testing.T) {
	probe := probeBillingMultiplierFromUsageDelta(
		&usageCostSnapshot{StandardCost: 10, ActualCost: 1},
		&usageCostSnapshot{StandardCost: 10, ActualCost: 1.07},
	)

	require.Nil(t, probe.Multiplier)
	require.Empty(t, probe.Source)
}
