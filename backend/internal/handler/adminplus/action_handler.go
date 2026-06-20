package adminplus

import (
	actionsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/actions"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type ActionHandler struct {
	service *actionsapp.Service
}

func NewActionHandler(service *actionsapp.Service) *ActionHandler {
	return &ActionHandler{service: service}
}

type generateActionsRequest struct {
	Suppliers       []supplierSignalDTO                   `json:"suppliers" binding:"required"`
	BalanceEvents   []*adminplusdomain.BalanceEvent       `json:"balance_events"`
	PromotionEvents []*adminplusdomain.PromotionEvent     `json:"promotion_events"`
	HealthEvents    []*adminplusdomain.HealthEvent        `json:"health_events"`
	Reconciliation  adminplusdomain.ReconciliationSummary `json:"reconciliation"`
	MinProfitMargin float64                               `json:"min_profit_margin"`
}

type supplierSignalDTO struct {
	SupplierID         int64  `json:"supplier_id" binding:"required"`
	Name               string `json:"name"`
	RuntimeStatus      string `json:"runtime_status"`
	HealthStatus       string `json:"health_status"`
	BalanceCents       int64  `json:"balance_cents"`
	Currency           string `json:"currency"`
	EffectiveCostCents int64  `json:"effective_cost_cents"`
}

func (h *ActionHandler) Generate(c *gin.Context) {
	var req generateActionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	suppliers := make([]actionsapp.SupplierSignal, 0, len(req.Suppliers))
	for _, supplier := range req.Suppliers {
		suppliers = append(suppliers, actionsapp.SupplierSignal{
			SupplierID:         supplier.SupplierID,
			Name:               supplier.Name,
			RuntimeStatus:      adminplusdomain.NormalizeSupplierRuntimeStatus(supplier.RuntimeStatus),
			HealthStatus:       adminplusdomain.NormalizeSupplierHealthStatus(supplier.HealthStatus),
			BalanceCents:       supplier.BalanceCents,
			Currency:           supplier.Currency,
			EffectiveCostCents: supplier.EffectiveCostCents,
		})
	}
	result, err := h.service.Generate(c.Request.Context(), actionsapp.GenerateInput{
		Suppliers:       suppliers,
		BalanceEvents:   req.BalanceEvents,
		PromotionEvents: req.PromotionEvents,
		HealthEvents:    req.HealthEvents,
		Reconciliation:  req.Reconciliation,
		MinProfitMargin: req.MinProfitMargin,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}
