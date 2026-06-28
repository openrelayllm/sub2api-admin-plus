package purity

import "strings"

type validationSpec struct {
	ID       string
	Name     string
	CheckIDs []string
}

func validationSpecs() []validationSpec {
	return []validationSpec{
		{ID: "llm_fingerprint", Name: "LLM 指纹验证", CheckIDs: []string{"base_url", "models_schema"}},
		{ID: "schema_integrity", Name: "结构完整性", CheckIDs: []string{"responses_schema"}},
		{ID: "behavior", Name: "行为验证", CheckIDs: []string{"tool_call", "streaming"}},
		{ID: "signature", Name: "签名校验", CheckIDs: []string{"usage"}},
		{ID: "multimodal", Name: "多模态能力", CheckIDs: []string{"multimodal"}},
		{ID: "model_identity", Name: "模型身份验证", CheckIDs: []string{"model_identity"}},
		{ID: "wrapper_fingerprint", Name: "包装指纹验证", CheckIDs: []string{"wrapper_fingerprint"}},
		{ID: "token_audit", Name: "Token 用量审计", CheckIDs: []string{"token_audit"}},
	}
}

func buildLLMFingerprintValidation(baseURLCheck CheckResult, modelsCheck CheckResult) ValidationResult {
	validation := validationFromExecutedChecks("llm_fingerprint", "LLM 指纹验证", []CheckResult{baseURLCheck, modelsCheck})
	validation.Details["detector"] = "openai_base_url_and_models_probe"
	return validation
}

func validationFromExecutedChecks(id string, name string, checks []CheckResult) ValidationResult {
	out := ValidationResult{
		ID:              id,
		Name:            name,
		Status:          CheckStatusPass,
		RelatedCheckIDs: checkIDsFromResults(checks),
		Details: map[string]any{
			"detector": "programmatic_probe",
		},
	}
	messages := make([]string, 0, len(checks))
	evidence := make(map[string]any, len(checks))
	for _, check := range checks {
		evidence[check.ID] = map[string]any{
			"name":       check.Name,
			"status":     check.Status,
			"score":      check.Score,
			"max_score":  check.MaxScore,
			"message":    check.Message,
			"probe_data": check.Details,
		}
		messages = append(messages, check.Message)
		switch check.Status {
		case CheckStatusFail:
			out.Status = CheckStatusFail
		case CheckStatusWarn:
			if out.Status != CheckStatusFail {
				out.Status = CheckStatusWarn
			}
		}
	}
	out.Details["evidence"] = evidence
	out.Message = strings.Join(messages, " ")
	return out
}

func checkIDsFromResults(checks []CheckResult) []string {
	ids := make([]string, 0, len(checks))
	for _, check := range checks {
		ids = append(ids, check.ID)
	}
	return ids
}

func appendAndEmitChecks(report *PublicReport, emit PublicCheckEventSink, checks ...CheckResult) {
	if report == nil {
		return
	}
	for _, check := range checks {
		report.Checks = append(report.Checks, check)
		checkCopy := check
		emitPublicCheckEvent(emit, PublicCheckEvent{
			Type:     PublicCheckEventCheck,
			ReportID: report.ReportID,
			Check:    &checkCopy,
		})
	}
}

func skippedValidation(id string, name string, checkIDs []string, message string) ValidationResult {
	return ValidationResult{
		ID:              id,
		Name:            name,
		Status:          CheckStatusFail,
		Message:         message,
		RelatedCheckIDs: checkIDs,
		Details: map[string]any{
			"detector": "programmatic_probe",
			"skipped":  true,
		},
	}
}

func upsertValidation(report *PublicReport, validation ValidationResult) {
	if report == nil || validation.ID == "" {
		return
	}
	for i := range report.Validations {
		if report.Validations[i].ID == validation.ID {
			report.Validations[i] = validation
			return
		}
	}
	report.Validations = append(report.Validations, validation)
}

func upsertAndEmitValidation(report *PublicReport, emit PublicCheckEventSink, validation ValidationResult) {
	upsertValidation(report, validation)
	if report == nil {
		return
	}
	validationCopy := validation
	emitPublicCheckEvent(emit, PublicCheckEvent{
		Type:       PublicCheckEventValidation,
		ReportID:   report.ReportID,
		Validation: &validationCopy,
	})
}

