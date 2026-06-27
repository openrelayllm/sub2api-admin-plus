package adminplus

import (
	"encoding/json"
	"net/http"

	purityapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type PurityHandler struct {
	service     *purityapp.Service
	authService *service.AuthService
}

func NewPurityHandler(purityService *purityapp.Service, authService *service.AuthService) *PurityHandler {
	return &PurityHandler{service: purityService, authService: authService}
}

type publicPurityCheckRequest struct {
	Provider        string `json:"provider"`
	APIBaseURL      string `json:"api_base_url" binding:"required"`
	APIKey          string `json:"api_key" binding:"required"`
	ModelID         string `json:"model_id"`
	CheckTokenUsage *bool  `json:"check_token_usage"`
	TurnstileToken  string `json:"turnstile_token"`
}

type accountPurityCheckRequest struct {
	Provider string `json:"provider"`
	ModelID  string `json:"model_id"`
}

func (h *PurityHandler) PublicCheck(c *gin.Context) {
	h.runPublicCheck(c, true, false)
}

func (h *PurityHandler) PublicCheckStream(c *gin.Context) {
	h.runPublicCheckStream(c, true, false)
}

func (h *PurityHandler) WebCheck(c *gin.Context) {
	h.runPublicCheck(c, true, false)
}

func (h *PurityHandler) WebCheckStream(c *gin.Context) {
	h.runPublicCheckStream(c, true, false)
}

func (h *PurityHandler) APICheck(c *gin.Context) {
	h.runPublicCheck(c, false, true)
}

func (h *PurityHandler) APICheckStream(c *gin.Context) {
	h.runPublicCheckStream(c, false, true)
}

func (h *PurityHandler) runPublicCheck(c *gin.Context, verifyTurnstile bool, developerAPI bool) {
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	var req publicPurityCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	clientIP := ip.GetClientIP(c)
	if verifyTurnstile && h != nil && h.authService != nil {
		if err := h.authService.VerifyTurnstile(c.Request.Context(), req.TurnstileToken, clientIP); response.ErrorFrom(c, err) {
			return
		}
	}
	if h == nil || h.service == nil {
		response.InternalError(c, "purity service is not configured")
		return
	}
	input := purityapp.PublicCheckInput{
		Provider:       req.Provider,
		APIBaseURL:     req.APIBaseURL,
		APIKey:         req.APIKey,
		ModelID:        req.ModelID,
		ClientIP:       clientIP,
		SkipTokenAudit: req.CheckTokenUsage != nil && !*req.CheckTokenUsage,
	}
	var report *purityapp.PublicReport
	var err error
	if developerAPI {
		report, err = h.service.RunDeveloperAPICheck(c.Request.Context(), input)
	} else {
		report, err = h.service.RunPublicCheck(c.Request.Context(), input)
	}
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, report)
}

func (h *PurityHandler) runPublicCheckStream(c *gin.Context, verifyTurnstile bool, developerAPI bool) {
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	var req publicPurityCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	clientIP := ip.GetClientIP(c)
	if verifyTurnstile && h != nil && h.authService != nil {
		if err := h.authService.VerifyTurnstile(c.Request.Context(), req.TurnstileToken, clientIP); response.ErrorFrom(c, err) {
			return
		}
	}
	if h == nil || h.service == nil {
		response.InternalError(c, "purity service is not configured")
		return
	}

	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("X-Accel-Buffering", "no")
	encoder := json.NewEncoder(c.Writer)
	var writeErr error
	input := purityapp.PublicCheckInput{
		Provider:       req.Provider,
		APIBaseURL:     req.APIBaseURL,
		APIKey:         req.APIKey,
		ModelID:        req.ModelID,
		ClientIP:       clientIP,
		SkipTokenAudit: req.CheckTokenUsage != nil && !*req.CheckTokenUsage,
	}
	emit := func(event purityapp.PublicCheckEvent) {
		if writeErr != nil {
			return
		}
		if !c.Writer.Written() {
			c.Status(http.StatusOK)
		}
		writeErr = encoder.Encode(event)
		if writeErr == nil {
			c.Writer.Flush()
		}
	}
	var report *purityapp.PublicReport
	var err error
	if developerAPI {
		report, err = h.service.RunDeveloperAPICheckStream(c.Request.Context(), input, emit)
	} else {
		report, err = h.service.RunPublicCheckStream(c.Request.Context(), input, emit)
	}
	if err != nil {
		if !c.Writer.Written() {
			response.ErrorFrom(c, err)
			return
		}
		_ = encoder.Encode(purityapp.PublicCheckEvent{
			Type:         purityapp.PublicCheckEventError,
			Report:       report,
			ErrorClass:   "stream_error",
			ErrorMessage: err.Error(),
		})
		c.Writer.Flush()
		return
	}
}

func (h *PurityHandler) AccountCheckStream(c *gin.Context) {
	accountID, ok := parseAccountIDParam(c)
	if !ok {
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	var req accountPurityCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	if h == nil || h.service == nil {
		response.InternalError(c, "purity service is not configured")
		return
	}

	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("X-Accel-Buffering", "no")
	encoder := json.NewEncoder(c.Writer)
	var writeErr error
	report, err := h.service.RunAccountCheckStream(c.Request.Context(), purityapp.AccountCheckInput{
		AccountID: accountID,
		Provider:  req.Provider,
		ModelID:   req.ModelID,
	}, func(event purityapp.PublicCheckEvent) {
		if writeErr != nil {
			return
		}
		if !c.Writer.Written() {
			c.Status(http.StatusOK)
		}
		writeErr = encoder.Encode(event)
		if writeErr == nil {
			c.Writer.Flush()
		}
	})
	if err != nil {
		if !c.Writer.Written() {
			response.ErrorFrom(c, err)
			return
		}
		_ = encoder.Encode(purityapp.PublicCheckEvent{
			Type:         purityapp.PublicCheckEventError,
			Report:       report,
			ErrorClass:   "stream_error",
			ErrorMessage: err.Error(),
		})
		c.Writer.Flush()
		return
	}
}
