package initialize

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
)

// WebDist 返回前端打包目录：优先环境变量 WEB_DIST，其次常见相对路径
func WebDist() string {
	if dir := os.Getenv("WEB_DIST"); dir != "" {
		return dir
	}

	candidates := []string{
		"./dist",
		"../soybean-admin/dist",
		"../../soybean-admin/dist",
		"E:/ai/soybean-admin/dist",
	}

	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
			return dir
		}
	}

	return ""
}

// selfPort 返回当前进程监听端口，供 /proxy-default 反代回自己
func selfPort() string {
	if port := os.Getenv("SERVER_PORT"); port != "" {
		return port
	}

	if global.GVA_CONFIG.System.Addr > 0 {
		return strconv.Itoa(global.GVA_CONFIG.System.Addr)
	}

	return "8888"
}

// MountWebDist 用 Go 直接托管前端产物：不需要 nginx
//   - /proxy-default/xxx 反代回本进程，去掉前缀后就是真实接口
//   - 未命中的路径回退 index.html（单页应用路由）
func MountWebDist(r *gin.Engine) {
	dist := WebDist()

	if dist != "" {
		absDist, err := filepath.Abs(dist)

		if err == nil {
			indexFile := filepath.Join(absDist, "index.html")

			r.NoRoute(func(c *gin.Context) {
				requestPath := c.Request.URL.Path

				// 接口路由没命中时返回 JSON 404，不要回退成 index.html
				for _, prefix := range []string{"/api", "/swagger", "/proxy-default"} {
					if strings.HasPrefix(requestPath, prefix) {
						c.JSON(404, gin.H{"code": 404, "msg": "接口不存在"})
						return
					}
				}

				rel := strings.TrimPrefix(requestPath, "/")
				if rel != "" {
					file := filepath.Join(absDist, rel)

					if info, err := os.Stat(file); err == nil && !info.IsDir() {
						c.File(file)
						return
					}
				}

				c.File(indexFile)
			})
		}
	}

	// 前端 .env.prod 里配的接口前缀：/proxy-default 去掉后反代给本进程
	target := &url.URL{Scheme: "http", Host: "127.0.0.1:" + selfPort()}
	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/proxy-default")
		req.Host = target.Host
	}

	r.Any("/proxy-default/*path", gin.WrapH(proxy))
}