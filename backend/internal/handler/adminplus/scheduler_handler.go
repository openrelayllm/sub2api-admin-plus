package adminplus

import (
	"strconv"
	"strings"

	schedulerapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/scheduler"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type SchedulerHandler struct {
	service *schedulerapp.Service
}

func NewSchedulerHandler(service *schedulerapp.Service) *SchedulerHandler {
	return &SchedulerHandler{service: service}
}

type runSchedulerRequest struct {
	Mode          string   `json:"mode"`
	SupplierID    int64    `json:"supplier_id"`
	TaskTypes     []string `json:"task_types"`
	WindowMinutes int      `json:"window_minutes"`
	DryRun        bool     `json:"dry_run"`
}

type schedulerStatusRequest struct {
	Status string `json:"status"`
}

func (h *SchedulerHandler) Run(c *gin.Context) {
	var req runSchedulerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	taskTypes := make([]adminplusdomain.ExtensionTaskType, 0, len(req.TaskTypes))
	for _, raw := range req.TaskTypes {
		taskTypes = append(taskTypes, adminplusdomain.ExtensionTaskType(strings.TrimSpace(raw)))
	}
	input := schedulerapp.RunInput{
		Mode:          req.Mode,
		SupplierID:    req.SupplierID,
		TaskTypes:     taskTypes,
		WindowMinutes: req.WindowMinutes,
		DryRun:        req.DryRun,
	}
	if req.DryRun {
		run, err := h.service.Run(c.Request.Context(), input)
		if response.ErrorFrom(c, err) {
			return
		}
		response.Success(c, run)
		return
	}
	summary, err := h.service.EnqueueRun(c.Request.Context(), input)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Accepted(c, summary)
}

func (h *SchedulerHandler) Status(c *gin.Context) {
	response.Success(c, h.service.Status())
}

func (h *SchedulerHandler) CenterStatus(c *gin.Context) {
	response.Success(c, h.service.CenterStatus(c.Request.Context()))
}

func (h *SchedulerHandler) ListPlans(c *gin.Context) {
	response.Success(c, h.service.ListPlans(c.Request.Context()))
}

func (h *SchedulerHandler) UpdatePlanStatus(c *gin.Context) {
	var req schedulerStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	plan, err := h.service.UpdatePlanStatus(c.Request.Context(), c.Param("id"), req.Status)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, plan)
}

func (h *SchedulerHandler) UpdatePlanConfig(c *gin.Context) {
	var req adminplusdomain.SchedulerPlanConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	plan, err := h.service.UpdatePlanConfig(c.Request.Context(), c.Param("id"), req)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, plan)
}

func (h *SchedulerHandler) CreateRun(c *gin.Context) {
	var req runSchedulerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	taskTypes := make([]adminplusdomain.ExtensionTaskType, 0, len(req.TaskTypes))
	for _, raw := range req.TaskTypes {
		taskTypes = append(taskTypes, adminplusdomain.ExtensionTaskType(strings.TrimSpace(raw)))
	}
	summary, err := h.service.EnqueueRun(c.Request.Context(), schedulerapp.RunInput{
		Mode:          schedulerFirstNonEmpty(req.Mode, "manual"),
		SupplierID:    req.SupplierID,
		TaskTypes:     taskTypes,
		WindowMinutes: req.WindowMinutes,
		DryRun:        req.DryRun,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Accepted(c, summary)
}

func (h *SchedulerHandler) ListRuns(c *gin.Context) {
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			offset = parsed
		}
	}
	response.Success(c, h.service.ListRuns(c.Request.Context(), limit, offset, c.Query("task_type")))
}

func (h *SchedulerHandler) GetRun(c *gin.Context) {
	detail, err := h.service.GetRunDetail(c.Request.Context(), c.Param("id"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, detail)
}

func (h *SchedulerHandler) DeleteRun(c *gin.Context) {
	result, err := h.service.DeleteRun(c.Request.Context(), c.Param("id"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *SchedulerHandler) DeleteRuns(c *gin.Context) {
	result, err := h.service.DeleteRuns(c.Request.Context(), c.Query("task_type"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *SchedulerHandler) CancelRun(c *gin.Context) {
	run, err := h.service.CancelRun(c.Request.Context(), c.Param("id"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Accepted(c, run)
}

func (h *SchedulerHandler) RetryRunFailedSteps(c *gin.Context) {
	detail, err := h.service.RetryFailedSteps(c.Request.Context(), c.Param("id"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Accepted(c, detail)
}

func (h *SchedulerHandler) ListSteps(c *gin.Context) {
	limit := 200
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			offset = parsed
		}
	}
	steps, err := h.service.ListSteps(c.Request.Context(), c.Query("run_id"), limit, offset)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, steps)
}

func (h *SchedulerHandler) RetryStep(c *gin.Context) {
	stepID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || stepID <= 0 {
		response.BadRequest(c, "invalid scheduler step id")
		return
	}
	step, err := h.service.RetryStep(c.Request.Context(), stepID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Accepted(c, step)
}

func (h *SchedulerHandler) CancelStep(c *gin.Context) {
	stepID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || stepID <= 0 {
		response.BadRequest(c, "invalid scheduler step id")
		return
	}
	step, err := h.service.CancelStep(c.Request.Context(), stepID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Accepted(c, step)
}

func (h *SchedulerHandler) ListSupplierStatuses(c *gin.Context) {
	items, err := h.service.ListSupplierStatuses(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, items)
}

func (h *SchedulerHandler) GetSupplierChecklist(c *gin.Context) {
	supplierID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || supplierID <= 0 {
		response.BadRequest(c, "invalid scheduler supplier id")
		return
	}
	checklist, err := h.service.GetSupplierChecklist(c.Request.Context(), supplierID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, checklist)
}

func (h *SchedulerHandler) ListActions(c *gin.Context) {
	response.Success(c, h.service.ListActions(c.Request.Context()))
}

func (h *SchedulerHandler) UpdateActionStatus(c *gin.Context) {
	var req schedulerStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	action, err := h.service.ResolveAction(c.Request.Context(), c.Param("id"), req.Status)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, action)
}

func (h *SchedulerHandler) Settings(c *gin.Context) {
	response.Success(c, h.service.Settings(c.Request.Context()))
}

func (h *SchedulerHandler) UpdateSettings(c *gin.Context) {
	var req adminplusdomain.SchedulerSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	settings, err := h.service.UpdateSettings(c.Request.Context(), req)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, settings)
}

func schedulerFirstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
