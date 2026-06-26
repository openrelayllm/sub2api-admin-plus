package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterPublicProxyAIRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	if h == nil || h.AdminPlus == nil || h.AdminPlus.PublicProxyAI == nil {
		return
	}
	public := v1.Group("/public/proxyai", publicProxyAICORS())
	{
		public.OPTIONS("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		public.GET("/summary", h.AdminPlus.PublicProxyAI.Summary)
		public.HEAD("/summary", h.AdminPlus.PublicProxyAI.Summary)
		public.GET("/sites", h.AdminPlus.PublicProxyAI.ListSites)
		public.HEAD("/sites", h.AdminPlus.PublicProxyAI.ListSites)
		public.GET("/sites/:slug", h.AdminPlus.PublicProxyAI.GetSite)
		public.HEAD("/sites/:slug", h.AdminPlus.PublicProxyAI.GetSite)
	}
}

func publicProxyAICORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Accept, Content-Type")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
