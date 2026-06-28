package purity

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceRunPublicCheck_BlocksPrivateBaseURL(t *testing.T) {
	service := NewService(nil)
	service.limiter = nil

	_, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: "http://127.0.0.1:8080",
		APIKey:     "sk-test",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "PURITY_BASE_URL_INVALID")
}
func TestServiceRunPublicCheck_AllowsPrivateBaseURLForTrustedLocalClient(t *testing.T) {
	server := newOpenAIStoreIncludeServer(t, http.StatusOK, "", nil)
	defer server.Close()

	service := NewService(nil)
	service.accountHTTPClient = server.Client()
	service.limiter = nil
	service.now = func() time.Time { return time.Date(2026, 6, 28, 15, 0, 0, 0, time.UTC) }

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:            ProviderOpenAI,
		APIBaseURL:          server.URL,
		APIKey:              "sk-store-include",
		ModelID:             "gpt-5.4",
		ClientIP:            "127.0.0.1",
		SkipTokenAudit:      true,
		AllowPrivateBaseURL: true,
	})

	require.NoError(t, err)
	require.Equal(t, ProviderOpenAI, report.Provider)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "responses_schema").Status)
}
