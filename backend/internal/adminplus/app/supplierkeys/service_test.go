package supplierkeys

import (
	"context"
	"testing"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/ports"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestServiceProvisionCreatesProviderKeyLocalAccountAndBinding(t *testing.T) {
	repo := NewMemoryRepository()
	repo.PutSupplier(&adminplusdomain.Supplier{
		ID:            7,
		Name:          "Relay",
		Type:          adminplusdomain.SupplierTypeSub2API,
		RuntimeStatus: adminplusdomain.SupplierRuntimeStatusMonitorOnly,
		HealthStatus:  adminplusdomain.SupplierHealthStatusNormal,
	})
	repo.PutGroup(&adminplusdomain.SupplierGroup{
		ID:              10,
		SupplierID:      7,
		ExternalGroupID: "88",
		Name:            "Low Cost",
		ProviderFamily:  "openai",
		Status:          adminplusdomain.SupplierGroupStatusActive,
	})
	session := &stubSessionReader{
		input: ports.SessionProbeInput{
			SupplierID: 7,
			APIBaseURL: "https://relay.example.com/api/v1",
			Bundle:     map[string]any{"access_token": "browser-token"},
		},
	}
	keyAdapter := &stubKeyAdapter{
		result: &ports.ProviderKeyResult{
			SupplierID:      7,
			ExternalGroupID: "88",
			ExternalKeyID:   "99",
			Name:            "ops-key",
			Secret:          "sk-provider-secret",
			Status:          "active",
			RawPayload:      map[string]any{"id": 99},
			CreatedAt:       time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC),
		},
	}
	local := &stubLocalAccountCreator{}
	svc := NewService(repo, session, keyAdapter, local)
	rate := 0.7

	result, err := svc.Provision(context.Background(), ProvisionKeyInput{
		SupplierID:                 7,
		SupplierGroupID:            10,
		Name:                       "ops-key",
		QuotaUSD:                   25,
		LocalAccountPlatform:       service.PlatformOpenAI,
		LocalAccountName:           "local-upstream",
		LocalAccountBaseURL:        "https://relay.example.com/v1",
		LocalAccountConcurrency:    3,
		LocalAccountPriority:       40,
		LocalAccountRateMultiplier: &rate,
		RuntimeStatus:              adminplusdomain.SupplierRuntimeStatusMonitorOnly,
		HealthStatus:               adminplusdomain.SupplierHealthStatusNormal,
		BalanceCurrency:            "USD",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Key)
	require.NotNil(t, result.Binding)
	require.Equal(t, adminplusdomain.SupplierKeyStatusBound, result.Key.Status)
	require.Equal(t, "99", result.Key.ExternalKeyID)
	require.Equal(t, "cret", result.Key.KeyLast4)
	require.NotEqual(t, "sk-provider-secret", result.Key.KeyFingerprint)
	require.Equal(t, int64(1001), result.Key.LocalSub2APIAccountID)
	require.Equal(t, int64(1001), result.Binding.LocalSub2APIAccountID)
	require.Equal(t, result.Key.ID, result.Binding.SupplierKeyID)
	require.Equal(t, "openai", local.input.Platform)
	require.Equal(t, service.AccountTypeAPIKey, local.input.Type)
	require.Equal(t, "sk-provider-secret", local.input.Credentials["api_key"])
	require.Equal(t, "https://relay.example.com/v1", local.input.Credentials["base_url"])
	require.Equal(t, []ports.CreateProviderKeyInput{{
		SupplierID:      7,
		ExternalGroupID: "88",
		Name:            "ops-key",
		QuotaUSD:        25,
		ExpiresInDays:   nil,
		Metadata: map[string]any{
			"supplier_group_id": int64(10),
			"provider_family":   "openai",
		},
	}}, keyAdapter.calls)
}

type stubSessionReader struct {
	input ports.SessionProbeInput
}

func (s *stubSessionReader) DecryptedProbeInput(_ context.Context, _ int64) (ports.SessionProbeInput, error) {
	return s.input, nil
}

type stubKeyAdapter struct {
	result *ports.ProviderKeyResult
	calls  []ports.CreateProviderKeyInput
}

func (s *stubKeyAdapter) CreateKey(_ context.Context, _ ports.SessionProbeInput, request ports.CreateProviderKeyInput) (*ports.ProviderKeyResult, error) {
	s.calls = append(s.calls, request)
	return s.result, nil
}

type stubLocalAccountCreator struct {
	input service.CreateAccountInput
}

func (s *stubLocalAccountCreator) CreateAccount(_ context.Context, input *service.CreateAccountInput) (*service.Account, error) {
	s.input = *input
	return &service.Account{
		ID:          1001,
		Name:        input.Name,
		Platform:    input.Platform,
		Type:        input.Type,
		Credentials: input.Credentials,
	}, nil
}
