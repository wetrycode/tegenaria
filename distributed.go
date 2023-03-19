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

// DistributedWorkerInterface 分布式组件接口
type DistributedWorkerInterface interface {

	// AddNode 新增一个节点
	AddNode() error
	// DelNode 删除当前的节点
	DelNode() error
	// PauseNode 停止当前的节点
	PauseNode() error
	// Heartbeat 心跳
	Heartbeat() error
	// CheckAllNodesStop 检查所有的节点是否都已经停止
	CheckAllNodesStop() (bool, error)
	// CheckMasterLive 检测主节点是否还在线
	CheckMasterLive() (bool, error)
	// SetMaster 是否将当前的节点设置为主节点
	SetMaster(flag bool)
	// SetCurrentSpider 设置当前的spider
	SetCurrentSpider(spider SpiderInterface)
	// GetWorkerID 当前工作节点的id
	GetWorkerID() string
	// IsMaster 是否是主节点
	IsMaster() bool
}
