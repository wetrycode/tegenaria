package tegenaria

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-kiss/monkey"
)

func TestEventWatcher(t *testing.T) {
	config := NewDistributedWorkerConfig("", "", 0)
	mockRedis := miniredis.RunT(t)
	defer mockRedis.Close()
	worker := NewDistributedWorker(mockRedis.Addr(), config)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		engine := newTestEngine("testDistributedSpider", EngineWithDistributedWorker(worker))
		engine.Start("testDistributedSpider")
	}()
	time.Sleep(500 * time.Millisecond)
	mockRedis.FastForward(5 * time.Second)

}

func TestEventWatcherWithError(t *testing.T) {
	monkey.Patch((*DistributedHooks).Start,func(_ *DistributedHooks,_ ...interface{}) error {
		return fmt.Errorf("start error")
	
	})
	monkey.Patch((*DistributedHooks).Stop,func(_ *DistributedHooks,_ ...interface{}) error {
		return fmt.Errorf("stop error")
	
	})
	monkey.Patch((*DistributedHooks).Error,func(_ *DistributedHooks,_ ...interface{}) error {
		return fmt.Errorf("error")
	
	})
	monkey.Patch((*DistributedHooks).Heartbeat,func(_ *DistributedHooks,_ ...interface{}) error {
		return fmt.Errorf("heartbeat error")
	
	})

	monkey.Patch((*DistributedHooks).Exit,func(_ *DistributedHooks,_ ...interface{}) error {
		return fmt.Errorf("exit error")
	
	})

	config := NewDistributedWorkerConfig("", "", 0)
	mockRedis := miniredis.RunT(t)
	defer mockRedis.Close()
	worker := NewDistributedWorker(mockRedis.Addr(), config)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		engine := newTestEngine("testDistributedSpider", EngineWithDistributedWorker(worker))
		engine.Start("testDistributedSpider")
	}()

	time.Sleep(500 * time.Millisecond)
	mockRedis.FastForward(5 * time.Second)
	wg.Wait()
	monkey.Unpatch((*DistributedHooks).Start)
	monkey.Unpatch((*DistributedHooks).Stop)
	monkey.Unpatch((*DistributedHooks).Error)
	monkey.Unpatch((*DistributedHooks).Heartbeat)
	monkey.Unpatch((*DistributedHooks).Exit)




}
