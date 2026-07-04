package kanban

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
)

const maxParsedMarketPrices = 50

var marketPriceAmountPattern = regexp.MustCompile(`(?i)(\$|¥|usd|cny|rmb)\s*([0-9]+(?:\.[0-9]+)?)|([0-9]+(?:\.[0-9]+)?)\s*(usd|cny|rmb|美元|人民币|元)`)
var marketPriceQuotaPattern = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?\s*(?:m|k|million|thousand)?\s*(?:tokens?|token|次|额度))`)
var marketPricePercentPattern = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*%`)
var marketPriceMultiplierPattern = regexp.MustCompile(`(?i)(?:倍率|rate|multiplier)\s*[:：]?\s*([0-9]+(?:\.[0-9]+)?)\s*x?`)
var marketModelTokenPattern = regexp.MustCompile(`(?i)\b(?:gpt|claude|gemini|llama|deepseek|qwen)[a-z0-9._:/-]*\b`)

type MarketPriceParseInput struct {
	SourceType      string
	SourceName      string
	SourceURL       string
	SiteID          int64
	SupplierID      int64
	DefaultCurrency string
	Confidence      float64
	Text            string
	ObservedAt      *time.Time
}

type MarketPriceParseResult struct {
	Items []*adminplusdomain.MarketPriceSnapshot `json:"items"`
	Total int                                    `json:"total"`
}

func (s *Service) ParseMarketPrices(ctx context.Context, in MarketPriceParseInput) (*MarketPriceParseResult, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("kanban service is not configured")
	}
	inputs, err := s.marketPriceInputsFromText(in)
	if err != nil {
		return nil, err
	}
	items := make([]*adminplusdomain.MarketPriceSnapshot, 0, len(inputs))
	for _, input := range inputs {
		created, err := s.RecordMarketPrice(ctx, input)
		if err != nil {
			return nil, err
		}
		items = append(items, created)
	}
	s.recordMarketTextEvents(ctx, in)
	return &MarketPriceParseResult{Items: items, Total: len(items)}, nil
}

func (s *Service) marketPriceInputsFromText(in MarketPriceParseInput) ([]MarketPriceInput, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return nil, badRequest("KANBAN_PRICE_TEXT_REQUIRED", "price text is required")
	}
	defaultCurrency := normalizeCurrency(in.DefaultCurrency)
	confidence := in.Confidence
	if confidence <= 0 {
		confidence = 0.7
	}
	if confidence > 1 {
		return nil, badRequest("KANBAN_CONFIDENCE_INVALID", "confidence must be between 0 and 1")
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	items := make([]MarketPriceInput, 0)
	currentModel := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		matches := marketPriceAmountPattern.FindAllStringSubmatchIndex(line, -1)
		if len(matches) == 0 {
			if looksLikeModelHeading(line) {
				currentModel = cleanParsedModel(line)
			}
			continue
		}
		for _, match := range matches {
			if len(items) >= maxParsedMarketPrices {
				break
			}
			if amountLooksLikePackageOrRecharge(line, match[0]) {
				continue
			}
			amount, currency, ok := parsedAmount(line, match, defaultCurrency)
			if !ok || amount < 0 {
				continue
			}
			contextText := strings.TrimSpace(line)
			model := cleanParsedModel(line[:match[0]])
			if model == "" {
				model = currentModel
			}
			if model == "" {
				continue
			}
			packageLabel, packagePriceCents, packageQuota := parsedPackageFields(contextText, defaultCurrency)
			minRechargeCents := parsedMinRechargeCents(contextText, defaultCurrency)
			bonusPercent := parsedBonusPercent(contextText)
			rateMultiplier := parsedRateMultiplier(contextText)
			priceItem := inferParsedPriceItem(contextText)
			priceMicros := parsedPriceMicros(amount, contextText)
			items = append(items, MarketPriceInput{
				SourceType:        in.SourceType,
				SourceName:        in.SourceName,
				SourceURL:         in.SourceURL,
				SiteID:            in.SiteID,
				SupplierID:        in.SupplierID,
				Model:             model,
				BillingMode:       "tokens",
				PriceItem:         priceItem,
				Unit:              "1m_tokens",
				Currency:          currency,
				PriceMicros:       priceMicros,
				PackageLabel:      packageLabel,
				PackagePriceCents: packagePriceCents,
				PackageQuota:      packageQuota,
				RateMultiplier:    rateMultiplier,
				MinRechargeCents:  minRechargeCents,
				BonusPercent:      bonusPercent,
				Confidence:        confidence,
				ObservedAt:        in.ObservedAt,
				RawPayload: map[string]any{
					"line":   line,
					"parser": "market_price_text_v1",
				},
			})
		}
	}
	if len(items) == 0 {
		return nil, badRequest("KANBAN_PRICE_TEXT_UNPARSEABLE", "no market prices were found in text")
	}
	return items, nil
}

