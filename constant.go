// Copyright (c) 2023 wetrycode
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package tegenaria

// StatusType 当前引擎的状态
type StatusType uint

const (
	// ON_START 启动状态
	ON_START StatusType = iota
	// ON_STOP 停止状态
	ON_STOP
	// ON_PAUSE 暂停状态
	ON_PAUSE
)

// GetTypeName 获取引擎状态的字符串形式
func (p StatusType) GetTypeName() string {
	switch p {
	case ON_START:
		return "running"
	case ON_STOP:
		return "stop"
	case ON_PAUSE:
		return "pause"
	}
	return "unknown"
}
