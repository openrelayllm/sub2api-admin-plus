package suppliergroups

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/notifications"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/stretchr/testify/require"
)

func TestFeishuNotifierDispatchesOpenAISuperLowRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok"}`))
	}))
	t.Cleanup(server.Close)

	repo := notifications.NewMemoryRepository()
	service := notifications.NewService(repo)
	settings := service.Settings(context.Background())
	settings.Feishu.Enabled = true
	settings.Feishu.WebhookURL = server.URL
	require.NoError(t, repo.SaveSettings(context.Background(), settings))
	notifier := NewFeishuNotifier(service)

	err := notifier.NotifyGroupChange(context.Background(), &adminplusdomain.SupplierGroupChangeEvent{
		ID:                         11,
		SupplierID:                 7,
		ExternalGroupID:            "gpt-low",
		GroupName:                  "GPT 低价",
		ProviderFamily:             "openai",
		Direction:                  adminplusdomain.SupplierGroupChangeDirectionNew,
		NewEffectiveRateMultiplier: 0.05,
		CreatedAt:                  time.Date(2026, 6, 26, 9, 0, 0, 0, time.UTC),
	})

	require.NoError(t, err)
	items, err := repo.ListDeliveries(context.Background(), notifications.DeliveryFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, groupEventTypeSuperLowRate, items[0].EventType)
	require.Equal(t, adminplusdomain.NotificationStatusSucceeded, items[0].Status)
}

func TestFeishuNotifierDispatchesOpenAIPriceIncrease(t *testing.T) {
	repo := notifications.NewMemoryRepository()
	service := notifications.NewService(repo)
	settings := service.Settings(context.Background())
	settings.Feishu.Enabled = false
	require.NoError(t, repo.SaveSettings(context.Background(), settings))
	notifier := NewFeishuNotifier(service)
	oldRate := 0.06
	changePercent := 116.7

	err := notifier.NotifyGroupChange(context.Background(), &adminplusdomain.SupplierGroupChangeEvent{
		ID:                         12,
		SupplierID:                 7,
		ExternalGroupID:            "gpt-plus",
		GroupName:                  "GPT Plus",
		ProviderFamily:             "openai",
		Direction:                  adminplusdomain.SupplierGroupChangeDirectionIncrease,
		OldEffectiveRateMultiplier: &oldRate,
		NewEffectiveRateMultiplier: 0.13,
		ChangePercent:              &changePercent,
		CreatedAt:                  time.Date(2026, 6, 26, 9, 0, 0, 0, time.UTC),
	})

	require.NoError(t, err)
	items, err := repo.ListDeliveries(context.Background(), notifications.DeliveryFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, groupEventTypePriceIncrease, items[0].EventType)
	require.Equal(t, adminplusdomain.NotificationStatusSuppressed, items[0].Status)
	require.Equal(t, "channel_disabled", items[0].LastError)
}

func TestFeishuNotifierSkipsNonOpenAIAndBelowThresholdIncrease(t *testing.T) {
	repo := notifications.NewMemoryRepository()
	service := notifications.NewService(repo)
	settings := service.Settings(context.Background())
	settings.SupplierGroup.OpenAIPriceIncreaseRate = 0.2
	require.NoError(t, repo.SaveSettings(context.Background(), settings))
	notifier := NewFeishuNotifier(service)

	require.NoError(t, notifier.NotifyGroupChange(context.Background(), &adminplusdomain.SupplierGroupChangeEvent{
		ID:                         13,
		SupplierID:                 7,
		ExternalGroupID:            "claude",
		GroupName:                  "Claude",
		ProviderFamily:             "anthropic",
		Direction:                  adminplusdomain.SupplierGroupChangeDirectionNew,
		NewEffectiveRateMultiplier: 0.01,
		CreatedAt:                  time.Now().UTC(),
	}))
	require.NoError(t, notifier.NotifyGroupChange(context.Background(), &adminplusdomain.SupplierGroupChangeEvent{
		ID:                         14,
		SupplierID:                 7,
		ExternalGroupID:            "gpt-mid",
		GroupName:                  "GPT mid",
		ProviderFamily:             "openai",
		Direction:                  adminplusdomain.SupplierGroupChangeDirectionIncrease,
		NewEffectiveRateMultiplier: 0.13,
		CreatedAt:                  time.Now().UTC(),
	}))

	items, err := repo.ListDeliveries(context.Background(), notifications.DeliveryFilter{Limit: 10})
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestFeishuNotifierDispatchesWhenDecreaseCrossesSuperLowThreshold(t *testing.T) {
	repo := notifications.NewMemoryRepository()
	service := notifications.NewService(repo)
	settings := service.Settings(context.Background())
	settings.Feishu.Enabled = false
	require.NoError(t, repo.SaveSettings(context.Background(), settings))
	notifier := NewFeishuNotifier(service)
	oldRate := 0.08

	err := notifier.NotifyGroupChange(context.Background(), &adminplusdomain.SupplierGroupChangeEvent{
		ID:                         15,
		SupplierID:                 7,
		ExternalGroupID:            "gpt-drop",
		GroupName:                  "GPT drop",
		ProviderFamily:             "openai",
		Direction:                  adminplusdomain.SupplierGroupChangeDirectionDecrease,
		OldEffectiveRateMultiplier: &oldRate,
		NewEffectiveRateMultiplier: 0.05,
		CreatedAt:                  time.Now().UTC(),
	})

	require.NoError(t, err)
	items, err := repo.ListDeliveries(context.Background(), notifications.DeliveryFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, groupEventTypeSuperLowRate, items[0].EventType)
}