func parsedAmount(line string, match []int, defaultCurrency string) (float64, string, bool) {
	if match[2] >= 0 && match[3] >= 0 && match[4] >= 0 && match[5] >= 0 {
		amount, err := strconv.ParseFloat(line[match[4]:match[5]], 64)
		return amount, currencyFromPriceToken(line[match[2]:match[3]], defaultCurrency), err == nil
	}
	if match[6] >= 0 && match[7] >= 0 {
		amount, err := strconv.ParseFloat(line[match[6]:match[7]], 64)
		currency := defaultCurrency
		if match[8] >= 0 && match[9] >= 0 {
			currency = currencyFromPriceToken(line[match[8]:match[9]], defaultCurrency)
		}
		return amount, currency, err == nil
	}
	return 0, defaultCurrency, false
}

func currencyFromPriceToken(token string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "¥", "cny", "rmb", "元", "人民币":
		return "CNY"
	case "$", "usd", "美元":
		return "USD"
	default:
		return normalizeCurrency(fallback)
	}
}

func parsedPriceMicros(amount float64, line string) int64 {
	scale := 1.0
	lower := strings.ToLower(line)
	if strings.Contains(lower, "1k") || strings.Contains(lower, "/k") || strings.Contains(line, "千") {
		scale = 1000
	}
	return int64(amount*scale*1_000_000 + 0.5)
}

func amountLooksLikePackageOrRecharge(line string, amountStart int) bool {
	prefix := amountPrefix(line, amountStart, 32)
	return strings.Contains(prefix, "package") || strings.Contains(prefix, "subscription") || strings.Contains(prefix, "min recharge") || strings.Contains(prefix, "recharge") || strings.Contains(prefix, "套餐") || strings.Contains(prefix, "最低充值") || strings.Contains(prefix, "充值")
}

func amountLooksLikePackage(line string, amountStart int) bool {
	prefix := amountPrefix(line, amountStart, 32)
	return strings.Contains(prefix, "package") || strings.Contains(prefix, "subscription") || strings.Contains(prefix, "套餐")
}

func amountLooksLikeMinRecharge(line string, amountStart int) bool {
	prefix := amountPrefix(line, amountStart, 48)
	return strings.Contains(prefix, "min recharge") || strings.Contains(prefix, "最低充值")
}

func amountPrefix(line string, amountStart int, window int) string {
	start := amountStart - 32
	if start < 0 {
		start = 0
	}
	if window > 0 {
		start = amountStart - window
		if start < 0 {
			start = 0
		}
	}
	return strings.ToLower(line[start:amountStart])
}

