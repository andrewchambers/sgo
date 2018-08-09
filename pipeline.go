package sgo

import (
	"sync"
)

// An ordered pipeline where work in performed with limited concurrency, but results are
// are collected in the same order they are scheduled.
type OrderedPipeline struct {
	input        chan orderedPipelineTask
	collectCalls chan func()

	wg sync.WaitGroup

	lock            sync.Mutex
	nextTaskNotify  chan struct{}
	dueResultsCount int64
}

type orderedPipelineTask struct {
	ConcurrentFunc  func() interface{}
	CollectResult func(interface{})
	myTurn        chan struct{}
	nextTurn      chan struct{}
}

// Create a pipeline that provides backpressure when its work buffer is
// and executes a 'concurrency' number of times concurrently.
func NewOrderedPipeline(bufferSize, concurrency int) *OrderedPipeline {
	if bufferSize <= 0 {
		panic("bad buffer size")
	}

	if concurrency <= 0 {
		panic("bad concurrency")
	}

	p := &OrderedPipeline{
		input:           make(chan orderedPipelineTask, bufferSize),
		collectCalls:    make(chan func()),
		nextTaskNotify:  make(chan struct{}, 1),
		dueResultsCount: 0,
	}

	// The first task can report immediately...
	p.nextTaskNotify <- struct{}{}

	p.wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer p.wg.Done()
			for {
				t, ok := <-p.input
				if !ok {
					return
				}

				r := t.ConcurrentFunc()

				<-t.myTurn
				collect := t.CollectResult
				p.collectCalls <- func() {
					collect(r)
				}
				t.nextTurn <- struct{}{}
			}
		}()
	}

	return p
}

// schedule concurrentFunc in the work pool, possibly collecting a previous result from the pipeline
// if there is no space for concurrentFunc. The passed collectResult will be called by a
// goroutine when it either adds a task and there is no room in the task buffer, or the
// pipeline is closed.
// Collection happens in the calling goroutine.
func (p *OrderedPipeline) AddTask(concurrentFunc func() interface{}, collectResult func(interface{})) {
	p.lock.Lock()
	defer p.lock.Unlock()

	t := orderedPipelineTask{
		ConcurrentFunc:  concurrentFunc,
		CollectResult: collectResult,
	}

	t.myTurn = p.nextTaskNotify
	t.nextTurn = make(chan struct{}, 1)
	p.nextTaskNotify = t.nextTurn

	for {
		select {
		case p.input <- t:
			p.dueResultsCount += 1
			return
		default:
		}
		collect := <-p.collectCalls
		collect()
		p.dueResultsCount -= 1
	}
}

// Collect all pending results from the pipeline and close worker threads.
// Collection happens in the calling goroutine.
func (p *OrderedPipeline) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	for p.dueResultsCount != 0 {
		collect := <-p.collectCalls
		collect()
		p.dueResultsCount -= 1
	}

	close(p.input)
	p.wg.Wait()
}


// A pipeline where work in performed with limited concurrency, and results are
// collected in the order they are ready.
type Pipeline struct {
	input        chan pipelineTask
	collectCalls chan func()

	wg sync.WaitGroup

	lock            sync.Mutex
	dueResultsCount int64
}

type pipelineTask struct {
	ConcurrentFunc  func() interface{}
	CollectResult func(interface{})
}

// Create a pipeline that provides backpressure when its work buffer is
// and executes a 'concurrency' number of times concurrently.
func NewPipeline(bufferSize, concurrency int) *Pipeline {
	if bufferSize <= 0 {
		panic("bad buffer size")
	}

	if concurrency <= 0 {
		panic("bad concurrency size")
	}

	p := &Pipeline{
		input:           make(chan pipelineTask, bufferSize),
		collectCalls:    make(chan func()),
		dueResultsCount: 0,
	}

	p.wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer p.wg.Done()
			for {
				t, ok := <-p.input
				if !ok {
					return
				}

				r := t.ConcurrentFunc()

				collect := t.CollectResult
				p.collectCalls <- func() {
					collect(r)
				}
			}
		}()
	}

	return p
}

// schedule concurrentFunc in the work pool, possibly collecting a previous result from the pipeline
// if there is no space for concurrentFunc. The passed collectResult will be called by a
// goroutine when it either adds a task and there is no room in the task buffer, or the
// pipeline is closed.
// Collection happens in the calling goroutine.
func (p *Pipeline) AddTask(concurrentFunc func() interface{}, collectResult func(interface{})) {
	p.lock.Lock()
	defer p.lock.Unlock()

	t := pipelineTask{
		ConcurrentFunc:  concurrentFunc,
		CollectResult: collectResult,
	}

	for {
		select {
		case p.input <- t:
			p.dueResultsCount += 1
			return
		default:
		}
		collect := <-p.collectCalls
		collect()
		p.dueResultsCount -= 1
	}
}

// Collect all pending results from the pipeline and close worker threads.
// Collection happens in the calling goroutine.
func (p *Pipeline) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	for p.dueResultsCount != 0 {
		collect := <-p.collectCalls
		collect()
		p.dueResultsCount -= 1
	}

	close(p.input)
	p.wg.Wait()
}
