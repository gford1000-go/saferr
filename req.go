package saferr

import (
	"sync"
)

type req[T any, U any] struct {
	ch   chan *resp[U]
	data *T
}

// Since we expect a lot of traffic between Requestors and Responders,
// use pools to minimise object creation
type reqPool[T any, U any] struct {
	pool sync.Pool
}

// Get is a slight variance to idiomatic Go for sync.Pool,
// returning fully initialised req[T, U] instance
func (p *reqPool[T, U]) Get(t *T) *req[T, U] {
	r := p.pool.Get().(*req[T, U])

	// This is a blocking chan, forcing the Requestor goroutine to wait for its response from Responder
	// Create a new chan each Get(), as this is lightweight and avoids a class of race condition errors
	r.ch = make(chan *resp[U])
	r.data = t
	return r
}

// Put ensures that the returned instance is reset before reuse
func (p *reqPool[T, U]) Put(x *req[T, U]) {
	// Ensure reset of instance (including chan closing) before handing back to the pool
	// Close is invoked to ensure a panic for a late replying Responder - otherwise it could block on the chan
	close(x.ch)
	x.data, x.ch = nil, nil
	p.pool.Put(x)
}

func newReqPool[T any, U any]() *reqPool[T, U] {
	return &reqPool[T, U]{
		pool: sync.Pool{
			New: func() any {
				var v any = new(req[T, U])
				return v
			},
		},
	}
}
