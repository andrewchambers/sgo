package sgo

import (
	"context"
	"sync"
)

// A process group is a set of processes that run
// together and terminate together, the purpose
// is to help clean startup and shutdown of groups
// of goroutines.
type ProcessGroup struct {
	ctx context.Context
	wg  sync.WaitGroup
}

// Create a process group that shares a context
func NewProcessGroup(ctx context.Context) *ProcessGroup {
	pg := &ProcessGroup{
		ctx: ctx,
	}

	return pg
}

// Return a chanel that signals when all started goroutines
// are complete, be sure to get this channel after all
// goroutines are started.
func (pg *ProcessGroup) Done() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		pg.wg.Wait()
		close(done)
	}()

	return done
}

func (pg *ProcessGroup) Go(proc func(ctx context.Context)) {
	pg.wg.Add(1)
	go func() {
		defer pg.wg.Done()
		proc(pg.ctx)
	}()
}
