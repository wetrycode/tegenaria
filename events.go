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
	"runtime"
)

// EventType 事件类型
type EventType int

const (
	// START 启动
	START EventType = iota
	// HEARTBEAT 心跳
	HEARTBEAT
	// PAUSE 暂停
	PAUSE
	// ERROR 错误
	ERROR
	// EXIT 退出
	EXIT
)

// Hook 事件处理函数类型
type Hook func(params ...interface{}) error

// EventHooksInterface 事件处理函数接口
type EventHooksInterface interface {
	// Start 处理引擎启动事件
	Start(params ...interface{}) error
	// Stop 处理引擎停止事件
	Pause(params ...interface{}) error
	// Error处理错误事件
	Error(params ...interface{}) error
	// Exit 退出引擎事件
	Exit(params ...interface{}) error
	// Heartbeat 心跳检查事件
	Heartbeat(params ...interface{}) error
	// EventsWatcher 事件监听器
	EventsWatcher(ch chan EventType) error

	SetCurrentSpider(spider SpiderInterface)
}
type DefaultHooks struct {
	spider SpiderInterface
}

// NewDefaultHooks 构建新的默认事件监听器
func NewDefaultHooks() *DefaultHooks {
	return &DefaultHooks{}
}

// Start 处理START事件
func (d *DefaultHooks) Start(params ...interface{}) error {
	return nil
}

// Pause 处理STOP事件
func (d *DefaultHooks) Pause(params ...interface{}) error {
	return nil
}

// Error 处理ERROR事件
func (d *DefaultHooks) Error(params ...interface{}) error {
	return nil
}

// Exit 处理EXIT事件
func (d *DefaultHooks) Exit(params ...interface{}) error {
	return nil
}

// Heartbeat 处理HEARTBEAT事件
func (d *DefaultHooks) Heartbeat(params ...interface{}) error {
	return nil
}

// DefaultWatcher 默认的事件监听器
// ch 用于接收事件
// hooker 事件处理实例化接口，比如DefaultHooks
func DefaultWatcher(ch chan EventType, hooker EventHooksInterface) error {
	for {
		select {
		case event := <-ch:
			switch event {
			case START:
				err := hooker.Start()
				if err != nil {
					return err
				}
			case PAUSE:
				err := hooker.Pause()
				if err != nil {
					return err
				}
			case ERROR:
				err := hooker.Error()
				if err != nil {
					return nil
				}
			case HEARTBEAT:
				err := hooker.Heartbeat()
				if err != nil {
					return err
				}
			case EXIT:
				err := hooker.Exit()
				if err != nil {
					return err
				}
				return nil
			default:

			}
		default:

		}
		runtime.Gosched()
	}

}

// EventsWatcher DefualtHooks 的事件监听器
func (d *DefaultHooks) EventsWatcher(ch chan EventType) error {
	return DefaultWatcher(ch, d)

}

func (d *DefaultHooks) SetCurrentSpider(spider SpiderInterface) {
	d.spider = spider
}
