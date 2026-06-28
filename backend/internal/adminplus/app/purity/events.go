package purity

func emitMetrics(report *PublicReport, emit PublicCheckEventSink) {
	if report == nil {
		return
	}
	metrics := report.Metrics
	syncMetricsCompat(&metrics)
	emitPublicCheckEvent(emit, PublicCheckEvent{
		Type:     PublicCheckEventMetrics,
		ReportID: report.ReportID,
		Metrics:  &metrics,
	})
}

func emitProgress(report *PublicReport, emit PublicCheckEventSink, step int, stepName string) {
	if report == nil {
		return
	}
	if step < 1 {
		step = 1
	}
	if step > 7 {
		step = 7
	}
	report.Status = RunStatusRunning
	report.Step = step
	report.StepName = stepName
	report.Progress = roundProgress(float64(step) / 7)
	report.Scores = scoreBreakdown(report)
	syncReportCompat(report)
	emitPublicCheckEvent(emit, PublicCheckEvent{
		Type:               PublicCheckEventProgress,
		ReportID:           report.ReportID,
		Status:             report.Status,
		Step:               report.Step,
		StepName:           report.StepName,
		Progress:           report.Progress,
		Scores:             cloneScores(report.Scores),
		TokenAuditProgress: report.TokenAuditProgress,
		TokenAuditPartial:  append([]TokenAuditSample(nil), report.TokenAuditPartial...),
		Report:             clonePublicReport(report),
	})
}

func emitFinalReport(report *PublicReport, emit PublicCheckEventSink) {
	if report == nil {
		return
	}
	if report.Status == RunStatusError {
		if report.Step <= 0 {
			report.Step = 1
		}
		if report.StepName == "" {
			report.StepName = "tag"
		}
		if report.Progress <= 0 {
			report.Progress = roundProgress(float64(report.Step) / 7)
		}
	} else {
		report.Status = RunStatusDone
		report.Step = 7
		report.StepName = "evaluate"
		report.Progress = 1
	}
	report.Scores = scoreBreakdown(report)
	syncReportCompat(report)
	emitPublicCheckEvent(emit, PublicCheckEvent{
		Type:     PublicCheckEventReport,
		ReportID: report.ReportID,
		Status:   report.Status,
		Step:     report.Step,
		StepName: report.StepName,
		Progress: report.Progress,
		Scores:   cloneScores(report.Scores),
		Report:   clonePublicReport(report),
	})
}

func emitPublicCheckEvent(emit PublicCheckEventSink, event PublicCheckEvent) {
	if emit != nil {
		if event.StepNameCompat == "" {
			event.StepNameCompat = event.StepName
		}
		if event.Metrics != nil {
			syncMetricsCompat(event.Metrics)
		}
		if event.Report != nil {
			syncReportCompat(event.Report)
		}
		emit(event)
	}
}

func clonePublicReport(report *PublicReport) *PublicReport {
	if report == nil {
		return nil
	}
	value := *report
	value.Scores = cloneScores(report.Scores)
	value.WrapperSignals = append([]string(nil), report.WrapperSignals...)
	value.WrapperSignalsCompat = append([]string(nil), report.WrapperSignalsCompat...)
	value.ModelIdentity = cloneModelIdentity(report.ModelIdentity)
	value.ModelIdentityCompat = cloneModelIdentity(report.ModelIdentityCompat)
	value.TokenAuditPartial = append([]TokenAuditSample(nil), report.TokenAuditPartial...)
	syncReportCompat(&value)
	return &value
}

func cloneScores(scores map[string]int) map[string]int {
	if len(scores) == 0 {
		return nil
	}
	out := make(map[string]int, len(scores))
	for key, value := range scores {
		out[key] = value
	}
	return out
}
