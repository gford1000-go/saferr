package saferr

import "sync"

type resp[U any] struct {
	data *U
	err  error
	pool *respPool[U]
}

// close will trigger this instance to be reset and added back to the pool it came from (if known)
func (r *resp[U]) close() {
	if r.pool != nil {
		r.pool.Put(r)
	}
}

// Since we expect a lot of traffic between Requestors and Responders,
// use pools to minimise object creation
type respPool[U any] struct {
	pool sync.Pool
}

// Get is a slight variance to idiomatic Go for sync.Pool,
// returning fully initialised resp[U] instance
func (p *respPool[U]) Get(u *U, err error) *resp[U] {
	r := p.pool.Get().(*resp[U])

	r.data, r.err, r.pool = u, err, p
	return r
}

// Put ensures that the returned instance is reset before reuse
func (p *respPool[U]) Put(x *resp[U]) {
	x.data, x.err, x.pool = nil, nil, nil
	p.pool.Put(x)
}

func newRespPool[U any]() *respPool[U] {
	return &respPool[U]{
		pool: sync.Pool{
			New: func() any {
				var v any = new(resp[U])
				return v
			},
		},
	}
}
