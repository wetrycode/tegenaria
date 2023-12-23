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

// PipelinesInterface pipeline 接口
// pipeline 主要用于处理item,例如数据存储、数据清洗
// 将多个pipeline注册到引擎可以实现责任链模式的数据处理
type PipelinesInterface interface {
	// GetPriority 获取当前pipeline的优先级
	GetPriority() int
	// ProcessItem item处理单元
	ProcessItem(spider SpiderInterface, item *ItemMeta) error
}
type PipelinesBase struct {
	Priority int
}

type ItemPipelines []PipelinesInterface

func (p ItemPipelines) Len() int           { return len(p) }
func (p ItemPipelines) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ItemPipelines) Less(i, j int) bool { return p[i].GetPriority() < p[j].GetPriority() }
