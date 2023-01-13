package quotes

import (
	"log"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	"github.com/wetrycode/tegenaria"
)
var exampleLog *logrus.Entry = tegenaria.GetLogger("example")

type ExampleSpider struct{
	Name     string
	FeedUrls []string
}
// QuotesbotSpider an example of tegenaria item
type QuotesbotItem struct {
	Text   string
	Author string
	Tags   string
}
func (e *ExampleSpider)StartRequest(req chan<- *tegenaria.Context){
	for i := 0; i < 1000; i++ {
		for _, url := range e.FeedUrls {
			// get a new request
			exampleLog.Infof("request %s", url)
			request := tegenaria.NewRequest(url, tegenaria.GET, e.Parser)
			// get request context
			ctx := tegenaria.NewContext(request, e)
			// send request context to engine
			req <- ctx
		}
	}
}

// Parser parse response ,it can generate ItemMeta and send to engine
// it also can generate new Request
func (e *ExampleSpider) Parser(resp *tegenaria.Context,req chan<- *tegenaria.Context) error{
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
		exampleLog.Infof("text:%s,author:%s, tag: %s",qText, author, tags)
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
			// ready to send a new request context to engine
			newRequest := tegenaria.NewRequest(s, tegenaria.GET, e.Parser)
			newCtx := tegenaria.NewContext(newRequest, e)
			req <- newCtx
		}
	}
	return nil
}
// ErrorHandler it is used to handler all error recive from engine
func (e *ExampleSpider) ErrorHandler(err *tegenaria.Context, req chan<- *tegenaria. Context){
	

}
// GetName get spider name
func (e *ExampleSpider) GetName() string{
	return e.Name
}

