package gateway

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func Proxy(conf *ApiConf) gin.HandlerFunc {
	// 配置检查
	if conf.Addr == "" {
		panic("gateway backend addr required")
	}
	if conf.AllowRequest == nil {
		if conf.AllowAll {
			conf.AllowRequest = AllowAll
		} else if conf.MatchPathPrefix != "" {
			conf.AllowRequest = MatchPathPrefix(conf.MatchPathPrefix)
		} else {
			panic("gateway match condition required")
		}
	}
	if conf.Transport == nil {
		conf.Transport = http.DefaultClient.Transport
	}
	if conf.TrimPath == nil && conf.TrimPathPrefix != "" {
		conf.TrimPath = func(path string) string {
			return strings.TrimPrefix(path, conf.TrimPathPrefix)
		}
	}

	targetUrl, e := url.Parse("http://" + conf.Addr)
	if e != nil {
		panic(e)
	}
	proxyHandler := httputil.NewSingleHostReverseProxy(targetUrl)
	proxyHandler.Transport = conf.Transport
	if conf.BufferPool == nil {
		conf.BufferPool = &TransBuffPool{}
	}
	proxyHandler.BufferPool = conf.BufferPool
	proxyHandler.ErrorHandler = conf.ErrorHandler
	if conf.RequestInterceptor != nil {
		rawDirector := proxyHandler.Director
		proxyHandler.Director = func(request *http.Request) {
			rawDirector(request)
			conf.RequestInterceptor(request)
		}
	}

	control := func(c *gin.Context) {
		if !conf.AllowRequest(c) {
			return
		}

		if conf.TrimPath != nil {
			c.Request.URL.Path = conf.TrimPath(c.Request.URL.Path)
		}

		//转发请求
		proxyHandler.ServeHTTP(c.Writer, c.Request)
	}

	if conf.Middleware != nil {
		return func(c *gin.Context) {
			conf.Middleware(c)
			if !c.IsAborted() {
				control(c)
			}
		}
	}

	return control
}
