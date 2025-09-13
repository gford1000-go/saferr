package saferr

import (
	"context"
	"fmt"
	"sync"
	"time"
)

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
