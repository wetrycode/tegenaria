# 1.入门

## 1.1.编写第一个爬虫

Tegenaria的所有爬虫都是[SpiderInterface](SpiderInterface)的实例，包含如下几个部分：

- 必须有一个爬虫启动的入口`StartRequest(req chan<- *Context)`用于发起初始请求；

- 一个默认的解析函数`Parser(resp *Context, req chan<- *Context) error`用于解析请求响应;

- 错误处理函数`ErrorHandler(err *Context, req chan<- *Context)`处理单条请求整个生命周期过程中捕获到的错误

- 爬虫名获取接口`GetName() string`

- 爬虫种子url获取接口`GetFeedUrls()[]string`

```go
package main

import (

"log"

"net/url"

"strings"

"github.com/PuerkitoBio/goquery"

"github.com/sirupsen/logrus"

"github.com/wetrycode/tegenaria"

)

var exampleLog *logrus.Entry = tegenaria.GetLogger("example")

// ExampleSpider 定义一个spider

type ExampleSpider struct {

// Name 爬虫名

Name string

// 种子urls

FeedUrls []string

}

// QuotesbotSpider tegenaria item示例

type QuotesbotItem struct {

Text string

Author string

Tags string

}

// StartRequest 爬虫启动，请求种子urls

func (e *ExampleSpider) StartRequest(req chan<- *tegenaria.Context) {

for i := 0; i < 512; i++ {

for _, url := range e.GetFeedUrls() {

// 生成新的request 对象

exampleLog.Infof("request %s", url)

request := tegenaria.NewRequest(url, tegenaria.GET, e.Parser)

// 生成新的Context

ctx := tegenaria.NewContext(request, e)

// 将context发送到req channel

req <- ctx

}

}

}

// Parser 默认的解析函数

func (e *ExampleSpider) Parser(resp *tegenaria.Context, req chan<- *tegenaria.Context) error {

text := resp.Response.String()

doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))

if err != nil {

log.Fatal(err)

}

doc.Find(".quote").Each(func(i int, s *goquery.Selection) {

// For each item found, get the title

qText := s.Find(".text").Text()

author := s.Find(".author").Text()

tags := make([]string, 0)

s.Find("a.tag").Each(func(i int, s *goquery.Selection) {

tags = append(tags, s.Text())

})

// ready to send a item to engine

var quoteItem = QuotesbotItem{

Text: qText,

Author: author,

Tags: strings.Join(tags, ","),

}

exampleLog.Infof("text:%s,author:%s, tag: %s", qText, author, tags)

// 构建item发送到指定的channel

itemCtx := tegenaria.NewItem(resp, &quoteItem)

resp.Items <- itemCtx

})

doamin_url := resp.Request.Url

next := doc.Find("li.next")

if next != nil {

nextUrl, ok := next.Find("a").Attr("href")

if ok {

u, _ := url.Parse(doamin_url)

nextInfo, _ := url.Parse(nextUrl)

s := u.ResolveReference(nextInfo).String()

exampleLog.Infof("the next url is %s", s)

// 生成新的请求

newRequest := tegenaria.NewRequest(s, tegenaria.GET, e.Parser)

newCtx := tegenaria.NewContext(newRequest, e)

req <- newCtx

}

}

return nil

}

// ErrorHandler 异常处理函数,用于处理数据抓取过程中出现的错误

func (e *ExampleSpider) ErrorHandler(err *tegenaria.Context, req chan<- *tegenaria.Context) {

}

// GetName 获取爬虫名

func (e *ExampleSpider) GetName() string {

return e.Name

}

// GetFeedUrls 获取种子urls

func(e *ExampleSpider)GetFeedUrls()[]string{

return e.FeedUrls

}
```

## 1.2.初始化引擎

```go
// NewQuotesEngine 创建引擎
func NewQuotesEngine(opts ...tegenaria.EngineOption) *tegenaria.CrawlEngine {
    ExampleSpiderInstance := &ExampleSpider{
        Name:     "example",
        FeedUrls: []string{"http://quotes.toscrape.com/"},
    }

    Engine := tegenaria.NewEngine(opts...)
    // 注册spider
    Engine.RegisterSpiders(ExampleSpiderInstance)
    return Engine

}
```

