package saferr

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

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
	pool *reqPool[T, U]
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

	// Get an initialised req[T, U] from the pool to reduce allocations
	req := r.pool.Get(t)

	// Deferred return of the req[T, U] (req) instance means that after a timeout
	// for the Requestor, the Responder may attempt to send to its embedded chan after it has closed.
	// Hence the trap for panic in Responder.sendResp(), which simply discards the resp.
	defer r.pool.Put(req)

	retry := true
	attempts := 0
	maxAttempts := 3
	for retry {
		var err error
		submitTimer := acquireTimer(100 * time.Microsecond)

		select {
		case <-r.done:
			// The done chan read will return a zero when it is closed, signalling
			// that the Responder has closed and will not reply
			r.setClosed()
			err = ErrCommsChannelIsClosed
		case r.ch <- req:
			retry = false // only put the req onto the r.ch once
		case <-submitTimer.C:
			// There is a possibility that a large number of concurrent Send() calls
			// could fill up r.ch before the done chan is closed.
			// This could mean that a Send() could block indefinitely trying to write to r.ch
			// even though the Responder has closed.
			// Retrying should detect done has closed, and so return an error
			//
			// Alternatively, the Responder could be very slow to respond,
			// and so the Send() could be blocked trying to write to r.ch
			// even though the Responder is taking a long time to respond.
			// Hence don't call setClosed() as this might be a temporary condition
			attempts++
			if attempts >= maxAttempts {
				err = ErrUnableToSendRequest
			}
		}

		releaseTimer(submitTimer)

		if err != nil {
			return nil, err
		}
	}

	// If here, then either:
	// (a) the Responder exists and a response will be provided
	// (b) the Responder has closed, but the request was sent before the done chan was closed,
	//     in which case the Requestor will now have to wait until their request is timed out
	//
	// Need to ensure ghost messages are captured and discarded.
	// Also only allow the r.timeout duration to receive the correct resp for the req.
	retry = true
	responseTimer := acquireTimer(r.timeout)
	defer releaseTimer(responseTimer)

	var resp *resp[U]
	for retry {
		select {
		case <-responseTimer.C:
			return nil, ErrSendTimeout
		case resp = <-req.ch:
			if resp.id != req.id {
				resp.close() // Not interested in this one; discard as Requestor timed out
			} else {
				retry = false // Have matched response
			}
		}
	}

	// Have a valid resp if we reach here
	defer resp.close()
	return resp.data, resp.err
}

type responder[T any, U any] struct {
	commsBase[T, U]
	requestorGoneAwayTimeout time.Duration
	hasGoneAway              time.Time
	once                     sync.Once
	initialise               sync.Once
	pool                     *respPool[U]
}

func (r *responder[T, U]) ListenAndHandle(ctx context.Context, requestHandler Handler[T, U]) error {
	// Initialise the hasGoneAway time the first time ListenAndHandle is called
	// allowing for other work to be done in the goroutine before the first request is handled
	r.initialise.Do(func() {
		r.hasGoneAway = time.Now().Add(r.requestorGoneAwayTimeout)
	})

	listenTimer := acquireTimer(r.timeout)
	defer releaseTimer(listenTimer)

	// Note: The caller is expected to loop on ListenAndHandle() from a single goroutine only
	//       This is NOT thread safe if called from multiple goroutines, nor is request sequencing guaranteed
	select {
	case <-listenTimer.C:
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
			req.ch <- r.pool.Get(req.id, nil, ErrResponderIsClosed)
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
			resp.close()
		}
	}()

	ch <- resp
}

func (r *responder[T, U]) handle(ctx context.Context, h Handler[T, U], req *req[T, U]) error {
	defer func() {
		if rc := recover(); rc != nil {
			r.sendResp(req.ch, r.pool.Get(req.id, nil, fmt.Errorf("%w: %v", ErrUncaughtHandlerPanic, rc)))
		}
	}()

	ch := req.ch

	u, err := h(ctx, req.data)
	resp := r.pool.Get(req.id, u, err)

	r.sendResp(ch, resp)

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
