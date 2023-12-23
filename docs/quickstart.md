## 安装
1. go 版本要求>=1.19 

```bash
go get -u github.com/wetrycode/tegenaria@latest
```

2. 在您的项目中导入

```go
import "github.com/wetrycode/tegenaria"
```

## 快速开始
 
### 编写第一个爬虫  
Tegenaria 的所有爬虫都是[SpiderInterface](SpiderInterface)的实例，包含如下几个部分：

- 必须有一个爬虫启动的入口`StartRequest(req chan<- *Context)`用于发起初始请求；

- 一个默认的解析函数`Parser(resp *Context, req chan<- *Context) error`用于解析请求响应;

- 错误处理函数`ErrorHandler(err *Context, req chan<- *Context)`处理单条请求整个生命周期过程中捕获到的错误

- 爬虫名获取接口`GetName() string`

- 爬虫种子 url 获取接口`GetFeedUrls()[]string`

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
    Text   string
    Author string
    Tags   string
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
            Text:   qText,
            Author: author,
            Tags:   strings.Join(tags, ","),
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
func (e *ExampleSpider) GetFeedUrls() []string {
    return e.FeedUrls
}
```

### 初始化引擎

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

### 启动爬虫

```go
func main() {
    opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithLimiter(tegenaria.NewDefaultLimiter(64))}
    engine := NewQuotesEngine(opts...)
    engine.Execute("example")
}
```

```shell
go run main.go crawl example
```

### 如何构造新的请求?

- 在`StartRequest`中请求通过`req chan<- *tegenaria.Context`将请求提交到引擎进行调度处理，其中`\*tegenaria.Context``是单条数抓取过程的数据流通单元，贯穿单条数据抓取的整个生命周期，其数据结构参加代码[Context](../context.go)

- Request 对象是组成 Context 对象的成员之一，通过`tegenaria.NewRequest`进行构造，其核心参数为 url,对应的绑定到爬虫实例并与之相关的解析函数及请求方法

- 构造新的请求步骤如下:

  - 创建新的 request 对象

    ```go
    // 依次传入url、请求方式和爬虫实例对应的解析函数
    request := tegenaria.NewRequest(url, tegenaria.GET, e.Parser)
    ```

  - 构造新的 Context 并传入 request

    ```go
    ctx := tegenaria.NewContext(request, e)
    ```

  - 通过 channel 发送到引擎

    ```go
    req <- ctx
    ```

### 如何构造新的 item

- 在解析函数`Parser(resp *tegenaria.Context, req chan<- *tegenaria.Context) error`中 item 通过 resp.Items channel 将 item 专递到引擎，resp 实际上是 Context 对象其内置了当前请求实例所需的 items channel

- items channel 传递的是[ItemMeta](items.go)对象，包含两个属性 CtxID 即请求 id 以及 ItemInterface 实例是实际上的 item 数据

- 构造一条 item 的流程如下:

  - 调用`tegenaria.NewItem`生成新的 ItemMeta 对象

    ```go
    itemCtx := tegenaria.NewItem(resp, &quoteItem)
    ```

  - 将 item 发送到引擎

    ```go
    resp.Items <- itemCtx
    ```

