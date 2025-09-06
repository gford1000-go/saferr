package saferr

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type resp[U any] struct {
	data *U
	err  error
}

type req[T any, U any] struct {
	ch   chan *resp[U]
	data *T
}

type commsBase[T any, U any] struct {
	ch      chan *req[T, U]
	done    chan struct{}
	closed  atomic.Bool
	ctx     context.Context
	timeout time.Duration
}

func (c *commsBase[T, U]) isClosed() bool {
	return c.closed.Load()
}

func (c *commsBase[T, U]) setClosed() {
	c.closed.Store(true)
}

type requestor[T any, U any] struct {
	commsBase[T, U]
}

func (r *requestor[T, U]) Send(ctx context.Context, t *T) (*U, error) {

	select {
	case <-r.ctx.Done():
		r.setClosed()
		return nil, ErrContextCompleted
	case <-ctx.Done():
		r.setClosed()
		return nil, ErrContextCompleted
	default:
		if r.isClosed() {
			return nil, ErrRequestorIsClosed
		}
		return r.attemptSend(t)
	}
}

func (r *requestor[T, U]) attemptSend(t *T) (u *U, err error) {
	defer func() {
		if rc := recover(); rc != nil {
			u = nil
			err = fmt.Errorf("%w: %v", ErrUncaughtSendPanic, rc)
		}
	}()

	ch := make(chan *resp[U])

	// Deferred close of the resp chan means that there is a possibility via
	// timeout that the Responder attempts to send to this chan after it has closed.
	// Hence the trap for panic in Responder.sendResp(), which discards the resp.
	defer close(ch)

	// The done chan read will return a zero when it is closed, signalling
	// that the Responder has closed and will not reply
	// Note that there is a small possibility that the r.ch will block if its buffer is full,
	// i.e. if the Requestor is making many concurrent requests.
	// So make sure that this is ChanSize is set to a reasonable level (default = 100)
	select {
	case <-r.done:
		r.setClosed()
		return nil, ErrCommsChannelIsClosed
	case r.ch <- &req[T, U]{
		data: t,
		ch:   ch,
	}:
	}

	// If here, then either:
	// (a) the Responder exists and a response will be provided
	// (b) the Responder has closed, but the request was sent before the done chan was closed,
	//     in which case the Requestor will have to wait until their request is timed out
	select {
	case <-time.After(r.timeout):
		return nil, ErrSendTimeout
	case resp := <-ch:
		return resp.data, resp.err
	}
}

type responder[T any, U any] struct {
	commsBase[T, U]
	requestorGoneAwayTimeout time.Duration
	hasGoneAway              time.Time
	once                     sync.Once
	initialise               sync.Once
}

func (r *responder[T, U]) ListenAndHandle(ctx context.Context, requestHandler Handler[T, U]) error {
	// Initialise the hasGoneAway time the first time ListenAndHandle is called
	// allowing for other work to be done in the goroutine before the first request is handled
	r.initialise.Do(func() {
		r.hasGoneAway = time.Now().Add(r.requestorGoneAwayTimeout)
	})

	// Note: The caller is expected to loop on ListenAndHandle() from a single goroutine only
	//       This is NOT thread safe if called from multiple goroutines, nor is request sequencing guaranteed
	select {
	case <-time.After(r.timeout):
		if time.Now().After(r.hasGoneAway) {
			r.setClosed()
			return ErrRequestorGoneAway
		}
		return nil
	case <-r.ctx.Done():
		r.setClosed()
		return ErrContextCompleted
	case <-ctx.Done():
		r.setClosed()
		return ErrContextCompleted
	case req, ok := <-r.ch:
		if !ok {
			return ErrCommsChannelIsClosed
		}
		if r.isClosed() {
			req.ch <- &resp[U]{
				err: ErrResponderIsClosed,
			}
			return nil
		}
		r.hasGoneAway = time.Now().Add(r.requestorGoneAwayTimeout) // Reset the gone away timer
		return r.handle(ctx, requestHandler, req)
	}
}

func (r *responder[T, U]) Close() {
	r.setClosed()
	r.once.Do(func() {
		close(r.done)
		// close(r.ch) // Don't close the data channel - let this be garbage collected later
	})
}

func (r *responder[T, U]) sendResp(ch chan *resp[U], resp *resp[U]) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("caught panic - dropping resp: %v", r)
		}
	}()

	ch <- resp
}

func (r *responder[T, U]) handle(ctx context.Context, h Handler[T, U], req *req[T, U]) error {
	defer func() {
		if rc := recover(); rc != nil {
			r.sendResp(req.ch, &resp[U]{
				err: fmt.Errorf("%w: %v", ErrUncaughtHandlerPanic, rc),
			})
		}
	}()

	u, err := h(ctx, req.data)

	r.sendResp(req.ch, &resp[U]{
		data: u,
		err:  err,
	})

	return nil
}

// Options holds the available options that can be set in the NewComms call
type Options struct {
	// RequestorTimeout is the timeout for a Send
	RequestorTimeout time.Duration
	// RequestorGoneAwayTimeout is the timeout between Requestor requests
	RequestorGoneAwayTimeout time.Duration
	// ResponderTimeout is the timeout for ListenAndHandle to wait for requests before returning
	ResponderTimeout time.Duration
	// ChanSize sets the size of the communication buffer
	ChanSize int
}

var defaults Options = Options{
	RequestorTimeout:         30 * time.Second,
	RequestorGoneAwayTimeout: 2 * time.Minute,
	ResponderTimeout:         1 * time.Second, // Expect the Responder to spin on a for { } loop
	ChanSize:                 100,
}

// WithChanSize sets the size of the communication buffer
func WithChanSize(size int) func(*Options) {
	return func(o *Options) {
		if size > 0 {
			o.ChanSize = size
		}
	}
}

// WithRequestorTimeout sets the timeout for a given Send
func WithRequestorTimeout(d time.Duration) func(*Options) {
	return func(o *Options) {
		if d > 0 {
			o.RequestorTimeout = d
		}
	}
}

// WithResponderTimeout sets the timeout for the Responder to wait for a request in a ListenAndHandle() before returning
func WithResponderTimeout(d time.Duration) func(*Options) {
	return func(o *Options) {
		if d > 0 {
			o.ResponderTimeout = d
		}
	}
}

// WithRequestorGoneWayTimeout sets the timeout before the Responder assumes the Requestor has disappeared,
// and so shuts itself down
func WithRequestorGoneWayTimeout(d time.Duration) func(*Options) {
	return func(o *Options) {
		if d > 0 {
			o.RequestorGoneAwayTimeout = d
		}
	}
}

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
		}, &responder[T, U]{
			commsBase: commsBase[T, U]{
				ch:      ch,
				done:    done,
				ctx:     ctx,
				timeout: o.ResponderTimeout,
			},
			requestorGoneAwayTimeout: o.RequestorGoneAwayTimeout,
		}
}
