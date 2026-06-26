package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterPublicProxyAIRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	if h == nil || h.AdminPlus == nil || h.AdminPlus.PublicProxyAI == nil {
		return
	}
	public := v1.Group("/public/proxyai")
	{
		public.GET("/summary", h.AdminPlus.PublicProxyAI.Summary)
		public.HEAD("/summary", h.AdminPlus.PublicProxyAI.Summary)
		public.GET("/sites", h.AdminPlus.PublicProxyAI.ListSites)
		public.HEAD("/sites", h.AdminPlus.PublicProxyAI.ListSites)
		public.GET("/sites/:slug", h.AdminPlus.PublicProxyAI.GetSite)
		public.HEAD("/sites/:slug", h.AdminPlus.PublicProxyAI.GetSite)
	}
}
