#### 爬虫组件
基于tegenaria开发的所有爬虫项目，其爬虫实例必须继承和实现```SpiderInterface```接口以满足引擎的适配要求  
接口定义如下:
```go
type SpiderInterface interface {
	// StartRequest 通过GetFeedUrls()获取种子
	// urls并构建初始请求
	StartRequest(req chan<- *Context)

	// Parser 默认的请求响应解析函数
	// 在解析过程中生成的新的请求可以推送到req channel
	Parser(resp *Context, req chan<- *Context) error

	// ErrorHandler 错误处理函数，允许在此过程中生成新的请求
	// 并推送到req channel
	ErrorHandler(err *Context, req chan<- *Context)

	// GetName 获取spider名称
	GetName() string
	// GetFeedUrls 获取种子urls
	GetFeedUrls() []string
}
```
#### 说明  

- ```StartRequest(req chan<- *Context)```  启动爬虫时主动调用并会发起爬虫的请求
    - 从引擎传入一个用于接收`Context`对象的channel,实现爬虫实例与引擎之前的请求对象交互  
    - 该函数在引擎中会以一个单独的协程异步执行，以提高引擎的调度效率
    - 一个简单的示例
    ```go
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
    ```

- ```Parser(resp *Context, req chan<- *Context) error``` 请求响应的解析函数实际上是函数类型```type Parser func(resp *Context, req chan<- *Context) error```的实现
    - 在构建请求时通过传参将其绑定到需求解析的请求上,具体的用法如上述示例,  
    - 一个简单的实现示例如下:
    ```go
    func (e *ExampleSpider) Parser(resp *tegenaria.Context, req chan<- *tegenaria.Context) error {
    	text, _ := resp.Response.String()
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
    ```

    - 上述的解析函数中，在生成item之后通过```Context.Items```提交到引擎进行处理；  
    - 在进行翻页操作时通过```req chan<- *tegenaria.Context```参数提交到引擎，也就是说该函数在生成item的同时也允许生成新的请求  

- ```ErrorHandler(err *Context, req chan<- *Context)``` 处理捕获到错误信息
    - ```err```参数是在最开始构建请求时生的```Context```对象，内部包含了请求的全部信息
    - ```req``` 参数是引擎传入的用于传递```Request```对象的channel,`ErrorHandler`该函数允许用户在处理错误后重新生成新的`Request`对象进行重试操作，  
    注意如果进行重试应该条请求应当跳过去重逻辑，即传入`RequestWithDoNotFilter(true)`   
    - 一个简单的示例  
    ```go
    func (e *ExampleSpider) ErrorHandler(err *tegenaria.Context, req chan<- *tegenaria.Context) {
	srcRequest,_:=err.Request.ToMap()
	newRequest := tegenaria.RequestFromMap(srcRequest, tegenaria.RequestWithDoNotFilter(true))
	newCtx := tegenaria.NewContext(newRequest, e)
	req <- newCtx
    }
    ```

- ```GetName() string```获取爬虫名称  

- ```GetFeedUrls() []string``` 获取爬虫urls