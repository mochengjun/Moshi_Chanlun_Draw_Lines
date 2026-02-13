package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"kline-indicator-service/internal/models"
)

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string
	Issuer     string
	SkipPaths  []string // 跳过认证的路径
}

// Claims JWT Claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// jwtConfig 全局JWT配置
var jwtConfig *JWTConfig

// InitJWT 初始化JWT配置
func InitJWT(secret string, skipPaths []string) {
	jwtConfig = &JWTConfig{
		Secret:    secret,
		Issuer:    "kline-indicator-service",
		SkipPaths: skipPaths,
	}
}

// JWTAuth JWT认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否跳过认证
		if jwtConfig != nil {
			for _, path := range jwtConfig.SkipPaths {
				if strings.HasPrefix(c.Request.URL.Path, path) {
					c.Next()
					return
				}
			}
		}
		
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				models.ErrCodeUnauthorized,
				"缺少Authorization头",
			))
			c.Abort()
			return
		}
		
		// 解析Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				models.ErrCodeUnauthorized,
				"Authorization格式错误，应为: Bearer <token>",
			))
			c.Abort()
			return
		}
		
		tokenString := parts[1]
		
		// 解析token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtConfig.Secret), nil
		})
		
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				models.ErrCodeUnauthorized,
				"无效的token",
			))
			c.Abort()
			return
		}
		
		// 设置用户信息到上下文
		c.Set("UserID", claims.UserID)
		c.Set("Username", claims.Username)
		
		c.Next()
	}
}

// OptionalJWTAuth 可选的JWT认证中间件
func OptionalJWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}
		
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}
		
		tokenString := parts[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtConfig.Secret), nil
		})
		
		if err == nil && token.Valid {
			c.Set("UserID", claims.UserID)
			c.Set("Username", claims.Username)
		}
		
		c.Next()
	}
}
