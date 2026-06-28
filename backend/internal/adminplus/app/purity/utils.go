package purity

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"
	"time"
)

func (s *Service) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func percent(score int, max int) int {
	if max <= 0 {
		return 0
	}
	return int(math.Round(float64(score) * 100 / float64(max)))
}

func tokensPerSecond(usage *TokenUsage, latencyMS int64) float64 {
	if usage == nil || usage.OutputTokens <= 0 || latencyMS <= 0 {
		return 0
	}
	return roundRatio(float64(usage.OutputTokens) / (float64(latencyMS) / 1000))
}

func roundRatio(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Round(value*100) / 100
}

func roundMoney(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Round(value*1_000_000) / 1_000_000
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func appendUniqueString(values []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func reportHash(report *PublicReport, baseURL string) string {
	if report == nil {
		return ""
	}
	raw := strings.Join([]string{
		report.Provider,
		baseURL,
		report.ModelID,
		report.CheckedAt.UTC().Format("2006-01-02T15:04"),
	}, "\x00")
	return sha256Hex(raw)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func roundProgress(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return math.Round(value*1000) / 1000
}

func upsertTokenAuditPartial(samples []TokenAuditSample, sample TokenAuditSample) []TokenAuditSample {
	out := make([]TokenAuditSample, 0, len(samples)+1)
	replaced := false
	for _, existing := range samples {
		if existing.Index == sample.Index {
			out = append(out, sample)
			replaced = true
			continue
		}
		out = append(out, existing)
	}
	if !replaced {
		out = append(out, sample)
	}
	return out
}
