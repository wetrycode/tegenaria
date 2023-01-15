package tegenaria

import "runtime"

type EventType int

const (
	START EventType = iota
	HEARTBEAT
	STOP
	ERROR
	EXIT
)

type Hook func(params ...interface{}) error
type EventHooksInterface interface {
	Start(params ...interface{}) error
	Stop(params ...interface{}) error
	Error(params ...interface{}) error
	Exit(params ...interface{}) error
	Heartbeat(params ...interface{}) error

	EventsWatcher(ch chan EventType) GoFunc
}
type DefualtHooks struct {
}

type DistributedHooks struct {
	worker DistributedWorkerInterface
}

func NewDistributedHooks(worker DistributedWorkerInterface) *DistributedHooks {
	return &DistributedHooks{
		worker: worker,
	}

}
func NewDefualtHooks() *DefualtHooks {
	return &DefualtHooks{}
}
func (d *DefualtHooks) Start(params ...interface{}) error {
	return nil
}
func (d *DefualtHooks) Stop(params ...interface{}) error {
	return nil
}
func (d *DefualtHooks) Error(params ...interface{}) error {
	return nil
}
func (d *DefualtHooks) Exit(params ...interface{}) error {
	return nil
}

func (d *DefualtHooks) Heartbeat(params ...interface{}) error {
	return nil
}

func eventsWatcher(ch chan EventType, hooker EventHooksInterface) {

	for {
		select {
		case event := <-ch:
			switch event {
			case START:
				err := hooker.Start()
				if err != nil {
					engineLog.Errorf("start event error %s", err.Error())
				}
			case STOP:
				err := hooker.Stop()
				if err != nil {
					engineLog.Errorf("stop event error %s", err.Error())

				}
			case ERROR:
				err := hooker.Error()
				if err != nil {
					engineLog.Errorf("error event error %s", err.Error())
				}
			case HEARTBEAT:
				err := hooker.Heartbeat()
				if err != nil {
					engineLog.Errorf("error event error %s", err.Error())

				}
			case EXIT:
				err := hooker.Exit()
				if err != nil {
					engineLog.Errorf("error event error %s", err.Error())

				}
				return
			default:

			}
		default:

		}
		runtime.Gosched()
	}

}
func (d *DefualtHooks) EventsWatcher(ch chan EventType) GoFunc {
	return func() {
		eventsWatcher(ch, d)

	}
}

func (d *DistributedHooks) Start(params ...interface{}) error {
	return d.worker.AddNode()

}
func (d *DistributedHooks) Stop(params ...interface{}) error {
	return d.worker.StopNode()

}
func (d *DistributedHooks) Error(params ...interface{}) error {
	return nil
}

func (d *DistributedHooks) Exit(params ...interface{}) error {
	return d.worker.DelNode()
}

func (d *DistributedHooks) EventsWatcher(ch chan EventType) GoFunc {
	return func() {
		eventsWatcher(ch, d)

	}
}
func (d *DistributedHooks) Heartbeat(params ...interface{}) error {
	return d.worker.Heartbeat()
}
