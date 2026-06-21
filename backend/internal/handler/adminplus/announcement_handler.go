package adminplus

import (
	"net/http"
	"strconv"

	announcementsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/announcements"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type AnnouncementHandler struct {
	service *announcementsapp.Service
}

func NewAnnouncementHandler(service *announcementsapp.Service) *AnnouncementHandler {
	return &AnnouncementHandler{service: service}
}

type recordAnnouncementRequest struct {
	SupplierID       int64          `json:"supplier_id" binding:"required"`
	Source           string         `json:"source"`
	Type             string         `json:"type" binding:"required"`
	Title            string         `json:"title" binding:"required"`
	Description      string         `json:"description"`
	Currency         string         `json:"currency"`
	MinRechargeCents int64          `json:"min_recharge_cents"`
	BonusPercent     *float64       `json:"bonus_percent"`
	DiscountPercent  *float64       `json:"discount_percent"`
	RuntimeStatus    string         `json:"runtime_status"`
	BalanceCents     int64          `json:"balance_cents"`
	StartsAt         string         `json:"starts_at"`
	EndsAt           string         `json:"ends_at"`
	CapturedAt       string         `json:"captured_at"`
	RawPayload       map[string]any `json:"raw_payload"`
}

func (h *AnnouncementHandler) SyncSupplierAnnouncements(c *gin.Context) {
	supplierID, ok := parseSupplierID(c)
	if !ok {
		return
	}
	result, err := h.service.SyncFromSession(c.Request.Context(), announcementsapp.SyncFromSessionInput{
		SupplierID: supplierID,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, result)
}

func (h *AnnouncementHandler) RecordAnnouncement(c *gin.Context) {
	var req recordAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	startsAt, ok := parseOptionalTime(c, req.StartsAt)
	if !ok {
		return
	}
	endsAt, ok := parseOptionalTime(c, req.EndsAt)
	if !ok {
		return
	}
	capturedAt, ok := parseOptionalTime(c, req.CapturedAt)
	if !ok {
		return
	}
	event, err := h.service.RecordAnnouncement(c.Request.Context(), announcementsapp.RecordAnnouncementInput{
		SupplierID:       req.SupplierID,
		Source:           req.Source,
		Type:             adminplusdomain.AnnouncementType(req.Type),
		Title:            req.Title,
		Description:      req.Description,
		Currency:         req.Currency,
		MinRechargeCents: req.MinRechargeCents,
		BonusPercent:     req.BonusPercent,
		DiscountPercent:  req.DiscountPercent,
		RuntimeStatus:    adminplusdomain.NormalizeSupplierRuntimeStatus(req.RuntimeStatus),
		BalanceCents:     req.BalanceCents,
		StartsAt:         startsAt,
		EndsAt:           endsAt,
		CapturedAt:       capturedAt,
		RawPayload:       req.RawPayload,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, event)
}

func (h *AnnouncementHandler) ListEvents(c *gin.Context) {
	page := parsePagination(c)
	items, err := h.service.ListEvents(c.Request.Context(), announcementsapp.EventFilter{
		SupplierID:     parseInt64Query(c, "supplier_id"),
		Status:         adminplusdomain.AnnouncementStatus(c.Query("status")),
		Recommendation: adminplusdomain.AnnouncementRecommendation(c.Query("recommendation")),
		Limit:          fetchLimitForPagination(page),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	paged, total := paginateSlice(items, page)
	response.Success(c, paginatedData(paged, total, page))
}

func (h *AnnouncementHandler) AcknowledgeEvent(c *gin.Context) {
	id, ok := parseAnnouncementEventID(c)
	if !ok {
		return
	}
	event, err := h.service.AcknowledgeEvent(c.Request.Context(), id)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, event)
}

func parseAnnouncementEventID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid announcement event id")
		return 0, false
	}
	return id, true
}
