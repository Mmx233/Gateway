package gateway

import (
	"github.com/Mmx233/tool"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"io"
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
	if conf.ErrorHandler == nil {
		conf.ErrorHandler = func(c *gin.Context, e error) {
			_ = c.AbortWithError(500, e)
		}
	}
	if conf.Client == nil {
		conf.Client = http.DefaultClient
	}
	if conf.TrimPath == nil && conf.TrimPathPrefix != "" {
		conf.TrimPath = func(path string) string {
			return strings.TrimPrefix(path, conf.TrimPathPrefix)
		}
	}

	control := func(c *gin.Context) {
		if !conf.AllowRequest(c) {
			return
		}

		if conf.TrimPath != nil {
			c.Request.URL.Path = conf.TrimPath(c.Request.URL.Path)
		}

		if c.IsWebsocket() {
			var wsUrl = "ws://" + conf.Addr + c.Request.URL.Path
			if c.Request.URL.RawQuery != "" {
				wsUrl += "?" + c.Request.URL.RawQuery
			}
			proxyWs(conf.ErrorHandler, wsUrl, c)
			return
		}

		defer c.Request.Body.Close()

		//转发请求
		targetUrl, e := url.Parse("http://" + conf.Addr)
		if e != nil {
			conf.ErrorHandler(c, e)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(targetUrl)
		proxy.Transport = conf.Client.Transport
		proxy.BufferPool = &TransBuffPool{}
		proxy.ErrorHandler = func(_ http.ResponseWriter, _ *http.Request, e error) {
			conf.ErrorHandler(c, e)
		}
		proxy.Director = func(request *http.Request) {
			if conf.RequestInterceptor != nil && !c.IsAborted() {
				conf.RequestInterceptor(c, request)
			}
		}
		if !c.IsAborted() {
			proxy.ServeHTTP(c.Writer, c.Request)
		}
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

func proxyWs(ErrorHandler func(c *gin.Context, e error), url string, c *gin.Context) {
	connS, _, e := websocket.DefaultDialer.Dial(url, nil)
	if e != nil {
		ErrorHandler(c, e)
		return
	}

	connC, e := UpgradeWs(c)
	if e != nil {
		ErrorHandler(c, e)
		return
	}

	var closeAll = func() {
		_ = connC.Close()
		_ = connS.Close()
	}

	var transfer = func(from, to *websocket.Conn) {
		defer tool.Recover()
		for {
			t, i, e := from.NextReader()
			if e != nil {
				closeAll()
				return
			}

			writer, e := to.NextWriter(t)
			if e != nil {
				closeAll()
				return
			}

			buf := BuffPool.Get().([]byte)
			_, e = io.CopyBuffer(writer, i, buf)
			BuffPool.Put(buf)
			if e != nil {
				closeAll()
				return
			}
		}
	}

	go transfer(connS, connC)
	go transfer(connC, connS)

	c.AbortWithStatus(101)
}
