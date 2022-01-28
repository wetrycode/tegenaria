package tegenaria

import "sync"

var onceInit sync.Once

func init() {
	onceInit.Do(func() {
		initSettings()
		initLog()
	})

}
