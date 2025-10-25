package middleware

import (
	"MyGoChat/internal/logic/service"
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/token"
	"net/http"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware 用来验证JWT令牌的中间件，确保请求携带有效的令牌。
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查token
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			tokenString = c.Query("token")
			if tokenString == "" {
				c.JSON(http.StatusUnauthorized, response.FailMsg("Authorization header or token query parameter required"))
				c.Abort()
				return
			}
		}

		// 检查携带的token格式是否正确
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		claims, err := token.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.FailMsg("Invalid token"))
			c.Abort()
			return
		}

		user, err := service.UserService.GetUserByUuid(claims.UserUuid)
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.FailMsg("Invalid user"))
			c.Abort()
			return
		}

		c.Set("userID", user.ID)
		c.Set("useruuid", claims.UserUuid)
		c.Set("username", claims.Username)
		c.Next()
	}
}