func finalizeValidations(report *PublicReport) {
	if report == nil {
		return
	}
	byID := make(map[string]ValidationResult, len(report.Validations))
	for _, validation := range report.Validations {
		if validation.ID != "" {
			byID[validation.ID] = validation
		}
	}
	ordered := make([]ValidationResult, 0, len(validationSpecs()))
	seen := make(map[string]struct{}, len(validationSpecs())+len(report.Validations))
	for _, spec := range validationSpecs() {
		validation, ok := byID[spec.ID]
		if !ok {
			validation = skippedValidation(spec.ID, spec.Name, spec.CheckIDs, "该验证项未执行。")
		}
		ordered = append(ordered, validation)
		seen[spec.ID] = struct{}{}
	}
	for _, validation := range report.Validations {
		if validation.ID == "" {
			continue
		}
		if _, ok := seen[validation.ID]; ok {
			continue
		}
		ordered = append(ordered, validation)
		seen[validation.ID] = struct{}{}
	}
	report.Validations = ordered
}

func scoreBreakdown(report *PublicReport) map[string]int {
	if report == nil {
		return nil
	}
	out := map[string]int{
		"tag_check":       0,
		"structure":       0,
		"behavior":        0,
		"signature_proto": 0,
		"multimodal":      0,
	}
	if hasCheckStatus(report.Checks, "base_url", CheckStatusPass, CheckStatusWarn) ||
		hasCheckStatus(report.Checks, "models_schema", CheckStatusPass, CheckStatusWarn) ||
		hasCheckStatus(report.Checks, "claude_messages_schema", CheckStatusPass, CheckStatusWarn) ||
		hasCheckStatus(report.Checks, "gemini_models_schema", CheckStatusPass, CheckStatusWarn) {
		out["tag_check"] = validationWeightedScore(report, "llm_fingerprint", 10)
	}
	out["structure"] = validationWeightedScore(report, "schema_integrity", 20)
	out["behavior"] = validationWeightedScore(report, "behavior", 30)
	out["signature_proto"] = validationWeightedScore(report, "signature", 30)
	out["multimodal"] = validationWeightedScore(report, "multimodal", 10)
	if hasValidation(report.Validations, "token_audit") {
		out["token_audit"] = validationWeightedScore(report, "token_audit", 10)
	}
	return out
}

func officialScoreFromBreakdown(report *PublicReport, breakdown map[string]int, fallback int) int {
	if len(breakdown) == 0 {
		return fallback
	}
	coreScore := 0
	for key, value := range breakdown {
		if key == "token_audit" {
			continue
		}
		coreScore += value
	}
	if !tokenAuditAffectsScore(report) {
		if coreScore == 0 && fallback > 0 {
			return fallback
		}
		return coreScore
	}
	tokenScore := breakdown["token_audit"]
	return (coreScore*9+5)/10 + tokenScore
}

func tokenAuditAffectsScore(report *PublicReport) bool {
	if report == nil {
		return false
	}
	if report.TokenAudit != nil {
		return true
	}
	for _, check := range report.Checks {
		if check.ID != "token_audit" {
			continue
		}
		if check.Details != nil {
			if skipped, _ := check.Details["skipped"].(bool); skipped {
				return false
			}
		}
		return true
	}
	return false
}

func validationWeightedScore(report *PublicReport, validationID string, weight int) int {
	if report == nil || weight <= 0 {
		return 0
	}
	for _, validation := range report.Validations {
		if validation.ID != validationID {
			continue
		}
		switch validation.Status {
		case CheckStatusPass:
			return weight
		case CheckStatusWarn:
			return weight / 2
		default:
			return 0
		}
	}
	return 0
}

func hasValidation(validations []ValidationResult, id string) bool {
	for _, validation := range validations {
		if validation.ID == id {
			return true
		}
	}
	return false
}

func hasCheckStatus(checks []CheckResult, id string, statuses ...string) bool {
	for _, check := range checks {
		if check.ID != id {
			continue
		}
		for _, status := range statuses {
			if check.Status == status {
				return true
			}
		}
	}
	return false
}
