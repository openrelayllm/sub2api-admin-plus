package purity

import (
	"context"
	"math"
	"net/http"
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
	after := s.captureBillingUsageSnapshot(ctx, client, provider, baseURL, apiKey)
	applyTokenAuditBillingMultiplierFromProbe(audit, probeBillingMultiplierFromUsageDelta(before, after))
}

func billingUsageHeaders(provider string, apiKey string) map[string]string {
	headers := map[string]string{"Accept": "application/json"}
	switch normalizeProvider(provider) {
	case ProviderAnthropic:
		headers["x-api-key"] = apiKey
	case ProviderGemini:
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

func parseUsageCostSnapshot(body []byte) (usageCostSnapshot, bool) {
	for _, prefix := range []string{"usage.total", "data.usage.total", "data.total", "total", "usage"} {
		standard := firstGJSONFloat(body,
			prefix+".cost",
			prefix+".total_cost",
			prefix+".standard_cost",
		)
		actual := firstGJSONFloat(body,
			prefix+".actual_cost",
			prefix+".total_actual_cost",
			prefix+".user_cost",
		)
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

func firstGJSONFloat(body []byte, paths ...string) *float64 {
	for _, path := range paths {
		result := gjson.GetBytes(body, path)
		if !result.Exists() {
			continue
		}
		value := result.Float()
		if !isFinite(value) {
			continue
		}
		return &value
	}
	return nil
}

func finitePositive(value float64) bool {
	return isFinite(value) && value > 0
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
