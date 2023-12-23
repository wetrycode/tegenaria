package quotes

import (
	"fmt"

	"github.com/wetrycode/tegenaria"
)

// HeadersDownloadMiddler 请求头设置下载中间件
type HeadersDownloadMiddler struct {
	// Priority 优先级
	Priority int
	// Name 中间件名称
	Name string
}

// ProxyDownloadMiddler 代理挂载中间件
type ProxyDownloadMiddler struct {
	Priority int
	Name     string
}

// GetPriority 获取优先级，数字越小优先级越高
func (m HeadersDownloadMiddler) GetPriority() int {
	return m.Priority
}

// ProcessRequest 处理request请求对象
// 此处用于增加请求头
// 按优先级执行
func (m HeadersDownloadMiddler) ProcessRequest(ctx *tegenaria.Context) error {
	header := map[string]string{
		"Accept":       "*/*",
		"Content-Type": "application/json",

		"User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.64 Safari/537.36",
	}

	for key, value := range header {
		if _, ok := ctx.Request.Headers[key]; !ok {
			ctx.Request.Headers[key] = value
		}
	}
	return nil
}

// ProcessResponse 用于处理请求成功之后的response
// 执行顺序你优先级，及优先级越高执行顺序越晚
func (m HeadersDownloadMiddler) ProcessResponse(ctx *tegenaria.Context, req chan<- *tegenaria.Context) error {
	if ctx.Response.Status != 200 {
		return fmt.Errorf("非法状态码:%d", ctx.Response.Status)
	}
	return nil

}
func (m HeadersDownloadMiddler) GetName() string {
	return m.Name
}
