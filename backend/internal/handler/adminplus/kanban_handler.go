package adminplus

import (
	"net/http"
	"strconv"

	kanbanapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/kanban"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type KanbanHandler struct {
	service *kanbanapp.Service
}

func NewKanbanHandler(service *kanbanapp.Service) *KanbanHandler {
	return &KanbanHandler{service: service}
}

type recordMarketPriceRequest struct {
	SourceType        string         `json:"source_type"`
	SourceName        string         `json:"source_name"`
	SourceURL         string         `json:"source_url"`
	SiteID            int64          `json:"site_id"`
	SupplierID        int64          `json:"supplier_id"`
	Model             string         `json:"model" binding:"required"`
	BillingMode       string         `json:"billing_mode"`
	PriceItem         string         `json:"price_item"`
	Unit              string         `json:"unit"`
	Currency          string         `json:"currency"`
	PriceMicros       int64          `json:"price_micros"`
	PackageLabel      string         `json:"package_label"`
	PackagePriceCents *int64         `json:"package_price_cents"`
	PackageQuota      string         `json:"package_quota"`
	RateMultiplier    *float64       `json:"rate_multiplier"`
	MinRechargeCents  *int64         `json:"min_recharge_cents"`
	BonusPercent      *float64       `json:"bonus_percent"`
	Confidence        float64        `json:"confidence"`
	ObservedAt        string         `json:"observed_at"`
	RawPayload        map[string]any `json:"raw_payload"`
}

type parseMarketPriceRequest struct {
	SourceType      string  `json:"source_type"`
	SourceName      string  `json:"source_name"`
	SourceURL       string  `json:"source_url"`
	SiteID          int64   `json:"site_id"`
	SupplierID      int64   `json:"supplier_id"`
	DefaultCurrency string  `json:"default_currency"`
	Confidence      float64 `json:"confidence"`
	Text            string  `json:"text" binding:"required"`
	ObservedAt      string  `json:"observed_at"`
}

type importMarketPriceURLRequest struct {
	SourceType      string  `json:"source_type"`
	SourceName      string  `json:"source_name"`
	SourceURL       string  `json:"source_url" binding:"required"`
	SiteID          int64   `json:"site_id"`
	SupplierID      int64   `json:"supplier_id"`
	DefaultCurrency string  `json:"default_currency"`
	Confidence      float64 `json:"confidence"`
	ObservedAt      string  `json:"observed_at"`
}

type recordCacheEfficiencyRequest struct {
	SupplyType            string         `json:"supply_type"`
	SupplierID            int64          `json:"supplier_id"`
	LocalSub2APIAccountID int64          `json:"local_sub2api_account_id"`
	Model                 string         `json:"model" binding:"required"`
	RoutingStrategy       string         `json:"routing_strategy"`
	StickyScope           string         `json:"sticky_scope"`
	SampleRequests        int            `json:"sample_requests"`
	CacheReadTokens       int64          `json:"cache_read_tokens"`
	CacheWriteTokens      int64          `json:"cache_write_tokens"`
	InputTokens           int64          `json:"input_tokens"`
	OutputTokens          int64          `json:"output_tokens"`
	CacheHitRatio         *float64       `json:"cache_hit_ratio"`
	DuplicateInputTokens  int64          `json:"duplicate_input_tokens"`
	EstimatedWasteCents   int64          `json:"estimated_waste_cents"`
	AvgTTFTMS             *int64         `json:"avg_ttft_ms"`
	AvgTotalLatencyMS     *int64         `json:"avg_total_latency_ms"`
	Status                string         `json:"status"`
	Notes                 string         `json:"notes"`
	ObservedAt            string         `json:"observed_at"`
	RawPayload            map[string]any `json:"raw_payload"`
}

type recordSupplyQualityRequest struct {
	SupplyType            string         `json:"supply_type"`
	SupplierID            int64          `json:"supplier_id"`
	LocalSub2APIAccountID int64          `json:"local_sub2api_account_id"`
	Model                 string         `json:"model"`
	AvailabilityRatio     float64        `json:"availability_ratio"`
	ErrorRatio            float64        `json:"error_ratio"`
	AvgTTFTMS             *int64         `json:"avg_ttft_ms"`
	AvgTotalLatencyMS     *int64         `json:"avg_total_latency_ms"`
	CacheHitRatio         float64        `json:"cache_hit_ratio"`
	PurityScore           float64        `json:"purity_score"`
	UsageTrustScore       float64        `json:"usage_trust_score"`
	BalanceRiskScore      float64        `json:"balance_risk_score"`
	ConcurrencyScore      float64        `json:"concurrency_score"`
	QualityScore          float64        `json:"quality_score"`
	Decision              string         `json:"decision"`
	Notes                 string         `json:"notes"`
	ObservedAt            string         `json:"observed_at"`
	RawPayload            map[string]any `json:"raw_payload"`
}

