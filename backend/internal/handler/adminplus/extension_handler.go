package adminplus

import (
	"net/http"
	"strconv"
	"time"

	extensionapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/extension"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type ExtensionHandler struct {
	service *extensionapp.Service
}

func NewExtensionHandler(service *extensionapp.Service) *ExtensionHandler {
	return &ExtensionHandler{service: service}
}

type createExtensionTaskRequest struct {
	SupplierID     int64          `json:"supplier_id" binding:"required"`
	Type           string         `json:"type" binding:"required"`
	Priority       int            `json:"priority"`
	MaxAttempts    int            `json:"max_attempts"`
	AvailableAfter string         `json:"available_after"`
	Payload        map[string]any `json:"payload"`
}

type claimExtensionTaskRequest struct {
	DeviceID        string   `json:"device_id" binding:"required"`
	Types           []string `json:"types"`
	LeaseTTLSeconds int64    `json:"lease_ttl_seconds"`
}

type extensionTaskLeaseRequest struct {
	DeviceID        string         `json:"device_id" binding:"required"`
	LeaseToken      string         `json:"lease_token" binding:"required"`
	LeaseTTLSeconds int64          `json:"lease_ttl_seconds"`
	Result          map[string]any `json:"result"`
	ErrorCode       string         `json:"error_code"`
	ErrorMessage    string         `json:"error_message"`
	RetryAfter      string         `json:"retry_after"`
}

func (h *ExtensionHandler) CreateTask(c *gin.Context) {
	var req createExtensionTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	availableAfter, ok := parseOptionalNamedTime(c, "available_after", req.AvailableAfter)
	if !ok {
		return
	}
	task, err := h.service.CreateTask(c.Request.Context(), extensionapp.CreateTaskInput{
		SupplierID:     req.SupplierID,
		Type:           adminplusdomain.ExtensionTaskType(req.Type),
		Priority:       req.Priority,
		MaxAttempts:    req.MaxAttempts,
		AvailableAfter: availableAfter,
		Payload:        req.Payload,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, task)
}

func (h *ExtensionHandler) ClaimTask(c *gin.Context) {
	var req claimExtensionTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	taskTypes := make([]adminplusdomain.ExtensionTaskType, 0, len(req.Types))
	for _, taskType := range req.Types {
		taskTypes = append(taskTypes, adminplusdomain.ExtensionTaskType(taskType))
	}
	task, err := h.service.ClaimTask(c.Request.Context(), extensionapp.ClaimTaskInput{
		DeviceID: req.DeviceID,
		Types:    taskTypes,
		LeaseTTL: secondsDuration(req.LeaseTTLSeconds),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, task)
}

func (h *ExtensionHandler) Heartbeat(c *gin.Context) {
	id, ok := parseExtensionTaskID(c)
	if !ok {
		return
	}
	var req extensionTaskLeaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	task, err := h.service.Heartbeat(c.Request.Context(), extensionapp.HeartbeatInput{
		TaskID:     id,
		DeviceID:   req.DeviceID,
		LeaseToken: req.LeaseToken,
		LeaseTTL:   secondsDuration(req.LeaseTTLSeconds),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, task)
}

func (h *ExtensionHandler) CompleteTask(c *gin.Context) {
	id, ok := parseExtensionTaskID(c)
	if !ok {
		return
	}
	var req extensionTaskLeaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	task, err := h.service.CompleteTask(c.Request.Context(), extensionapp.CompleteTaskInput{
		TaskID:     id,
		DeviceID:   req.DeviceID,
		LeaseToken: req.LeaseToken,
		Result:     req.Result,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, task)
}

func (h *ExtensionHandler) FailTask(c *gin.Context) {
	id, ok := parseExtensionTaskID(c)
	if !ok {
		return
	}
	var req extensionTaskLeaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	retryAfter, ok := parseOptionalNamedTime(c, "retry_after", req.RetryAfter)
	if !ok {
		return
	}
	task, err := h.service.FailTask(c.Request.Context(), extensionapp.FailTaskInput{
		TaskID:       id,
		DeviceID:     req.DeviceID,
		LeaseToken:   req.LeaseToken,
		ErrorCode:    req.ErrorCode,
		ErrorMessage: req.ErrorMessage,
		RetryAfter:   retryAfter,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, task)
}

func (h *ExtensionHandler) ListTasks(c *gin.Context) {
	items, err := h.service.ListTasks(c.Request.Context(), extensionapp.TaskFilter{
		SupplierID: parseInt64Query(c, "supplier_id"),
		Status:     adminplusdomain.ExtensionTaskStatus(c.Query("status")),
		Type:       adminplusdomain.ExtensionTaskType(c.Query("type")),
		Limit:      parseIntQuery(c, "limit"),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{"items": items, "total": len(items)})
}

func parseExtensionTaskID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid extension task id")
		return 0, false
	}
	return id, true
}

func secondsDuration(seconds int64) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
