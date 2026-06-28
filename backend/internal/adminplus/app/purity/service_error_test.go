package purity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceRunPublicCheck_RedactsAPIKeyFromReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(t, w, map[string]any{
			"error": map[string]any{"message": "bad key sk-secret-value"},
		})
	}))
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: server.URL,
		APIKey:     "sk-secret-value",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})
	require.NoError(t, err)
	require.Equal(t, RunStatusError, report.Status)
	require.Contains(t, report.Error, "[redacted]")
	raw, err := json.Marshal(report)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "sk-secret-value")
	require.Contains(t, string(raw), "[redacted]")
}

func TestServiceRunPublicCheck_InsufficientBalanceIsFatalAccountState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeJSON(t, w, map[string]any{
			"error": map[string]any{"message": "Insufficient account balance"},
		})
	}))
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: server.URL,
		APIKey:     "sk-test",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})
	require.NoError(t, err)
	require.Equal(t, RunStatusError, report.Status)
	require.Equal(t, VerdictInvalidOrUnavailable, report.Verdict)
	require.Equal(t, 0, report.Score)
	require.Equal(t, 0, report.CompatibilityScore)
	require.Equal(t, 0, report.OfficialScore)
	require.Equal(t, errorClassAccountBalanceInsufficient, report.Metrics.ErrorClass)
	require.Contains(t, report.Error, "账号余额不足")
	require.Contains(t, report.Summary, "账号余额不足")
	require.Equal(t, CheckStatusFail, findCheck(t, report, "models_schema").Status)
	require.Contains(t, findCheck(t, report, "models_schema").Message, "账号余额不足")
}
