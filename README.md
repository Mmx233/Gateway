# Gateway

golang 简易网关

```shell
~$ go get github.com/Mmx233/Gateway/v2
```

```go
package main

import (
	gateway "github.com/Mmx233/Gateway/v2"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func Router() {
	E := gin.Default()

	// 代理所有流量
	E.Use(gateway.Proxy(&gateway.ApiConf{
		Addr:               "workload.address:8080",
		Transport:          http.DefaultTransport,
		ErrorHandler: func(_ http.ResponseWriter, _ *http.Request, err error) {
			log.Println("error: request backend failed:", err)
		},
		Middleware: func(c *gin.Context) {
			// do something here
			if value,_:=c.Get("example");value=="" {
				c.AbortWithStatus(403)
			}
		},
		AllowAll:           true,
	}))
	
	// 根据路径前缀匹配放行
	E.Use(gateway.Proxy(&gateway.ApiConf{
		Addr:      "workload.address:8080",
		Transport: http.DefaultTransport,
		ErrorHandler: func(_ http.ResponseWriter, _ *http.Request, err error) {
			log.Println("error: request backend failed:", err)
		},
		MatchPathPrefix:    "/example/",
		RequestInterceptor: func(request *http.Request) {
			// do something here
			request.URL.RawQuery="?example=1"
		},
	}))

	// 自定义放行
	E.Use(gateway.Proxy(&gateway.ApiConf{
		Addr:      "workload.address:8080",
		Transport: http.DefaultTransport,
		AllowRequest: func(c *gin.Context) bool {
			return c.GetHeader("example")=="1"
		},
	}))
}
```