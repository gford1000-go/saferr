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

// Go provides a simplified pattern for Requestor / Responder, creating and managing the goroutine in which
// the Responder handles requests, and is the preferred way to use this pattern.
// ListenAndServe() will call handler for each request it receives.
// Options allow hooks to be set for PreStart, to initialise with custom code; PostListen, to perform
// custom processing when ListenAndServe() times out between requests; PostEnd, to perform custom cleanup
func Go[T any, U any](ctx context.Context, handler func(context.Context, *T) (*U, error), opts ...func(*Options)) Requestor[T, U] {
	requestor, receiver := New[T, U](ctx, opts...)

	o := defaults
	for _, opt := range opts {
		opt(&o)
	}

	go func() {
		// Always ensure receiver resources are tidied up, and requestor knows it is not handling requests
		defer receiver.Close()

		var err error
		defer func() {
			if o.GoPostEnd != nil {
				o.GoPostEnd(err)
			}
		}()

		ctxLS := ctx
		if o.GoPreStart != nil {
			ctxLS, err = o.GoPreStart(ctx)
		}
		if err != nil {
			return
		}

		for err == nil {
			err = receiver.ListenAndHandle(ctxLS, handler)
			if err == nil && o.GoPostListen != nil {
				if err = o.GoPostListen(ctxLS); err != nil {
					return
				}
			}
		}
	}()

	return requestor
}
