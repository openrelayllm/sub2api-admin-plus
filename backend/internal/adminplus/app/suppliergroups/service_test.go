package suppliergroups

import (
	"context"
	"testing"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/ports"
	"github.com/stretchr/testify/require"
)

func TestServiceSyncUpsertsGroupsAndMarksMissing(t *testing.T) {
	repo := NewMemoryRepository()
	session := &stubSessionReader{
		input: ports.SessionProbeInput{
			SupplierID: 7,
			Origin:     "https://relay.example.com",
			APIBaseURL: "https://relay.example.com/api/v1",
			Bundle:     map[string]any{"access_token": "token"},
		},
	}
	reader := &stubSessionGroupReader{
		results: []*ports.ReadGroupsResult{
			{
				SupplierID: 7,
				SystemType: "sub2api",
				Origin:     "https://relay.example.com",
				APIBaseURL: "https://relay.example.com/api/v1",
				CapturedAt: time.Date(2026, 6, 21, 1, 2, 3, 0, time.UTC),
				Groups: []*ports.ProviderGroup{
					{
						ExternalGroupID:         "10",
						Name:                    "Low Cost",
						ProviderFamily:          "openai",
						RateMultiplier:          0.8,
						EffectiveRateMultiplier: 0.7,
						UserRateMultiplier:      float64Ptr(0.7),
						Status:                  "active",
						RawPayload:              map[string]any{"id": 10},
					},
					{
						ExternalGroupID:         "11",
						Name:                    "Private",
						ProviderFamily:          "anthropic",
						RateMultiplier:          1.2,
						EffectiveRateMultiplier: 1.2,
						IsPrivate:               true,
						Status:                  "active",
					},
				},
			},
			{
				SupplierID: 7,
				SystemType: "sub2api",
				Origin:     "https://relay.example.com",
				APIBaseURL: "https://relay.example.com/api/v1",
				CapturedAt: time.Date(2026, 6, 21, 2, 2, 3, 0, time.UTC),
				Groups: []*ports.ProviderGroup{
					{
						ExternalGroupID:         "10",
						Name:                    "Low Cost Updated",
						ProviderFamily:          "openai",
						RateMultiplier:          0.9,
						EffectiveRateMultiplier: 0.9,
						Status:                  "active",
					},
				},
			},
		},
	}
	svc := NewService(repo, session, reader)

	first, err := svc.Sync(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, 2, first.Total)
	require.Len(t, first.Events, 2)
	require.Equal(t, adminplusdomain.SupplierGroupChangeDirectionNew, first.Events[0].Direction)
	require.True(t, first.Events[0].LowRate)
	require.Equal(t, "Low Cost", first.Groups[0].Name)
	require.NotNil(t, first.Groups[0].UserRateMultiplier)
	require.Equal(t, 0.7, *first.Groups[0].UserRateMultiplier)

	second, err := svc.Sync(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, 1, second.Total)
	require.Len(t, second.Events, 1)
	require.Equal(t, adminplusdomain.SupplierGroupChangeDirectionIncrease, second.Events[0].Direction)
	require.NotNil(t, second.Events[0].OldEffectiveRateMultiplier)
	require.Equal(t, 0.7, *second.Events[0].OldEffectiveRateMultiplier)
	require.Equal(t, 0.9, second.Events[0].NewEffectiveRateMultiplier)
	require.Equal(t, "Low Cost Updated", second.Groups[0].Name)

	all, err := svc.List(context.Background(), ListFilter{SupplierID: 7})
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, "10", all[0].ExternalGroupID)
	require.Equal(t, adminplusdomain.SupplierGroupStatusActive, all[0].Status)
	require.Equal(t, "11", all[1].ExternalGroupID)
	require.Equal(t, adminplusdomain.SupplierGroupStatusMissing, all[1].Status)

	events, err := svc.ListChangeEvents(context.Background(), EventFilter{SupplierID: 7})
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.Equal(t, adminplusdomain.SupplierGroupChangeDirectionIncrease, events[0].Direction)
}

type stubSessionReader struct {
	input ports.SessionProbeInput
}

func (s *stubSessionReader) DecryptedProbeInput(_ context.Context, _ int64) (ports.SessionProbeInput, error) {
	return s.input, nil
}

type stubSessionGroupReader struct {
	results []*ports.ReadGroupsResult
	calls   int
}

func (s *stubSessionGroupReader) ReadGroups(_ context.Context, _ ports.SessionProbeInput) (*ports.ReadGroupsResult, error) {
	result := s.results[s.calls]
	s.calls++
	return result, nil
}

func float64Ptr(value float64) *float64 {
	return &value
}
