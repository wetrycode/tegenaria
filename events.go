package tegenaria

import (
	"runtime"
)

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

	EventsWatcher(ch chan EventType) error
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

func DefaultventsWatcher(ch chan EventType, hooker EventHooksInterface) error {
	for {
		select {
		case event := <-ch:
			switch event {
			case START:
				err := hooker.Start()
				if err != nil {
					return err
				}
			case STOP:
				err := hooker.Stop()
				if err != nil {
					return err

				}
			case ERROR:
				err := hooker.Error()
				if err != nil {
					return nil
				}
			case HEARTBEAT:
				err := hooker.Heartbeat()
				if err != nil {
					return err
				}
			case EXIT:
				err := hooker.Exit()
				if err != nil {
					return err
				}
				return nil
			default:

			}
		default:

		}
		runtime.Gosched()
	}

}
func (d *DefualtHooks) EventsWatcher(ch chan EventType) error {
	return DefaultventsWatcher(ch, d)

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

func (d *DistributedHooks) EventsWatcher(ch chan EventType) error {
	return DefaultventsWatcher(ch, d)

}
func (d *DistributedHooks) Heartbeat(params ...interface{}) error {
	return d.worker.Heartbeat()
}
