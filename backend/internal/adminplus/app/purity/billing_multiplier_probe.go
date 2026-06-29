package purity

import (
	"context"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

const billingMultiplierProbeTimeout = 8 * time.Second

type billingMultiplierProbe struct {
	Multiplier *float64
	Source     string
}

type usageCostSnapshot struct {
	StandardCost float64
	ActualCost   float64
}

func (s *Service) captureBillingUsageSnapshot(ctx context.Context, client *http.Client, provider string, baseURL string, apiKey string) *usageCostSnapshot {
	probeCtx, cancel := context.WithTimeout(ctx, billingMultiplierProbeTimeout)
	defer cancel()
	probe := s.doJSONWithHeaders(probeCtx, client, http.MethodGet, buildGatewayUsageURL(baseURL), nil, billingUsageHeaders(provider, apiKey), apiKey)
	if probe.StatusCode < 200 || probe.StatusCode >= 300 || len(probe.Body) == 0 {
		return nil
	}
	snapshot, ok := parseUsageCostSnapshot(probe.Body)
	if !ok {
		return nil
	}
	return &snapshot
}

func (s *Service) captureBillingUsageSnapshotForAudit(ctx context.Context, client *http.Client, provider string, baseURL string, apiKey string, options checkRunOptions) *usageCostSnapshot {
	if options.BillingMultiplier != nil {
		return nil
	}
	return s.captureBillingUsageSnapshot(ctx, client, provider, baseURL, apiKey)
}

func (s *Service) applyTokenAuditBillingMultiplierForAudit(ctx context.Context, client *http.Client, provider string, baseURL string, apiKey string, audit *TokenAuditReport, before *usageCostSnapshot, options checkRunOptions) {
	if options.BillingMultiplier != nil {
		applyTokenAuditBillingMultiplier(audit, options.BillingMultiplier)
		return
	}
	probe := s.probeBillingMultiplierAfterAudit(ctx, client, provider, baseURL, apiKey, before)
	applyTokenAuditBillingMultiplierFromProbe(audit, probe)
}

func (s *Service) probeBillingMultiplierAfterAudit(ctx context.Context, client *http.Client, provider string, baseURL string, apiKey string, before *usageCostSnapshot) billingMultiplierProbe {
	if before == nil {
		return billingMultiplierProbe{}
	}
	var lastAfter *usageCostSnapshot
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			if !sleepWithContext(ctx, time.Duration(attempt)*500*time.Millisecond) {
				return billingMultiplierProbe{}
			}
		}
		after := s.captureBillingUsageSnapshot(ctx, client, provider, baseURL, apiKey)
		if after != nil {
			lastAfter = after
		}
		probe := probeBillingMultiplierFromUsageDelta(before, after)
		if probe.Multiplier != nil {
			return probe
		}
	}
	if probe := probeBillingMultiplierFromUsageSnapshot(lastAfter); probe.Multiplier != nil {
		return probe
	}
	return probeBillingMultiplierFromUsageSnapshot(before)
}

func billingUsageHeaders(provider string, apiKey string) map[string]string {
	headers := map[string]string{"Accept": "application/json"}
	switch normalizeProvider(provider) {
	case ProviderAnthropic:
		headers["x-api-key"] = apiKey
	case ProviderGemini:
		headers["Authorization"] = "Bearer " + apiKey
		headers["x-goog-api-key"] = apiKey
	default:
		headers["Authorization"] = "Bearer " + apiKey
	}
	return headers
}

func probeBillingMultiplierFromUsageDelta(before *usageCostSnapshot, after *usageCostSnapshot) billingMultiplierProbe {
	if before == nil || after == nil {
		return billingMultiplierProbe{}
	}
	standardDelta := after.StandardCost - before.StandardCost
	actualDelta := after.ActualCost - before.ActualCost
	if !finitePositive(standardDelta) || actualDelta < 0 || !isFinite(actualDelta) {
		return billingMultiplierProbe{}
	}
	multiplier := roundRatio(actualDelta / standardDelta)
	if multiplier < 0 || !isFinite(multiplier) {
		return billingMultiplierProbe{}
	}
	return billingMultiplierProbe{
		Multiplier: &multiplier,
		Source:     "usage_delta",
	}
}

func probeBillingMultiplierFromUsageSnapshot(snapshot *usageCostSnapshot) billingMultiplierProbe {
	if snapshot == nil || !finitePositive(snapshot.StandardCost) || snapshot.ActualCost < 0 || !isFinite(snapshot.ActualCost) {
		return billingMultiplierProbe{}
	}
	multiplier := roundRatio(snapshot.ActualCost / snapshot.StandardCost)
	if multiplier < 0 || !isFinite(multiplier) {
		return billingMultiplierProbe{}
	}
	return billingMultiplierProbe{
		Multiplier: &multiplier,
		Source:     "usage_snapshot_ratio",
	}
}

func parseUsageCostSnapshot(body []byte) (usageCostSnapshot, bool) {
	for _, prefix := range []string{"usage.total", "data.usage.total", "data.total", "total", "usage.today", "data.usage.today", "usage", "summary", "data.summary", ""} {
		standard, standardPath := usageSnapshotStandardCost(body, prefix)
		actual := usageSnapshotActualCost(body, prefix, standardPath)
		if standard == nil || actual == nil {
			continue
		}
		if !isFinite(*standard) || !isFinite(*actual) {
			continue
		}
		return usageCostSnapshot{StandardCost: *standard, ActualCost: *actual}, true
	}
	return usageCostSnapshot{}, false
}

func usageSnapshotStandardCost(body []byte, prefix string) (*float64, string) {
	return firstGJSONFloatWithPath(body,
		usageSnapshotPath(prefix, "standard_cost"),
		usageSnapshotPath(prefix, "total_standard_cost"),
		usageSnapshotPath(prefix, "cost"),
		usageSnapshotPath(prefix, "total_cost"),
	)
}

func usageSnapshotActualCost(body []byte, prefix string, standardPath string) *float64 {
	actual := firstGJSONFloat(body,
		usageSnapshotPath(prefix, "actual_cost"),
		usageSnapshotPath(prefix, "total_actual_cost"),
		usageSnapshotPath(prefix, "account_cost"),
		usageSnapshotPath(prefix, "total_account_cost"),
	)
	if actual != nil {
		return actual
	}
	if strings.HasSuffix(standardPath, ".standard_cost") || strings.HasSuffix(standardPath, ".total_standard_cost") {
		actual = firstGJSONFloat(body, usageSnapshotPath(prefix, "cost"), usageSnapshotPath(prefix, "total_cost"))
		if actual != nil {
			return actual
		}
	}
	return firstGJSONFloat(body, usageSnapshotPath(prefix, "user_cost"), usageSnapshotPath(prefix, "total_user_cost"))
}

func usageSnapshotPath(prefix string, field string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}

func firstGJSONFloat(body []byte, paths ...string) *float64 {
	value, _ := firstGJSONFloatWithPath(body, paths...)
	return value
}

func firstGJSONFloatWithPath(body []byte, paths ...string) (*float64, string) {
	for _, path := range paths {
		result := gjson.GetBytes(body, path)
		if !result.Exists() {
			continue
		}
		value := result.Float()
		if !isFinite(value) {
			continue
		}
		return &value, path
	}
	return nil, ""
}

func finitePositive(value float64) bool {
	return isFinite(value) && value > 0
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func sleepWithContext(ctx context.Context, duration time.Duration) bool {
	if duration <= 0 {
		return true
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
