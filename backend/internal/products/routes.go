package products

import (
	"Tugas_Akhir/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, handler *Handler) {
	// Semua endpoint produk hanya bisa diakses owner dan admin.
	router.Use(auth.RequireRoles("owner", "admin"))

	router.POST("", handler.Create)
	router.GET("", handler.GetAll)
	router.GET("/:id", handler.GetByID)
	router.PUT("/:id", handler.Update)
	router.DELETE("/:id", handler.Delete)
}
