// MIT License

// Copyright (c) 2023 wetrycode

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package tegenaria

import "sync"

// SpiderInterface Tegenaria spider interface, developer can custom spider must be based on
// this interface to achieve custom spider.

// SpiderInterface Tegenaria spider interface, developer can custom spider must be based on
// this interface to achieve custom spider.

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

// BaseSpider base spider
type BaseSpider struct {
	// Name spider name
	Name string

	// FeedUrls feed urls
	FeedUrls []string
}

// Spiders 全局spiders管理器
// 用于接收注册的SpiderInterface实例
type Spiders struct {
	// SpidersModules spider名称和spider实例的映射
	SpidersModules map[string]SpiderInterface
	// Parsers parser函数名和函数的映射
	// 用于序列化和反序列化
	Parsers map[string]Parser
}

var SpidersList *Spiders
var onceSpiders sync.Once

func NewBaseSpider(name string, feedUrls []string) *BaseSpider {
	return &BaseSpider{
		Name:     name,
		FeedUrls: feedUrls,
	}
}

// NewSpiders 构建Spiders实例
func NewSpiders() *Spiders {
	onceSpiders.Do(func() {
		SpidersList = &Spiders{
			SpidersModules: make(map[string]SpiderInterface),
			Parsers:        make(map[string]Parser),
		}
	})
	return SpidersList
}

// Register spider实例注册到Spiders.SpidersModules
func (s *Spiders) Register(spider SpiderInterface) error {
	// 爬虫名不能为空
	if len(spider.GetName()) == 0 {
		return ErrEmptySpiderName
	}
	// 爬虫名不允许重复
	if _, ok := s.SpidersModules[spider.GetName()]; ok {
		return ErrDuplicateSpiderName
	} else {
		s.SpidersModules[spider.GetName()] = spider
		return nil
	}
}

// GetSpider 通过爬虫名获取spider实例
func (s *Spiders) GetSpider(name string) (SpiderInterface, error) {
	if _, ok := s.SpidersModules[name]; !ok {
		return nil, ErrSpiderNotExist
	} else {
		return s.SpidersModules[name], nil
	}
}
