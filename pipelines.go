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

// PipelinesInterface pipeline interface
// pipeline is mainly used for processing item, 
// the engine schedules ProcessItem according to 
// the priority of pipelines from highest to lowest
type PipelinesInterface interface {
	// GetPriority get pipeline Priority
	GetPriority() int
	// ProcessItem item handler
	ProcessItem(spider SpiderInterface, item *ItemMeta) error
}
type PipelinesBase struct {
	Priority int
}

type ItemPipelines []PipelinesInterface

func (p ItemPipelines) Len() int           { return len(p) }
func (p ItemPipelines) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ItemPipelines) Less(i, j int) bool { return p[i].GetPriority() < p[j].GetPriority() }
