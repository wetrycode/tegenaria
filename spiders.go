// Copyright 2022 vforfreedom96@gmail.com
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

type SpiderInterface interface {
	// StartRequest make new request by feed urls
	StartRequest(req chan<- *Context)

	// Parser parse response ,it can generate ItemMeta and send to engine
	// it also can generate new Request
	Parser(resp *Context, item chan<- *ItemMeta, req chan<- *Context) error

	// ErrorHandler it is used to handler all error recive from engine
	ErrorHandler(err *HandleError, req chan<- *Context)

	// GetName get spider name
	GetName() string
}

// BaseSpider base spider
type BaseSpider struct {
	// Name spider name
	Name     string

	// FeedUrls feed urls
	FeedUrls []string
}

type Spiders struct {
	SpidersModules map[string]SpiderInterface
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
func NewSpiders() *Spiders {
	onceSpiders.Do(func() {
		SpidersList = &Spiders{
			SpidersModules: make(map[string]SpiderInterface),
		}
	})
	return SpidersList
}
func (s *Spiders) Register(spider SpiderInterface) error {
	if len(spider.GetName()) == 0 {
		return ErrEmptySpiderName
	}
	if _, ok := s.SpidersModules[spider.GetName()]; ok {
		return ErrDuplicateSpiderName
	} else {
		s.SpidersModules[spider.GetName()] = spider
		return nil
	}
}
func (s *Spiders) GetSpider(name string) (SpiderInterface, error) {
	if _, ok := s.SpidersModules[name]; !ok {
		return nil, ErrSpiderNotExist
	} else {
		return s.SpidersModules[name], nil
	}
}
