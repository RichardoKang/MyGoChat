package router

import (
	"MyGoChat/api/v1"
	"MyGoChat/internal/middleware"
	"MyGoChat/pkg/common/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	routers := gin.Default()

	groups := routers.Group("/api")
	{
		groups.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "Hello, It`s My Go!!!!!")
		})
	}

	userRoutes := groups.Group("/user")
	{
		userRoutes.POST("/register", v1.Register)
		userRoutes.POST("/login", v1.Login)
	}

	// Protected routes
	protectedRoutes := groups.Group("/")
	protectedRoutes.Use(middleware.JWTAuthMiddleware())
	{
		protectedRoutes.GET("/protected", func(c *gin.Context) {
			username := c.MustGet("username").(string)
			c.JSON(http.StatusOK, response.SuccessMsg(gin.H{"message": "Welcome to the protected area, " + username}))
		})
	}

	return routers
}
