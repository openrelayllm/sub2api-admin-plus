package purity

func appendAndEmitWrapperFingerprint(report *PublicReport, emit PublicCheckEventSink) {
	check := buildWrapperFingerprintCheck(report)
	appendAndEmitChecks(report, emit, check)
	validation := validationFromExecutedChecks("wrapper_fingerprint", "包装指纹验证", []CheckResult{check})
	validation.Details["detector"] = "channel_signal_detectors"
	upsertAndEmitValidation(report, emit, validation)
}

func buildWrapperFingerprintCheck(report *PublicReport) CheckResult {
	details := map[string]any{}
	if report != nil {
		details["wrapper_signals"] = append([]string(nil), report.WrapperSignals...)
	}
	obfuscationSignals := wrapperObfuscationSignals(report)
	if len(obfuscationSignals) > 0 {
		details["obfuscation_signals"] = obfuscationSignals
		if attributionHasReasonCode(report, "bedrock_anthropic_signature_mask") {
			details["reason_code"] = "bedrock_anthropic_signature_mask"
			details["case_id"] = "PURITY-BEDROCK-MASK-001"
			details["score_penalty"] = 5
			details["client_impact"] = clientImpactNone
			details["impact_scope"] = "channel_attribution_only"
			details["evidence"] = []string{"bedrock_metadata_family_present", "anthropic_native_metadata_present"}
		}
		return CheckResult{
			ID:       "wrapper_fingerprint",
			Name:     "包装/反代指纹",
			Status:   CheckStatusFail,
			Score:    0,
			MaxScore: 0,
			Message:  wrapperObfuscationMessage(report),
			Details:  details,
		}
	}
	if report != nil && len(report.WrapperSignals) > 0 {
		return CheckResult{
			ID:       "wrapper_fingerprint",
			Name:     "包装/反代指纹",
			Status:   CheckStatusPass,
			Score:    0,
			MaxScore: 0,
			Message:  "检测到透明中转或兼容网关信号，当前未发现模型、协议、签名或 usage/cache 混淆证据。",
			Details:  details,
		}
	}
	return CheckResult{
		ID:       "wrapper_fingerprint",
		Name:     "包装/反代指纹",
		Status:   CheckStatusPass,
		Score:    0,
		MaxScore: 0,
		Message:  "未检测到包装、反代或兼容网关指纹。",
		Details:  details,
	}
}

func wrapperObfuscationMessage(report *PublicReport) string {
	if attributionHasReasonCode(report, "bedrock_anthropic_signature_mask") {
		return "判例 PURITY-BEDROCK-MASK-001：签名同时命中 AWS Bedrock 来源字段与 Anthropic 原生元数据字段。上游识别为 AWS Bedrock；本轮客户端能力探针均未受影响，仅按来源透明度扣 5 分。"
	}
	return "检测到模型、协议、签名或 usage/cache 混淆风险信号。"
}
