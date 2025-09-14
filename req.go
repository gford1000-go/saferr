package saferr

import (
	"math"
	"sync"
)

func getIncrementer() func() uint64 {
	var counter uint64 = 1 // So that 0 indicates req.id is unset
	var counterLck sync.Mutex

	return func() uint64 {
		counterLck.Lock()
		defer counterLck.Unlock()

		// Highly unlikely to wrap but protect just in case
		if counter == math.MaxUint64 {
			counter = 0
		}
		counter++
		return counter
	}
}

type req[T any, U any] struct {
	c    *correlatedChan[U]
	data *T
	id   uint64
}

// Since we expect a lot of traffic between Requestors and Responders,
// use pools to minimise object creation
type reqPool[T any, U any] struct {
	incrementer func() uint64
	chanPool    *correlatedChanPool[U]
	pool        sync.Pool
}

// Get is a slight variance to idiomatic Go for sync.Pool,
// returning fully initialised req[T, U] instance
func (p *reqPool[T, U]) Get(t *T) *req[T, U] {
	r := p.pool.Get().(*req[T, U])

	// Initialise with the data for the request, plus a unique id for that request
	r.id = p.incrementer()
	r.data = t
	r.c = p.chanPool.Get(r.id)
	return r
}

// Put ensures that the returned instance is reset before reuse
func (p *reqPool[T, U]) Put(x *req[T, U]) {
	// Ensure reset of instance (including chan closing) before handing back to the pool
	p.chanPool.Put(x.c)
	x.data, x.c, x.id = nil, nil, 0
	p.pool.Put(x)
}

func newReqPool[T any, U any](c *correlatedChanPool[U], incrementer func() uint64) *reqPool[T, U] {
	return &reqPool[T, U]{
		incrementer: incrementer,
		chanPool:    c,
		pool: sync.Pool{
			New: func() any {
				var v any = new(req[T, U])
				return v
			},
		},
	}
}
