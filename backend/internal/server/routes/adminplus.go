package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterAdminPlusRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	adminAuth middleware.AdminAuthMiddleware,
) {
	adminPlus := v1.Group("/admin-plus")
	adminPlus.Use(gin.HandlerFunc(adminAuth))
	{
		suppliers := adminPlus.Group("/suppliers")
		{
			suppliers.GET("", h.AdminPlus.Supplier.List)
			suppliers.POST("", h.AdminPlus.Supplier.Create)
			suppliers.GET("/:id", h.AdminPlus.Supplier.Get)
			suppliers.PATCH("/:id/status", h.AdminPlus.Supplier.UpdateStatus)
		}

		rates := adminPlus.Group("/rates")
		{
			rates.POST("/snapshots", h.AdminPlus.Rate.RecordSnapshot)
			rates.GET("/snapshots", h.AdminPlus.Rate.ListSnapshots)
			rates.GET("/events", h.AdminPlus.Rate.ListEvents)
			rates.PATCH("/events/:id/ack", h.AdminPlus.Rate.AcknowledgeEvent)
		}
	}
}
