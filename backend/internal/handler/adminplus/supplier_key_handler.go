package adminplus

import (
	"context"
	"net/http"

	supplierkeysapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/supplierkeys"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type SupplierKeyHandler struct {
	service *supplierkeysapp.Service
}

func NewSupplierKeyHandler(service *supplierkeysapp.Service) *SupplierKeyHandler {
	return &SupplierKeyHandler{service: service}
}

type provisionSupplierKeyRequest struct {
	SupplierGroupID            int64    `json:"supplier_group_id" binding:"required"`
	Name                       string   `json:"name"`
	QuotaUSD                   float64  `json:"quota_usd"`
	ExpiresInDays              *int     `json:"expires_in_days"`
	LocalAccountPlatform       string   `json:"local_account_platform"`
	LocalAccountName           string   `json:"local_account_name"`
	LocalAccountBaseURL        string   `json:"local_account_base_url"`
	LocalAccountConcurrency    int      `json:"local_account_concurrency"`
	LocalAccountPriority       int      `json:"local_account_priority"`
	LocalAccountRateMultiplier *float64 `json:"local_account_rate_multiplier"`
	LocalAccountGroupIDs       []int64  `json:"local_account_group_ids"`
	RuntimeStatus              string   `json:"runtime_status"`
	HealthStatus               string   `json:"health_status"`
	BalanceThresholdCents      int64    `json:"balance_threshold_cents"`
	BalanceCents               int64    `json:"balance_cents"`
	BalanceCurrency            string   `json:"balance_currency"`
}

func (h *SupplierKeyHandler) Provision(c *gin.Context) {
	supplierID, ok := parseSupplierID(c)
	if !ok {
		return
	}
	var req provisionSupplierKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	input := supplierkeysapp.ProvisionKeyInput{
		SupplierID:                 supplierID,
		SupplierGroupID:            req.SupplierGroupID,
		Name:                       req.Name,
		QuotaUSD:                   req.QuotaUSD,
		ExpiresInDays:              req.ExpiresInDays,
		LocalAccountPlatform:       req.LocalAccountPlatform,
		LocalAccountName:           req.LocalAccountName,
		LocalAccountBaseURL:        req.LocalAccountBaseURL,
		LocalAccountConcurrency:    req.LocalAccountConcurrency,
		LocalAccountPriority:       req.LocalAccountPriority,
		LocalAccountRateMultiplier: req.LocalAccountRateMultiplier,
		LocalAccountGroupIDs:       req.LocalAccountGroupIDs,
		RuntimeStatus:              adminplusdomain.NormalizeSupplierRuntimeStatus(req.RuntimeStatus),
		HealthStatus:               adminplusdomain.NormalizeSupplierHealthStatus(req.HealthStatus),
		BalanceThresholdCents:      req.BalanceThresholdCents,
		BalanceCents:               req.BalanceCents,
		BalanceCurrency:            req.BalanceCurrency,
	}
	executeAdminPlusIdempotentJSON(c, "admin-plus.supplier-key.provision", input, 0, http.StatusCreated, func(ctx context.Context) (any, error) {
		return h.service.Provision(ctx, input)
	})
}

func (h *SupplierKeyHandler) List(c *gin.Context) {
	supplierID, ok := parseSupplierID(c)
	if !ok {
		return
	}
	page := parsePagination(c)
	items, err := h.service.List(c.Request.Context(), supplierkeysapp.ListFilter{
		SupplierID: supplierID,
		Status:     adminplusdomain.NormalizeSupplierKeyStatus(c.Query("status")),
		Query:      c.Query("q"),
		Limit:      fetchLimitForPagination(page),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	paged, total := paginateSlice(items, page)
	response.Success(c, paginatedData(paged, total, page))
}
