package quotes

import (
	"log"
	"net/url"
	"strings"
	"time"

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
	for i := 0; i < 10; i++ {
		for _, url := range e.GetFeedUrls() {
			// 生成新的request 对象
			exampleLog.Infof("request %s", url)
			request := tegenaria.NewRequest(url, tegenaria.GET, e.Parser)
			// 生成新的Context
			ctx := tegenaria.NewContext(request, e)
			// 将context发送到req channel
			time.Sleep(1)
			req <- ctx
		}
	}
}

// Parser 默认的解析函数
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
