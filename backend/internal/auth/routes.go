package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, handler *Handler) {
	router.POST("/register", handler.Register)
	router.POST("/login", handler.Login)
}
