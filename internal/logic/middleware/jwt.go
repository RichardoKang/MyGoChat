package middleware

import (
	"MyGoChat/internal/logic/service"
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/token"
	"net/http"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware creates a middleware function that validates JWT tokens.
// It takes a UserService to verify the user exists.
func JWTAuthMiddleware(userService *service.UserService) gin.HandlerFunc {
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

		user, err := userService.GetUserByUuid(claims.UserUuid)
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
