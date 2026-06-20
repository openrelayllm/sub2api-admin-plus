package adminplus

import (
	"net/http"
	"strconv"

	suppliersapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/suppliers"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type SupplierHandler struct {
	service *suppliersapp.Service
}

func NewSupplierHandler(service *suppliersapp.Service) *SupplierHandler {
	return &SupplierHandler{service: service}
}

type createSupplierRequest struct {
	Name                string `json:"name" binding:"required"`
	Kind                string `json:"kind" binding:"required"`
	Type                string `json:"type" binding:"required"`
	RuntimeStatus       string `json:"runtime_status"`
	HealthStatus        string `json:"health_status"`
	DashboardURL        string `json:"dashboard_url"`
	APIBaseURL          string `json:"api_base_url"`
	Contact             string `json:"contact"`
	Notes               string `json:"notes"`
	AdminAPIKey         string `json:"admin_api_key"`
	PostgresReadDSN     string `json:"postgres_read_dsn"`
	RedisReadDSN        string `json:"redis_read_dsn"`
	BrowserLoginEnabled bool   `json:"browser_login_enabled"`
	BalanceCents        int64  `json:"balance_cents"`
	BalanceCurrency     string `json:"balance_currency"`
}

type updateSupplierStatusRequest struct {
	RuntimeStatus string `json:"runtime_status" binding:"required"`
	HealthStatus  string `json:"health_status" binding:"required"`
}

func (h *SupplierHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), suppliersapp.SupplierFilter{
		Kind:          adminplusdomain.NormalizeSupplierKind(c.Query("kind")),
		Type:          adminplusdomain.NormalizeSupplierType(c.Query("type")),
		RuntimeStatus: adminplusdomain.NormalizeSupplierRuntimeStatus(c.Query("runtime_status")),
		HealthStatus:  adminplusdomain.NormalizeSupplierHealthStatus(c.Query("health_status")),
		Query:         c.Query("q"),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{"items": items, "total": len(items)})
}

func (h *SupplierHandler) Create(c *gin.Context) {
	var req createSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	supplier, err := h.service.Create(c.Request.Context(), suppliersapp.CreateSupplierInput{
		Name:                req.Name,
		Kind:                adminplusdomain.NormalizeSupplierKind(req.Kind),
		Type:                adminplusdomain.NormalizeSupplierType(req.Type),
		RuntimeStatus:       adminplusdomain.NormalizeSupplierRuntimeStatus(req.RuntimeStatus),
		HealthStatus:        adminplusdomain.NormalizeSupplierHealthStatus(req.HealthStatus),
		DashboardURL:        req.DashboardURL,
		APIBaseURL:          req.APIBaseURL,
		Contact:             req.Contact,
		Notes:               req.Notes,
		AdminAPIKey:         req.AdminAPIKey,
		PostgresReadDSN:     req.PostgresReadDSN,
		RedisReadDSN:        req.RedisReadDSN,
		BrowserLoginEnabled: req.BrowserLoginEnabled,
		BalanceCents:        req.BalanceCents,
		BalanceCurrency:     req.BalanceCurrency,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, supplier)
}

func (h *SupplierHandler) Get(c *gin.Context) {
	id, ok := parseSupplierID(c)
	if !ok {
		return
	}
	supplier, err := h.service.Get(c.Request.Context(), id)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, supplier)
}

func (h *SupplierHandler) UpdateStatus(c *gin.Context) {
	id, ok := parseSupplierID(c)
	if !ok {
		return
	}

	var req updateSupplierStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	supplier, err := h.service.UpdateStatus(c.Request.Context(), id, suppliersapp.UpdateSupplierStatusInput{
		RuntimeStatus: adminplusdomain.NormalizeSupplierRuntimeStatus(req.RuntimeStatus),
		HealthStatus:  adminplusdomain.NormalizeSupplierHealthStatus(req.HealthStatus),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, supplier)
}

func parseSupplierID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid supplier id")
		return 0, false
	}
	return id, true
}
