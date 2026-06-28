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
		return CheckResult{
			ID:       "wrapper_fingerprint",
			Name:     "包装/反代指纹",
			Status:   CheckStatusFail,
			Score:    0,
			MaxScore: 0,
			Message:  "检测到模型、协议、签名或 usage/cache 混淆风险信号。",
			Details:  details,
		}
	}
	if report != nil && len(report.WrapperSignals) > 0 {
		return CheckResult{
			ID:       "wrapper_fingerprint",
			Name:     "包装/反代指纹",
			Status:   CheckStatusWarn,
			Score:    0,
			MaxScore: 0,
			Message:  "检测到透明中转或兼容网关信号，当前未发现混淆证据。",
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
