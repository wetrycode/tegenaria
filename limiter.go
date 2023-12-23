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

import (
	"go.uber.org/ratelimit"
)

// LimitInterface 限速器接口
type LimitInterface interface {
	// checkAndWaitLimiterPass 检查当前并发量
	// 如果并发量达到上限则等待
	CheckAndWaitLimiterPass() error
	// setCurrrentSpider 设置当前正在的运行的spider
	SetCurrentSpider(spider SpiderInterface)
}

// defaultLimiter 默认的限速器
type DefaultLimiter struct {
	limiter ratelimit.Limiter
	spider  SpiderInterface
}

// NewDefaultLimiter 创建一个新的限速器
// limitRate 最大请求速率
func NewDefaultLimiter(limitRate int) *DefaultLimiter {
	return &DefaultLimiter{
		limiter: ratelimit.New(limitRate, ratelimit.WithoutSlack),
	}
}

// checkAndWaitLimiterPass 检查当前并发量
// 如果并发量达到上限则等待
func (d *DefaultLimiter) CheckAndWaitLimiterPass() error {
	d.limiter.Take()
	return nil
}

// setCurrrentSpider 设置当前的spider名
func (d *DefaultLimiter) SetCurrentSpider(spider SpiderInterface) {
	d.spider = spider

}
