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

package distributed

import "github.com/wetrycode/tegenaria"

// DistributedHooks 分布式事件监听器
type DistributedHooks struct {
	worker tegenaria.DistributedWorkerInterface
	spider tegenaria.SpiderInterface
}

// DistributedHooks 构建新的分布式监听器组件对象
func NewDistributedHooks(worker tegenaria.DistributedWorkerInterface) *DistributedHooks {
	return &DistributedHooks{
		worker: worker,
	}

}

// Start 用于处理分布式模式下的START事件
func (d *DistributedHooks) Start(params ...interface{}) error {
	return d.worker.AddNode()

}

// Stop 用于处理分布式模式下的STOP事件
func (d *DistributedHooks) Stop(params ...interface{}) error {
	return d.worker.StopNode()

}

// Error 用于处理分布式模式下的ERROR事件
func (d *DistributedHooks) Error(params ...interface{}) error {
	return nil
}

// Exit 用于处理分布式模式下的Exit事件
func (d *DistributedHooks) Exit(params ...interface{}) error {
	return d.worker.DelNode()
}

// EventsWatcher 分布式模式下的事件监听器
func (d *DistributedHooks) EventsWatcher(ch chan tegenaria.EventType) error {
	return tegenaria.DefaultWatcher(ch, d)

}

// Exit 用于处理分布式模式下的HEARTBEAT事件
func (d *DistributedHooks) Heartbeat(params ...interface{}) error {
	return d.worker.Heartbeat()
}

// SetCurrentSpider 设置spider实例
func (d *DistributedHooks) SetCurrentSpider(spider tegenaria.SpiderInterface) {
	d.spider = spider
}
