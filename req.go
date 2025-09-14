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
	Get func(t *T) *req[T, U]
	Put func(x *req[T, U])
}

func newReqPool[T any, U any](c *correlatedChanPool[U], incrementer func() uint64) *reqPool[T, U] {
	p := sync.Pool{
		New: func() any {
			var v any = new(req[T, U])
			return v
		},
	}

	// Initialise with the data for the request, plus a unique id for that request
	getter := func(t *T) *req[T, U] {
		r := p.Get().(*req[T, U])

		r.id = incrementer()
		r.data = t
		r.c = c.Get(r.id)

		return r
	}

	// Ensure reset of instance (and chan return to its pool) before handing the instance back to the pool
	putter := func(x *req[T, U]) {
		c.Put(x.c)
		x.data, x.c, x.id = nil, nil, 0
		p.Put(x)
	}

	return &reqPool[T, U]{
		Get: getter,
		Put: putter,
	}
}
