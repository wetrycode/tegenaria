package tegenaria

import (
	"encoding/json"
	"sync/atomic"
)

type Statistic struct {
	ItemScraped  uint64 `json:"ItemScraped"`
	RequestSent  uint64 `json:"RequestSent"`
	DownloadFail uint64 `json:"DownloadFail"`
	ErrorCount   uint64 `json:"ErrorCount"`
}

func NewStatistic() *Statistic {
	return &Statistic{
		ItemScraped:  0,
		RequestSent:  0,
		DownloadFail: 0,
		ErrorCount:   0,
	}
}

func (s *Statistic) IncrItemScraped() {
	atomic.AddUint64(&s.ItemScraped, 1)
}

func (s *Statistic) IncrRequestSent() {
	atomic.AddUint64(&s.RequestSent, 1)
}
func (s *Statistic) IncrDownloadFail() {
	atomic.AddUint64(&s.DownloadFail, 1)
}

func (s *Statistic) IncrErrorCount() {
	atomic.AddUint64(&s.ErrorCount, 1)
}

func (s *Statistic)OutputStats(){
	jsonBytes, _ := json.Marshal(s)
	engineLog.Infof("Statistic:%s",string(jsonBytes))
}
