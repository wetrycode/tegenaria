// Copyright 2022 geebytes
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	GetFeedUrls()[]string
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
	Parsers        map[string]Parser
}

var SpidersList *Spiders
var onceSpiders sync.Once

func NewBaseSpider(name string, feedUrls []string) *BaseSpider {
	return &BaseSpider{
		Name:     name,
		FeedUrls: feedUrls,
	}
}
func (s *BaseSpider) StartRequest(req chan<- *Context) {
	// StartRequest start feed urls request
}

// Parser parse request respone
// it will send item or new request to engine
func (s *BaseSpider) Parser(resp *Context, item chan<- *ItemMeta, req chan<- *Context) error {
	return nil
}
func (s *BaseSpider) ErrorHandler(err *HandleError) {
	// ErrorHandler error handler

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
