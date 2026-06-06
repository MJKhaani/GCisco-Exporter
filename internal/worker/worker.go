package worker

import (
	"context"
	"sync"
)

type Pool struct {
	workers int
	jobs    chan func()
	wg      sync.WaitGroup
	mu      sync.Mutex
	ctx     context.Context
}

func New(workers int, cfg interface{}) *Pool {
	p := &Pool{
		workers: workers,
		jobs:    make(chan func(), workers*2),
	}

	for i := 0; i < workers; i++ {
		go p.worker()
	}

	return p
}

func (p *Pool) worker() {
	for job := range p.jobs {
		job()
		p.wg.Done()
	}
}

func (p *Pool) Submit(ctx context.Context, job func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.wg.Add(1)
	p.jobs <- job
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) Stop() {
	close(p.jobs)
}
