package saferr

import (
	"context"
	"fmt"
	"time"
)

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
