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
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/shopspring/decimal"
)

// StatsFieldType 统计指标的数据类型
type StatsFieldType string

// codeStatusName http状态码
var codeStatusName = [][]int{{100, 101}, {200, 206}, {300, 307}, {400, 417}, {500, 505}}

const (
	// RequestStats 发起的请求总数
	RequestStats string = "requests"
	// ItemsStats 获取到的items总数
	ItemsStats string = "items"
	// DownloadFailStats 请求失败总数
	DownloadFailStats string = "download_fail"
	// ErrorStats 错误总数
	ErrorStats string = "errors"
)

type RuntimeStatus struct {
	StartAt   int64
	Duration  float64
	StopAt    int64
	RestartAt int64
	// StatusOn 当前引擎的状态
	StatusOn StatusType
}

func NewRuntimeStatus() *RuntimeStatus {
	return &RuntimeStatus{
		StartAt:   0,
		Duration:  0,
		StopAt:    0,
		RestartAt: 0,
		StatusOn:  ON_STOP,
	}
}

// SetStatus 设置引擎状态
// 用于控制引擎的启停
func (r *RuntimeStatus) SetStatus(status StatusType) {
	r.StatusOn = status
}

// GetStatusOn 获取引擎的状态
func (r *RuntimeStatus) GetStatusOn() StatusType {
	return r.StatusOn
}
func (r *RuntimeStatus) SetStartAt(startAt int64) {
	r.StartAt = startAt
}

// GetStartAt 获取引擎启动的时间戳
func (r *RuntimeStatus) GetStartAt() int64 {
	return r.StartAt
}
func (r *RuntimeStatus) SetRestartAt(startAt int64) {
	r.RestartAt = startAt
}

// GetStartAt 获取引擎启动的时间戳
func (r *RuntimeStatus) GetRestartAt() int64 {
	return r.RestartAt
}
func (r *RuntimeStatus) SetStopAt(stopAt int64) {
	r.StopAt = stopAt
}

// GetStopAt 爬虫停止的时间戳
func (r *RuntimeStatus) GetStopAt() int64 {
	return r.StopAt
}
func (r *RuntimeStatus) SetDuration(duration float64) {
	r.Duration = duration
}

// GetDuration 爬虫运行时长
func (r *RuntimeStatus) GetDuration() float64 {
	return decimal.NewFromFloat(r.Duration).Round(2).InexactFloat64()
}

// StatisticInterface 数据统计组件接口
type StatisticInterface interface {
	GetAllStats() map[string]uint64
	Incr(metric string)
	Get(metric string) uint64
	SetCurrentSpider(spider SpiderInterface)
}

// Statistic 数据统计指标
type DefaultStatistic struct {

	// spider 当前正在运行的spider名
	Metrics  map[string]*uint64
	spider   SpiderInterface `json:"-"`
	register sync.Map
}

// NewStatistic 默认统计数据组件构造函数
func NewDefaultStatistic() *DefaultStatistic {
	m := map[string]*uint64{
		RequestStats:      new(uint64),
		DownloadFailStats: new(uint64),
		ItemsStats:        new(uint64),
		ErrorStats:        new(uint64),
	}
	for _, status := range codeStatusName {
		min, max := status[0], status[1]
		for i := min; i <= max; i++ {
			m[strconv.Itoa(i)] = new(uint64)
		}
	}
	for _, v := range m {
		atomic.StoreUint64(v, 0)

	}
	s := &DefaultStatistic{
		Metrics:  m,
		register: sync.Map{},
	}
	return s
}

// SetCurrentSpider 设置当前的spider
func (s *DefaultStatistic) SetCurrentSpider(spider SpiderInterface) {
	s.spider = spider
}

// Incr 新增一个指标值
func (s *DefaultStatistic) Incr(metrics string) {
	atomic.AddUint64(s.Metrics[metrics], 1)
	s.register.Store(metrics, true)
}

// Get 获取某个指标的数值
func (s *DefaultStatistic) Get(metric string) uint64 {
	return atomic.LoadUint64(s.Metrics[metric])
}

// GetAllStats 格式化统计数据
func (s *DefaultStatistic) GetAllStats() map[string]uint64 {
	result := make(map[string]uint64)
	s.register.Range(func(key any, _ any) bool {
		k := key.(string)
		result[k] = s.Get(k)
		return true

	})
	return result
}