func parsedPackageFields(line string, defaultCurrency string) (string, *int64, string) {
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "package") && !strings.Contains(lower, "subscription") && !strings.Contains(line, "套餐") {
		return "", nil, ""
	}
	label := "套餐"
	if strings.Contains(lower, "subscription") {
		label = "subscription"
	} else if strings.Contains(lower, "package") {
		label = "package"
	}
	var priceCents *int64
	for _, match := range marketPriceAmountPattern.FindAllStringSubmatchIndex(line, -1) {
		if !amountLooksLikePackage(line, match[0]) {
			continue
		}
		amount, _, ok := parsedAmount(line, match, defaultCurrency)
		if ok {
			value := int64(amount*100 + 0.5)
			priceCents = &value
			break
		}
	}
	quota := ""
	for _, raw := range marketPriceQuotaPattern.FindAllString(line, -1) {
		if !strings.Contains(strings.ToLower(raw), "token") && !strings.Contains(raw, "额度") && !strings.Contains(raw, "次") {
			continue
		}
		quota = strings.TrimSpace(raw)
		break
	}
	return label, priceCents, quota
}

func parsedMinRechargeCents(line string, defaultCurrency string) *int64 {
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "min recharge") && !strings.Contains(line, "最低充值") {
		return nil
	}
	for _, match := range marketPriceAmountPattern.FindAllStringSubmatchIndex(line, -1) {
		if !amountLooksLikeMinRecharge(line, match[0]) {
			continue
		}
		amount, _, ok := parsedAmount(line, match, defaultCurrency)
		if ok {
			value := int64(amount*100 + 0.5)
			return &value
		}
	}
	return nil
}

func parsedBonusPercent(line string) *float64 {
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "bonus") && !strings.Contains(lower, "gift") && !strings.Contains(line, "赠送") && !strings.Contains(line, "赠") {
		return nil
	}
	match := marketPricePercentPattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return nil
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return nil
	}
	return &value
}

func parsedRateMultiplier(line string) *float64 {
	match := marketPriceMultiplierPattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return nil
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return nil
	}
	return &value
}

func inferParsedPriceItem(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "cache read") || strings.Contains(lower, "cached input") || strings.Contains(line, "缓存读") || strings.Contains(line, "读缓存"):
		return "cache_read"
	case strings.Contains(lower, "cache write") || strings.Contains(lower, "cache creation") || strings.Contains(line, "缓存写") || strings.Contains(line, "写缓存"):
		return "cache_write"
	case strings.Contains(lower, "output") || strings.Contains(lower, "completion") || strings.Contains(line, "输出"):
		return "output"
	case strings.Contains(lower, "input") || strings.Contains(lower, "prompt") || strings.Contains(line, "输入"):
		return "input"
	default:
		return "blended"
	}
}

func cleanParsedModel(value string) string {
	model := strings.TrimSpace(value)
	replacements := []string{"价格", "价", "price", "Price", "input", "Input", "output", "Output", "prompt", "Prompt", "completion", "Completion", "输入", "输出", "缓存读", "缓存写", "cache read", "cache write"}
	for _, replacement := range replacements {
		model = strings.ReplaceAll(model, replacement, " ")
	}
	model = strings.Trim(model, " \t:-|·,，：")
	model = strings.Join(strings.Fields(model), " ")
	if len(model) > 160 {
		return strings.TrimSpace(model[:160])
	}
	return model
}

func looksLikeModelHeading(line string) bool {
	if len(line) > 120 {
		return false
	}
	lower := strings.ToLower(line)
	return strings.Contains(lower, "gpt") || strings.Contains(lower, "claude") || strings.Contains(lower, "gemini") || strings.Contains(lower, "llama") || strings.Contains(lower, "deepseek") || strings.Contains(lower, "qwen")
}

type marketTextEventSignal struct {
	EventType      string
	Severity       string
	Model          string
	Line           string
	Title          string
	Description    string
	Recommendation string
}

