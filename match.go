package gateway

import (
	"github.com/gin-gonic/gin"
	"strings"
)

func AllowAll(c *gin.Context) bool {
	return true
}

func MatchPathPrefix(prefix string) func(c *gin.Context) bool {
	return func(c *gin.Context) bool {
		return strings.HasPrefix(c.Request.URL.Path, prefix)
	}
}
