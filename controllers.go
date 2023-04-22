package gateway

import (
	"github.com/Mmx233/tool"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"io"
	"log"
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
		log.Fatalln("无法解析目标地址:", e)
	}
	proxyHandler := httputil.NewSingleHostReverseProxy(targetUrl)
	proxyHandler.Transport = conf.Transport
	proxyHandler.BufferPool = &TransBuffPool{}
	proxyHandler.ErrorHandler = conf.ErrorHandler
	rawDirector := proxyHandler.Director
	proxyHandler.Director = func(request *http.Request) {
		rawDirector(request)
		if conf.RequestInterceptor != nil {
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

		if c.IsWebsocket() {
			var wsUrl = "ws://" + conf.Addr + c.Request.URL.Path
			if c.Request.URL.RawQuery != "" {
				wsUrl += "?" + c.Request.URL.RawQuery
			}
			proxyWs(conf.ErrorHandler, wsUrl, c)
			return
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

func proxyWs(ErrorHandler func(http.ResponseWriter, *http.Request, error), url string, c *gin.Context) {
	connS, _, e := websocket.DefaultDialer.Dial(url, nil)
	if e != nil {
		ErrorHandler(c.Writer, c.Request, e)
		return
	}

	connC, e := UpgradeWs(c)
	if e != nil {
		ErrorHandler(c.Writer, c.Request, e)
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
