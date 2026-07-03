package adminplus

import (
	"net/http"
	"strconv"

	accountratesyncapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/accountratesync"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type AccountRateSyncHandler struct {
	service *accountratesyncapp.Service
}

func NewAccountRateSyncHandler(service *accountratesyncapp.Service) *AccountRateSyncHandler {
	return &AccountRateSyncHandler{service: service}
}

type accountRateSyncRequest struct {
	Protocol string `json:"protocol"`
	Limit    int    `json:"limit"`
}

func (h *AccountRateSyncHandler) List(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusInternalServerError, "account rate sync service is not configured")
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	result, err := h.service.List(c.Request.Context(), c.Query("protocol"), limit)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *AccountRateSyncHandler) Sync(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusInternalServerError, "account rate sync service is not configured")
		return
	}
	var req accountRateSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	result, err := h.service.Sync(c.Request.Context(), req.Protocol, req.Limit)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *AccountRateSyncHandler) RetryAccount(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusInternalServerError, "account rate sync service is not configured")
		return
	}
	accountID, ok := parseInt64Path(c, "accountID", "invalid local account id")
	if !ok {
		return
	}
	row, err := h.service.RetryAccount(c.Request.Context(), accountID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, row)
}

func (h *AccountRateSyncHandler) Rename(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusInternalServerError, "account rate sync service is not configured")
		return
	}
	historyID, ok := parseInt64Path(c, "historyID", "invalid account rate sync history id")
	if !ok {
		return
	}
	row, err := h.service.RenameFromHistory(c.Request.Context(), historyID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, row)
}

func (h *AccountRateSyncHandler) RenameMatched(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusInternalServerError, "account rate sync service is not configured")
		return
	}
	var req accountRateSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	result, err := h.service.RenameMatched(c.Request.Context(), req.Protocol, req.Limit)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *AccountRateSyncHandler) ClearHistory(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusInternalServerError, "account rate sync service is not configured")
		return
	}
	count, err := h.service.ClearHistory(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{"deleted": count})
}

func parseInt64Path(c *gin.Context, key string, message string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, message)
		return 0, false
	}
	return id, true
}