## 1.3.启动爬虫

```go
func main() {
    opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithLimiter(tegenaria.NewDefaultLimiter(64))}
    engine := NewQuotesEngine(opts...)
    engine.Execute()
}
```

```shell
go run main.go crawl example
```

## 1.4.如何构造新的请求?

- 在`StartRequest`中请求通过`req chan<- *tegenaria.Context`将请求提交到引擎进行调度处理，其中`*tegenaria.Context``是单条数抓取过程的数据流通单元，贯穿单条数据抓取的整个生命周期，其数据结构参加代码[Context](context.go) 

- Request对象是组成Context对象的成员之一，通过`tegenaria.NewRequest`进行构造，其核心参数为url,对应的绑定到爬虫实例并与之相关的解析函数及请求方法

- 构造新的请求步骤如下:
  
  - 创建新的request对象
    
    ```go
    // 一次传入url、请求方式和爬虫实例对应的解析函数
    request := tegenaria.NewRequest(url, tegenaria.GET, e.Parser)
    ```
  
  - 构造新的Context并传入request
    
    ```go
    ctx := tegenaria.NewContext(request, e)
    ```
  
  - 通过channel发送到引擎
    
    ```go
    req <- ctx 
    ```

# 2.爬虫组件

## 2.1.下载中间件

下载中间件是`MiddlewaresInterface`接口的实现.

下载中间主要用于在进入下载器之前对下载对象`Request`进行修饰，例如添加代理和请求头，也会对响应体对象`Response`进入解析函数之前进行处理,下载中间包含两个函数:

- `ProcessRequest(ctx *tegenaria.Context) error`在进入下载器之前对`Request`对象进行处理

- `ProcessResponse(ctx *tegenaria.Context, req chan<- *tegenaria.Context) error`在解析之前对响应体进行处理

一个spider引擎实例可以注册多个下载中间件，并按照优先级执行，引擎会通过`GetPriority() int`获取优先级，中间件调用规则如下:

- 数字越小优先级越高，`ProcessRequest`高优先级先执行

- `ProcessResponse`则反之，优先级越高越晚执行

下载中间件还需要实现`GetName() string`，引擎通过该方法获取中间件名称.

### 2.1.1.定义下载中间件

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

### 2.1.2.向引擎注册下载中间件

```go
middleware := HeadersDownloadMiddler{Priority: 1, Name: "AddHeader"}
Engine.RegisterDownloadMiddlewares(middleware)
```

## 2.2.Items Pipeline

item pipelines用于对item进行处理，例如持久化存储到数据库、数据去重等操作，所有的pipeline都应实现`PipelinesInterface`接口其中包含的两个函数如下:

- `GetPriority() int`给引擎提供当前pipeline的优先级，请注意数字越低优先级越高越早调用

- `ProcessItem(spider SpiderInterface, item *ItemMeta) error` 处理item的核心逻辑

### 2.2.1.定义item pipeline

```go
// QuotesbotItemPipeline tegenaria.PipelinesInterface 接口示例
// 用于item处理的pipeline
type QuotesbotItemPipeline struct {
    Priority int
}
// ProcessItem item处理函数
func (p *QuotesbotItemPipeline) ProcessItem(spider tegenaria.SpiderInterface, item *tegenaria.ItemMeta) error {
    i:=item.Item.(*QuotesbotItem)
    exampleLog.Infof("%s 抓取到数据:%s",item.CtxId, i.Text)
    return nil

}

// GetPriority 获取该pipeline的优先级
func (p *QuotesbotItemPipeline) GetPriority() int {
    return p.Priority
}
```

### 2.2.2.响应引擎注册piplines

```go
pipe := QuotesbotItemPipeline{Priority: 1}
Engine.RegisterPipelines(pipe)
```

## 
