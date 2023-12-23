#### DownloadMiddleware

下载中间件是`MiddlewaresInterface`接口的实现.

下载中间主要用于在进入下载器之前对下载对象`Request`进行修饰，例如添加代理和请求头，也会对响应体对象`Response`进入解析函数之前进行处理,下载中间包含两个函数:

- `ProcessRequest(ctx *tegenaria.Context) error`在进入下载器之前对`Request`对象进行处理

- `ProcessResponse(ctx *tegenaria.Context, req chan<- *tegenaria.Context) error`在解析之前对响应体进行处理

一个 spider 引擎实例可以注册多个下载中间件，并按照优先级执行，引擎会通过`GetPriority() int`获取优先级，中间件调用规则如下:

- 数字越小优先级越高，`ProcessRequest`高优先级先执行

- `ProcessResponse`则反之，优先级越高越晚执行

下载中间件还需要实现`GetName() string`，引擎通过该方法获取中间件名称.

#### 定义下载中间件

```go
import (
    "fmt"

    "github.com/wetrycode/tegenaria"
)

// HeadersDownloadMiddler 请求头设置下载中间件
type HeadersDownloadMiddler struct {
    // Priority 优先级
    Priority int
    // Name 中间件名称
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

        "User-Agent":   "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.64 Safari/537.36",
    }

    for key, value := range header {
        if _, ok := ctx.Request.Header[key]; !ok {
            ctx.Request.Header[key] = value
        }
    }
    return nil
}
// ProcessResponse 用于处理请求成功之后的response
// 执行顺序你优先级，及优先级越高执行顺序越晚
func (m HeadersDownloadMiddler) ProcessResponse(ctx *tegenaria.Context, req chan<- *tegenaria.Context) error {
    if ctx.Response.Status!=200{
        return fmt.Errorf("非法状态码:%d", ctx.Response.Status)
    }
    return nil

}
func (m HeadersDownloadMiddler) GetName() string {
    return m.Name
}
```

#### 向引擎注册下载中间件

```go
middleware := HeadersDownloadMiddler{Priority: 1, Name: "AddHeader"}
Engine.RegisterDownloadMiddlewares(middleware)
```