type recordAcceptanceReportRequest struct {
	SupplyType            string         `json:"supply_type"`
	SupplierID            int64          `json:"supplier_id"`
	LocalSub2APIAccountID int64          `json:"local_sub2api_account_id"`
	Model                 string         `json:"model"`
	Status                string         `json:"status"`
	ConnectivityStatus    string         `json:"connectivity_status"`
	ModelListStatus       string         `json:"model_list_status"`
	PurityStatus          string         `json:"purity_status"`
	TrialCallStatus       string         `json:"trial_call_status"`
	UsageMeteringStatus   string         `json:"usage_metering_status"`
	CacheAuditStatus      string         `json:"cache_audit_status"`
	BalanceStatus         string         `json:"balance_status"`
	ConcurrencyStatus     string         `json:"concurrency_status"`
	FailureReason         string         `json:"failure_reason"`
	Recommendation        string         `json:"recommendation"`
	ReportPayload         map[string]any `json:"report_payload"`
	ObservedAt            string         `json:"observed_at"`
}

type generateAcceptanceReportRequest struct {
	SupplyType            string `json:"supply_type"`
	SupplierID            int64  `json:"supplier_id"`
	LocalSub2APIAccountID int64  `json:"local_sub2api_account_id"`
	Model                 string `json:"model"`
	EnqueueEvidenceTasks  bool   `json:"enqueue_evidence_tasks"`
	ObservedAt            string `json:"observed_at"`
}

type refreshAcceptanceEvidenceRunRequest struct {
	RunID string `json:"run_id" binding:"required"`
}

type updateKanbanEventStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *KanbanHandler) Overview(c *gin.Context) {
	targetMarginPercent, ok := parseFloat64Query(c, "target_margin_percent")
	if !ok {
		return
	}
	riskBufferPercent, ok := parseFloat64Query(c, "risk_buffer_percent")
	if !ok {
		return
	}
	overview, err := h.service.Overview(c.Request.Context(), kanbanapp.OverviewFilter{
		Model:               c.Query("model"),
		TargetMarginPercent: targetMarginPercent,
		RiskBufferPercent:   riskBufferPercent,
		Limit:               parsePositiveIntQuery(c, "limit", 500),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, overview)
}

func (h *KanbanHandler) RecordMarketPrice(c *gin.Context) {
	var req recordMarketPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	observedAt, ok := parseOptionalTime(c, req.ObservedAt)
	if !ok {
		return
	}
	item, err := h.service.RecordMarketPrice(c.Request.Context(), kanbanapp.MarketPriceInput{
		SourceType:        req.SourceType,
		SourceName:        req.SourceName,
		SourceURL:         req.SourceURL,
		SiteID:            req.SiteID,
		SupplierID:        req.SupplierID,
		Model:             req.Model,
		BillingMode:       req.BillingMode,
		PriceItem:         req.PriceItem,
		Unit:              req.Unit,
		Currency:          req.Currency,
		PriceMicros:       req.PriceMicros,
		PackageLabel:      req.PackageLabel,
		PackagePriceCents: req.PackagePriceCents,
		PackageQuota:      req.PackageQuota,
		RateMultiplier:    req.RateMultiplier,
		MinRechargeCents:  req.MinRechargeCents,
		BonusPercent:      req.BonusPercent,
		Confidence:        req.Confidence,
		ObservedAt:        observedAt,
		RawPayload:        req.RawPayload,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, item)
}

func (h *KanbanHandler) ParseMarketPrices(c *gin.Context) {
	var req parseMarketPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	observedAt, ok := parseOptionalTime(c, req.ObservedAt)
	if !ok {
		return
	}
	result, err := h.service.ParseMarketPrices(c.Request.Context(), kanbanapp.MarketPriceParseInput{
		SourceType:      req.SourceType,
		SourceName:      req.SourceName,
		SourceURL:       req.SourceURL,
		SiteID:          req.SiteID,
		SupplierID:      req.SupplierID,
		DefaultCurrency: req.DefaultCurrency,
		Confidence:      req.Confidence,
		Text:            req.Text,
		ObservedAt:      observedAt,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, result)
}

func (h *KanbanHandler) ImportMarketPricesFromURL(c *gin.Context) {
	var req importMarketPriceURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	observedAt, ok := parseOptionalTime(c, req.ObservedAt)
	if !ok {
		return
	}
	result, err := h.service.ImportMarketPricesFromURL(c.Request.Context(), kanbanapp.MarketPriceImportURLInput{
		SourceType:      req.SourceType,
		SourceName:      req.SourceName,
		SourceURL:       req.SourceURL,
		SiteID:          req.SiteID,
		SupplierID:      req.SupplierID,
		DefaultCurrency: req.DefaultCurrency,
		Confidence:      req.Confidence,
		ObservedAt:      observedAt,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, result)
}

func (h *KanbanHandler) DiscoverMarketPriceSources(c *gin.Context) {
	result, err := h.service.DiscoverMarketPriceSources(c.Request.Context(), kanbanapp.PriceSourceDiscoveryInput{
		Query:                c.Query("q"),
		Limit:                parsePositiveIntQuery(c, "limit", 50),
		IncludeLowConfidence: c.Query("include_low_confidence") == "true",
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *KanbanHandler) ListMarketPrices(c *gin.Context) {
	page := parsePagination(c)
	items, err := h.service.ListMarketPrices(c.Request.Context(), kanbanapp.MarketPriceFilter{
		Model:      c.Query("model"),
		SourceType: c.Query("source_type"),
		SiteID:     parseInt64Query(c, "site_id"),
		SupplierID: parseInt64Query(c, "supplier_id"),
		Limit:      fetchLimitForPagination(page),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	paged, total := paginateSlice(items, page)
	response.Success(c, paginatedData(paged, total, page))
}

func (h *KanbanHandler) RecordCacheEfficiency(c *gin.Context) {
	var req recordCacheEfficiencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	observedAt, ok := parseOptionalTime(c, req.ObservedAt)
	if !ok {
		return
	}
	item, err := h.service.RecordCacheEfficiency(c.Request.Context(), kanbanapp.CacheEfficiencyInput{
		SupplyType:            req.SupplyType,
		SupplierID:            req.SupplierID,
		LocalSub2APIAccountID: req.LocalSub2APIAccountID,
		Model:                 req.Model,
		RoutingStrategy:       req.RoutingStrategy,
		StickyScope:           req.StickyScope,
		SampleRequests:        req.SampleRequests,
		CacheReadTokens:       req.CacheReadTokens,
		CacheWriteTokens:      req.CacheWriteTokens,
		InputTokens:           req.InputTokens,
		OutputTokens:          req.OutputTokens,
		CacheHitRatio:         req.CacheHitRatio,
		DuplicateInputTokens:  req.DuplicateInputTokens,
		EstimatedWasteCents:   req.EstimatedWasteCents,
		AvgTTFTMS:             req.AvgTTFTMS,
		AvgTotalLatencyMS:     req.AvgTotalLatencyMS,
		Status:                req.Status,
		Notes:                 req.Notes,
		ObservedAt:            observedAt,
		RawPayload:            req.RawPayload,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, item)
}

func (h *KanbanHandler) ListCacheEfficiency(c *gin.Context) {
	page := parsePagination(c)
	items, err := h.service.ListCacheEfficiency(c.Request.Context(), kanbanapp.CacheEfficiencyFilter{
		Model:                 c.Query("model"),
		SupplyType:            c.Query("supply_type"),
		SupplierID:            parseInt64Query(c, "supplier_id"),
		LocalSub2APIAccountID: parseInt64Query(c, "local_sub2api_account_id"),
		Status:                c.Query("status"),
		Limit:                 fetchLimitForPagination(page),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	paged, total := paginateSlice(items, page)
	response.Success(c, paginatedData(paged, total, page))
}

func (h *KanbanHandler) RecordSupplyQuality(c *gin.Context) {
	var req recordSupplyQualityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	observedAt, ok := parseOptionalTime(c, req.ObservedAt)
	if !ok {
		return
	}
	item, err := h.service.RecordSupplyQuality(c.Request.Context(), kanbanapp.SupplyQualityInput{
		SupplyType:            req.SupplyType,
		SupplierID:            req.SupplierID,
		LocalSub2APIAccountID: req.LocalSub2APIAccountID,
		Model:                 req.Model,
		AvailabilityRatio:     req.AvailabilityRatio,
		ErrorRatio:            req.ErrorRatio,
		AvgTTFTMS:             req.AvgTTFTMS,
		AvgTotalLatencyMS:     req.AvgTotalLatencyMS,
		CacheHitRatio:         req.CacheHitRatio,
		PurityScore:           req.PurityScore,
		UsageTrustScore:       req.UsageTrustScore,
		BalanceRiskScore:      req.BalanceRiskScore,
		ConcurrencyScore:      req.ConcurrencyScore,
		QualityScore:          req.QualityScore,
		Decision:              req.Decision,
		Notes:                 req.Notes,
		ObservedAt:            observedAt,
		RawPayload:            req.RawPayload,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, item)
}

func (h *KanbanHandler) ListSupplyQuality(c *gin.Context) {
	page := parsePagination(c)
	items, err := h.service.ListSupplyQuality(c.Request.Context(), kanbanapp.SupplyQualityFilter{
		Model:                 c.Query("model"),
		SupplyType:            c.Query("supply_type"),
		SupplierID:            parseInt64Query(c, "supplier_id"),
		LocalSub2APIAccountID: parseInt64Query(c, "local_sub2api_account_id"),
		Decision:              c.Query("decision"),
		Limit:                 fetchLimitForPagination(page),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	paged, total := paginateSlice(items, page)
	response.Success(c, paginatedData(paged, total, page))
}

func (h *KanbanHandler) RecordAcceptanceReport(c *gin.Context) {
	var req recordAcceptanceReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	observedAt, ok := parseOptionalTime(c, req.ObservedAt)
	if !ok {
		return
	}
	item, err := h.service.RecordAcceptanceReport(c.Request.Context(), kanbanapp.AcceptanceReportInput{
		SupplyType:            req.SupplyType,
		SupplierID:            req.SupplierID,
		LocalSub2APIAccountID: req.LocalSub2APIAccountID,
		Model:                 req.Model,
		Status:                req.Status,
		ConnectivityStatus:    req.ConnectivityStatus,
		ModelListStatus:       req.ModelListStatus,
		PurityStatus:          req.PurityStatus,
		TrialCallStatus:       req.TrialCallStatus,
		UsageMeteringStatus:   req.UsageMeteringStatus,
		CacheAuditStatus:      req.CacheAuditStatus,
		BalanceStatus:         req.BalanceStatus,
		ConcurrencyStatus:     req.ConcurrencyStatus,
		FailureReason:         req.FailureReason,
		Recommendation:        req.Recommendation,
		ReportPayload:         req.ReportPayload,
		ObservedAt:            observedAt,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, item)
}

func (h *KanbanHandler) GenerateAcceptanceReport(c *gin.Context) {
	var req generateAcceptanceReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	observedAt, ok := parseOptionalTime(c, req.ObservedAt)
	if !ok {
		return
	}
	item, err := h.service.GenerateAcceptanceReport(c.Request.Context(), kanbanapp.AcceptanceReportGenerateInput{
		SupplyType:            req.SupplyType,
		SupplierID:            req.SupplierID,
		LocalSub2APIAccountID: req.LocalSub2APIAccountID,
		Model:                 req.Model,
		EnqueueEvidenceTasks:  req.EnqueueEvidenceTasks,
		ObservedAt:            observedAt,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, item)
}

func (h *KanbanHandler) RefreshAcceptanceReportFromEvidenceRun(c *gin.Context) {
	var req refreshAcceptanceEvidenceRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	item, err := h.service.RefreshAcceptanceReportFromEvidenceRun(c.Request.Context(), req.RunID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, item)
}

func (h *KanbanHandler) ListAcceptanceReports(c *gin.Context) {
	page := parsePagination(c)
	items, err := h.service.ListAcceptanceReports(c.Request.Context(), kanbanapp.AcceptanceReportFilter{
		Model:                 c.Query("model"),
		SupplyType:            c.Query("supply_type"),
		SupplierID:            parseInt64Query(c, "supplier_id"),
		LocalSub2APIAccountID: parseInt64Query(c, "local_sub2api_account_id"),
		Status:                c.Query("status"),
		Limit:                 fetchLimitForPagination(page),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	paged, total := paginateSlice(items, page)
	response.Success(c, paginatedData(paged, total, page))
}

func (h *KanbanHandler) ListEvents(c *gin.Context) {
	page := parsePagination(c)
	items, err := h.service.ListEvents(c.Request.Context(), kanbanapp.KanbanEventFilter{
		Model:     c.Query("model"),
		EventType: c.Query("event_type"),
		Severity:  c.Query("severity"),
		Status:    c.Query("status"),
		Limit:     fetchLimitForPagination(page),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	paged, total := paginateSlice(items, page)
	response.Success(c, paginatedData(paged, total, page))
}

func (h *KanbanHandler) UpdateEventStatus(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req updateKanbanEventStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	item, err := h.service.UpdateEventStatus(c.Request.Context(), id, req.Status)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}

func parseFloat64Query(c *gin.Context, name string) (float64, bool) {
	raw := c.Query(name)
	if raw == "" {
		return 0, true
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return value, true
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	raw := c.Param(name)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return value, true
}
