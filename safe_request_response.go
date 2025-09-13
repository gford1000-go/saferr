package saferr

import (
	"context"
)

// New returns a Requestor and Responder pair, that have a dedicated communication channel
// that passes requests containing *T and responses containing *U.
func New[T any, U any](ctx context.Context, opts ...func(*Options)) (Requestor[T, U], Responder[T, U]) {
	var o Options = defaults
	for _, f := range opts {
		f(&o)
	}

	ch := make(chan *req[T, U], o.ChanSize)
	done := make(chan struct{})

	return &requestor[T, U]{
			commsBase: commsBase[T, U]{
				ch:      ch,
				done:    done,
				ctx:     ctx,
				timeout: o.RequestorTimeout,
			},
			pool: newReqPool[T, U](),
		}, &responder[T, U]{
			commsBase: commsBase[T, U]{
				ch:      ch,
				done:    done,
				ctx:     ctx,
				timeout: o.ResponderTimeout,
			},
			pool:                     newRespPool[U](),
			requestorGoneAwayTimeout: o.RequestorGoneAwayTimeout,
		}
}
