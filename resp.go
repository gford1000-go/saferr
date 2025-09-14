package saferr

import (
	"sync"
)

type resp[U any] struct {
	id       uint64
	data     *U
	err      error
	returner func(x *resp[U])
}

// close will trigger this instance to be reset and added back to the pool it came from (if known)
func (r *resp[U]) close() {
	if r.returner != nil {
		r.returner(r)
	}
}

// Since we expect a lot of traffic between Requestors and Responders,
// use pools to minimise object creation
type respPool[U any] struct {
	Get func(id uint64, u *U, err error) *resp[U]
	Put func(x *resp[U])
}

func newRespPool[U any]() *respPool[U] {

	p := sync.Pool{
		New: func() any {
			var v any = new(resp[U])
			return v
		},
	}

	putter := func(x *resp[U]) {
		x.data, x.err, x.returner, x.id = nil, nil, nil, 0
		p.Put(x)
	}

	getter := func(id uint64, u *U, err error) *resp[U] {
		r := p.Get().(*resp[U])

		r.data, r.err, r.returner, r.id = u, err, putter, id
		return r
	}

	return &respPool[U]{
		Get: getter,
		Put: putter,
	}
}
