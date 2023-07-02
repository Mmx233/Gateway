package gateway

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httputil"
)

type ApiConf struct {
	//Addr without protocol, domain and path only.
	// only support http and ws protocol for intranet usage.
	Addr string

	// api

	Transport    http.RoundTripper
	BufferPool   httputil.BufferPool
	ErrorHandler func(http.ResponseWriter, *http.Request, error)

	// match
	// Adding multiple only the first one takes effect, same below

	AllowRequest    func(c *gin.Context) bool
	AllowAll        bool
	MatchPathPrefix string

	// proxy options

	TrimPath       func(path string) string
	TrimPathPrefix string

	RequestInterceptor func(request *http.Request)
}
