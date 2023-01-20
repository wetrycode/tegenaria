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