func (s *Service) recordMarketTextEvents(ctx context.Context, in MarketPriceParseInput) {
	if s == nil || s.repo == nil {
		return
	}
	observedAt := s.now().UTC()
	if in.ObservedAt != nil {
		observedAt = in.ObservedAt.UTC()
	}
	seen := map[string]struct{}{}
	for _, signal := range marketTextEventSignals(in.Text) {
		key := signal.EventType + "\x00" + signal.Model + "\x00" + signal.Line
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		_, _ = s.repo.CreateKanbanEvent(ctx, &adminplusdomain.KanbanEvent{
			EventType:           signal.EventType,
			Severity:            signal.Severity,
			Status:              "open",
			Model:               signal.Model,
			SourceType:          normalizeSourceType(in.SourceType),
			SourceID:            firstPositiveID(in.SupplierID, in.SiteID),
			RelatedSnapshotType: "overview",
			Title:               signal.Title,
			Description:         signal.Description,
			Recommendation:      signal.Recommendation,
			OccurredAt:          observedAt,
			Payload: map[string]any{
				"line":        signal.Line,
				"source_name": strings.TrimSpace(in.SourceName),
				"source_url":  strings.TrimSpace(in.SourceURL),
				"parser":      "market_price_text_v1",
			},
		})
	}
}

func marketTextEventSignals(text string) []marketTextEventSignal {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n"), "\n")
	signals := make([]marketTextEventSignal, 0)
	currentModel := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if model := modelFromMarketTextLine(line); model != "" {
			currentModel = model
		} else if looksLikeModelHeading(line) {
			currentModel = cleanParsedModel(line)
		}
		eventType, severity := marketTextEventType(line)
		if eventType == "" {
			continue
		}
		model := firstNonEmpty(modelFromMarketTextLine(line), currentModel)
		signals = append(signals, marketTextEventSignal{
			EventType:      eventType,
			Severity:       severity,
			Model:          model,
			Line:           line,
			Title:          marketTextEventTitle(eventType),
			Description:    marketTextEventDescription(eventType, line),
			Recommendation: marketTextEventRecommendation(eventType),
		})
	}
	return signals
}

func modelFromMarketTextLine(line string) string {
	model := strings.TrimSpace(marketModelTokenPattern.FindString(line))
	if model == "" {
		return ""
	}
	if len(model) > 160 {
		return strings.TrimSpace(model[:160])
	}
	return model
}

func marketTextEventType(line string) (string, string) {
	lower := strings.ToLower(line)
	switch {
	case containsAny(lower, "下架", "停售", "停止提供", "retired", "deprecated", "removed", "discontinued"):
		return "market_model_removed", "warning"
	case containsAny(lower, "新增模型", "新模型", "新增", "上线", "new model", "launched", "newly available"):
		return "market_model_added", "info"
	case containsAny(lower, "限时", "活动", "促销", "折扣", "优惠", "赠送", "bonus", "promo", "promotion", "discount", "limited time"):
		return "market_promotion", "info"
	default:
		return "", ""
	}
}

func marketTextEventTitle(eventType string) string {
	switch eventType {
	case "market_model_added":
		return "市场新增模型"
	case "market_model_removed":
		return "市场模型下架"
	case "market_promotion":
		return "市场限时活动"
	default:
		return "市场价格事件"
	}
}

func marketTextEventDescription(eventType string, line string) string {
	switch eventType {
	case "market_model_added":
		return "公开价格文本出现新增或上线模型信号：" + line
	case "market_model_removed":
		return "公开价格文本出现下架或停售模型信号：" + line
	case "market_promotion":
		return "公开价格文本出现促销、赠送或限时活动信号：" + line
	default:
		return line
	}
}

func marketTextEventRecommendation(eventType string) string {
	switch eventType {
	case "market_model_added":
		return "复核模型纯度、供应成本和缓存效率后，再决定是否纳入报价或供应验收。"
	case "market_model_removed":
		return "检查自身是否仍在售该模型，避免继续主推已被主要同行下架的模型。"
	case "market_promotion":
		return "区分长期售价和限时活动，不要把促销价直接作为可持续跟价基准。"
	default:
		return "复核来源和口径后再进入定价决策。"
	}
}
