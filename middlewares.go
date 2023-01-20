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
