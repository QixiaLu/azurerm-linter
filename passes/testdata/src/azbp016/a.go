package azbp016

import (
	"context"
	"time"

	"testdata/src/mockpkg/pluginsdk"
)

func invalidStateChangeConf() {
	ctx := context.Background()
	stateConf := &pluginsdk.StateChangeConf{
		Pending: []string{"Creating"},
		Target:  []string{"Created"},
		Timeout: 10 * time.Minute,
	}
	_, _ = stateConf.WaitForStateContext(ctx) // want `AZBP016`
}

func invalidWaitForStateContextOnly() {
	ctx := context.Background()
	conf := &pluginsdk.StateChangeConf{}
	_, _ = conf.WaitForStateContext(ctx) // want `AZBP016`
}

func invalidStateChangeConfNoPointer() {
	ctx := context.Background()
	stateConf := pluginsdk.StateChangeConf{
		Pending: []string{"Deleting"},
		Target:  []string{"Deleted"},
		Timeout: 5 * time.Minute,
	}
	_, _ = stateConf.WaitForStateContext(ctx) // want `AZBP016`
}

// Custom poller pattern - no warnings
type MyCustomPoller struct{}

func (p *MyCustomPoller) Poll(ctx context.Context) error {
	return nil
}

func validCustomPoller() {
	p := &MyCustomPoller{}
	_ = p.Poll(context.Background())
}

type CustomWaiter struct{}

func (w *CustomWaiter) WaitForStateContext(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func validDifferentReceiver() {
	ctx := context.Background()
	w := &CustomWaiter{}
	_, _ = w.WaitForStateContext(ctx) // want `AZBP016`
}
