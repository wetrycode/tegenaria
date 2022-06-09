package tegenaria

import (
	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/flow"
)

func initSentinel() {
	err := sentinel.InitDefault()
	if err != nil {
		panic(err)
	}
	if _, err := flow.LoadRules([]*flow.Rule{
		{
			Resource:               "sentinel",
			Threshold:              16,
			StatIntervalInMs:       1000,
			ControlBehavior:        flow.Throttling,
			TokenCalculateStrategy: flow.Direct,
		},
	}); err != nil {
		panic(err)
	}
}
