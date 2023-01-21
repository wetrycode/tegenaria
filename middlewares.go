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

// MiddlewaresInterface 下载中间件的接口用于处理进入下载器之前的request对象
// 和下载之后的response
type MiddlewaresInterface interface {
	// GetPriority 获取优先级，数字越小优先级越高
	GetPriority() int

    // ProcessRequest 处理request请求对象
    // 此处用于增加请求头
    // 按优先级执行
	ProcessRequest(ctx *Context) error

	// ProcessResponse 用于处理请求成功之后的response
    // 执行顺序你优先级，及优先级越高执行顺序越晚
	ProcessResponse(ctx *Context, req chan<- *Context) error

	// GetName 获取中间件的名称
	GetName() string
}
// ProcessResponse 处理下载之后的response函数
type ProcessResponse func(ctx *Context) error
type MiddlewaresBase struct {
	Priority int
}
// Middlewares 下载中间件队列
type Middlewares []MiddlewaresInterface
// 实现sort接口
func (p Middlewares) Len() int           { return len(p) }
func (p Middlewares) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Middlewares) Less(i, j int) bool { return p[i].GetPriority() < p[j].GetPriority() }
