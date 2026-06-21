package billing

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/ports"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestServiceImportBillLinesNormalizesSupplierBill(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	startedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	lines, err := svc.ImportBillLines(context.Background(), []ImportBillLineInput{
		{
			SupplierID:        7,
			Source:            "Chrome",
			ExternalBillID:    "bill-1",
			ExternalRequestID: "req-1",
			APIKeyName:        "sk-prod",
			Model:             "gpt-4o-mini",
			Endpoint:          "/v1/chat/completions",
			RequestType:       "chat",
			BillingMode:       "token",
			ReasoningEffort:   "low",
			Currency:          "usd",
			CostCents:         123,
			InputTokens:       1000,
			OutputTokens:      500,
			CacheReadTokens:   200,
			FirstTokenMS:      680,
			DurationMS:        2200,
			UserAgent:         "OpenAI/Python",
			StartedAt:         startedAt,
		},
	})

	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, int64(1), lines[0].ID)
	require.Equal(t, "chrome", lines[0].Source)
	require.Equal(t, "USD", lines[0].Currency)
	require.Equal(t, int64(123), lines[0].CostCents)
	require.Equal(t, "sk-prod", lines[0].APIKeyName)
	require.Equal(t, "/v1/chat/completions", lines[0].Endpoint)
	require.Equal(t, "token", lines[0].BillingMode)
	require.Equal(t, int64(1700), lines[0].TotalTokens)
	require.Equal(t, int64(680), lines[0].FirstTokenMS)
}

func TestServiceImportBillLinesValidatesInput(t *testing.T) {
	svc := NewService(NewMemoryRepository())

	_, err := svc.ImportBillLines(context.Background(), []ImportBillLineInput{
		{
			SupplierID: 7,
			Model:      "gpt-4o-mini",
			CostCents:  -1,
			StartedAt:  time.Now(),
		},
	})

	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "BILLING_COST_INVALID", infraerrors.Reason(err))
}

func TestServiceImportBillLinesValidatesDetailMetrics(t *testing.T) {
	svc := NewService(NewMemoryRepository())

	_, err := svc.ImportBillLines(context.Background(), []ImportBillLineInput{
		{
			SupplierID: 7,
			Model:      "gpt-4o-mini",
			CostCents:  1,
			DurationMS: -1,
			StartedAt:  time.Now(),
		},
	})

	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "BILLING_LATENCY_INVALID", infraerrors.Reason(err))
}

func TestServiceSyncFromSessionImportsProviderBillingLines(t *testing.T) {
	repo := NewMemoryRepository()
	startedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(24 * time.Hour)
	session := &stubBillingSessionReader{input: ports.SessionProbeInput{
		SupplierID: 7,
		Origin:     "https://relay.example.com",
		APIBaseURL: "https://relay.example.com/api/v1",
		Bundle:     map[string]any{"access_token": "browser-token"},
	}}
	reader := &stubBillingAdapter{result: &ports.ReadBillingResult{
		SupplierID: 7,
		SystemType: "sub2api",
		Origin:     "https://relay.example.com",
		APIBaseURL: "https://relay.example.com/api/v1",
		CapturedAt: startedAt.Add(time.Hour),
		Lines: []ports.ProviderBillLine{
			{
				ExternalBillID:    "bill-1",
				ExternalRequestID: "req-1",
				APIKeyName:        "ops-key",
				Model:             "gpt-5-mini",
				Endpoint:          "/v1/responses",
				RequestType:       "responses",
				BillingMode:       "token",
				Currency:          "usd",
				CostCents:         123,
				InputTokens:       1000,
				OutputTokens:      500,
				StartedAt:         startedAt.Add(time.Hour),
				RawPayload:        map[string]any{"id": "bill-1"},
			},
		},
	}}
	svc := NewServiceWithDependencies(repo, session, reader)

	result, err := svc.SyncFromSession(context.Background(), SyncFromSessionInput{
		SupplierID: 7,
		StartedAt:  startedAt,
		EndedAt:    endedAt,
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), session.supplierID)
	require.Equal(t, startedAt, reader.request.StartedAt)
	require.Equal(t, endedAt, reader.request.EndedAt)
	require.Equal(t, "sub2api", result.SystemType)
	require.Equal(t, "provider_session", result.Items[0].Source)
	require.Equal(t, "USD", result.Items[0].Currency)
	require.Equal(t, int64(1500), result.Items[0].TotalTokens)
	require.Equal(t, int64(1), result.Items[0].ID)
}

func TestServiceSyncFromSessionAllowsEmptyProviderBillingLines(t *testing.T) {
	startedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	session := &stubBillingSessionReader{input: ports.SessionProbeInput{
		SupplierID: 7,
		Origin:     "https://relay.example.com",
		APIBaseURL: "https://relay.example.com/api/v1",
		Bundle:     map[string]any{"access_token": "browser-token"},
	}}
	reader := &stubBillingAdapter{result: &ports.ReadBillingResult{
		SupplierID: 7,
		SystemType: "sub2api",
		Origin:     "https://relay.example.com",
		APIBaseURL: "https://relay.example.com/api/v1",
		CapturedAt: startedAt.Add(time.Hour),
		Lines:      []ports.ProviderBillLine{},
	}}
	svc := NewServiceWithDependencies(NewMemoryRepository(), session, reader)

	result, err := svc.SyncFromSession(context.Background(), SyncFromSessionInput{
		SupplierID: 7,
		StartedAt:  startedAt,
		EndedAt:    startedAt.Add(24 * time.Hour),
	})

	require.NoError(t, err)
	require.Equal(t, 0, result.Total)
	require.Empty(t, result.Items)
}

func TestServiceSyncFromSessionRequiresDependencies(t *testing.T) {
	startedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	svc := NewService(NewMemoryRepository())

	_, err := svc.SyncFromSession(context.Background(), SyncFromSessionInput{
		SupplierID: 7,
		StartedAt:  startedAt,
		EndedAt:    startedAt.Add(time.Hour),
	})

	require.Error(t, err)
	require.Equal(t, http.StatusInternalServerError, infraerrors.Code(err))
	require.Equal(t, "ADMIN_PLUS_INTERNAL_ERROR", infraerrors.Reason(err))
}

type stubBillingSessionReader struct {
	input      ports.SessionProbeInput
	supplierID int64
}

func (r *stubBillingSessionReader) DecryptedProbeInput(_ context.Context, supplierID int64) (ports.SessionProbeInput, error) {
	r.supplierID = supplierID
	return r.input, nil
}

type stubBillingAdapter struct {
	result  *ports.ReadBillingResult
	request ports.ReadBillingInput
}

func (r *stubBillingAdapter) ReadBilling(_ context.Context, _ ports.SessionProbeInput, request ports.ReadBillingInput) (*ports.ReadBillingResult, error) {
	r.request = request
	return r.result, nil
}
