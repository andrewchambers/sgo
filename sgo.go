package sgo

import (
	"context"
	"runtime/debug"
	"sync"
)

type ErrAlreadyClosed struct {
	Reason interface{}
}

func (e *ErrAlreadyClosed) Error() string {
	return "process group was already closed."
}

type ReasonChildPanic struct {
	Panic      interface{}
	StackTrace []byte
}

type ReasonClosed struct{}

type ReasonParentContextDone struct{}

// A collection of goroutines that are started
// and stopped together.
type ProcessGroup struct {
	ctx          context.Context
	cancel       func(reason interface{})
	wg           sync.WaitGroup
	lock         sync.Mutex
	closed       bool
	cancelReason interface{}
}

func NewProcessGroup(ctx context.Context) *ProcessGroup {
	ctxWithCancel, ctxCancel := context.WithCancel(ctx)
	once := new(sync.Once)
	pg := &ProcessGroup{}
	pg.ctx = ctxWithCancel
	pg.cancel = func(reason interface{}) {
		once.Do(func() {
			pg.lock.Lock()
			defer pg.lock.Unlock()
			ctxCancel()
			pg.cancelReason = reason
			pg.closed = true
		})
	}

	err := pg.Go(func(ctx context.Context) {
		<-ctx.Done()
		// If our context is cancelled, this stops
		// new child goroutines from starting.
		pg.cancel(&ReasonParentContextDone{})
	})
	if err != nil {
		panic(err)
	}

	return pg
}

// Launch a goroutine owned by this process group.
// If any child goroutine panics, the process group is
// has it's context cancelled.
func (pg *ProcessGroup) Go(f func(context.Context)) error {
	pg.lock.Lock()
	defer pg.lock.Unlock()

	if pg.closed {
		return &ErrAlreadyClosed{
			Reason: pg.cancelReason,
		}
	}

	go func() {
		pg.wg.Add(1)
		defer func() {
			r := recover()
			if r != nil {
				pg.cancel(&ReasonChildPanic{
					Panic:      r,
					StackTrace: debug.Stack(),
				})
			}
			pg.wg.Done()

			if r != nil {
				panic(r)
			}
		}()
		f(pg.ctx)
	}()

	return nil
}

func (pg *ProcessGroup) Wait(ctx context.Context) {
	pg.wg.Wait()
}

func (pg *ProcessGroup) Close(ctx context.Context) error {
	pg.cancel(&ReasonClosed{})
	pg.Wait(ctx)
	return nil
}
