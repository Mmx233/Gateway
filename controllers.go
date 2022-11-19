package gateway

import (
	"github.com/Mmx233/tool"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
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

		var reqPath string
		if conf.TrimPath != nil {
			reqPath = conf.TrimPath(c.Request.URL.Path)
		} else {
			reqPath = c.Request.URL.Path
		}

		if c.IsWebsocket() {
			var wsUrl = "ws://" + conf.Addr + reqPath
			if c.Request.URL.RawQuery != "" {
				wsUrl += "?" + c.Request.URL.RawQuery
			}
			proxyWs(conf.ErrorHandler, wsUrl, c)
			return
		}

		defer c.Request.Body.Close()

		//转发请求

		req, e := http.NewRequest(c.Request.Method, "http://"+conf.Addr+reqPath, c.Request.Body)
		if e != nil {
			conf.ErrorHandler(c, e)
			return
		}

		req.URL.RawQuery = c.Request.URL.RawQuery

		for k, v := range c.Request.Header {
			if strings.Contains(k, "Content-") {
				req.Header[k] = v
			}
		}

		if conf.RequestInterceptor != nil {
			conf.RequestInterceptor(c, req)
			if c.IsAborted() {
				return
			}
		}

		res, e := conf.Client.Do(req)
		if e != nil {
			conf.ErrorHandler(c, e)
			return
		}
		defer res.Body.Close()

		for k, v := range res.Header {
			if strings.Contains(k, "Content-") {
				c.Header(k, v[0])
			}
		}
		c.Status(res.StatusCode)

		buff := BuffPool.Get().([]byte)
		_, e = io.CopyBuffer(c.Writer, res.Body, buff)
		BuffPool.Put(buff)

		if e != nil {
			conf.ErrorHandler(c, e)
			return
		}

		c.Abort()
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
