package provisionjobs

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/channelchecks"
)

func channelCheckInputFromSnapshot(supplierID int64, snapshot map[string]any) channelchecks.CheckInput {
	return channelchecks.CheckInput{
		SupplierID:                   supplierID,
		SupplierGroupID:              int64Value(snapshot["supplier_group_id"]),
		CandidateLimit:               intValue(snapshot["candidate_limit"]),
		AutoPauseOnFailure:           boolValue(snapshot["auto_pause_on_failure"], true),
		ProbeModel:                   stringValue(snapshot["probe_model"]),
		FirstTokenThresholdMS:        int64Value(snapshot["first_token_threshold_ms"]),
		TotalLatencyThresholdMS:      int64Value(snapshot["total_latency_threshold_ms"]),
		ActiveProbeDailyBudgetTokens: intValue(snapshot["active_probe_daily_budget_tokens"]),
		ActiveProbeEstimatedTokens:   intValue(snapshot["active_probe_estimated_tokens"]),
		ActiveProbeCooldownSeconds:   intValue(snapshot["active_probe_cooldown_seconds"]),
	}
}

func channelCheckResultSnapshot(result *channelchecks.CheckResult) map[string]any {
	if result == nil {
		return map[string]any{}
	}
	out := map[string]any{
		"supplier_id":                       result.SupplierID,
		"checked_at":                        result.CheckedAt.Format(time.RFC3339),
		"total":                             result.Total,
		"active_probe_budget_tokens":        result.ActiveProbeBudgetTokens,
		"active_probe_estimated_tokens":     result.ActiveProbeEstimatedTokens,
		"active_probe_tokens_used_today":    result.ActiveProbeTokensUsedToday,
		"active_probes_attempted":           result.ActiveProbesAttempted,
		"active_probes_skipped_by_budget":   result.ActiveProbesSkippedByBudget,
		"active_probes_skipped_by_cooldown": result.ActiveProbesSkippedByCooldown,
	}
	if result.Best != nil {
		out["best_supplier_group_id"] = result.Best.SupplierGroupID
		out["best_group_name"] = result.Best.GroupName
		out["best_effective_rate_multiplier"] = result.Best.EffectiveRateMultiplier
		out["best_first_token_ms"] = result.Best.FirstTokenMS
		out["best_duration_ms"] = result.Best.DurationMS
	}
	return out
}
