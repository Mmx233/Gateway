package gateway

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type ApiConf struct {
	//Addr without protocol, domain and path only.
	// only support http and ws protocol for intranet usage.
	Addr string

	// api

	Client       *http.Client
	ErrorHandler func(c *gin.Context, e error)
	Middleware   gin.HandlerFunc

	// match
	// Adding multiple only the first one takes effect, same below

	AllowRequest    func(c *gin.Context) bool
	AllowAll        bool
	MatchPathPrefix string

	// proxy options

	TrimPath       func(path string) string
	TrimPathPrefix string

	Request func(req *http.Request)
}
