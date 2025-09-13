package saferr

import (
	"sync"
	"time"
)

// correlatedChan supports a non-blocking sender and a blocking receiver pattern.
// This allows the sender to continue processing other requests, but allows the receiver to wait on the chan.
// Since the correlatedChan will be pooled, we use id to ensure that the receiver only gets messages that are
// correlated to the expected id of this instance, which is reset on return to the pool.
// This will mean that ghost resp[U] are dropped, which is valid as the correlatedChan instance has been
// recycled through the pool which implies the receiver has gone anyway.
type correlatedChan[U any] struct {
	recCh chan *resp[U]
	dstCh chan *resp[U]
	id    uint64
	lck   sync.Mutex
}

func (c *correlatedChan[U]) getId() uint64 {
	c.lck.Lock()
	defer c.lck.Unlock()
	return c.id
}

func (c *correlatedChan[U]) setId(id uint64) {
	c.lck.Lock()
	defer c.lck.Unlock()
	c.id = id
}

func (c *correlatedChan[U]) send(r *resp[U]) {
	c.recCh <- r // This should be minimally blocking for senders
}

func (c *correlatedChan[U]) getReceiverChan() chan *resp[U] {
	return c.dstCh // Access to blocking chan only for receiveers
}

func newCorrelatedChan[U any](maxRetries int, sendTimeout time.Duration, chanSize int) *correlatedChan[U] {
	c := &correlatedChan[U]{
		recCh: make(chan *resp[U], chanSize), // Non-blocking
		dstCh: make(chan *resp[U]),           // BLOCKING
	}

	go func() {

		forwardedOK := func(r *resp[U], d time.Duration) bool {
			timer := acquireTimer(d)
			defer releaseTimer(timer)

			select {
			case c.dstCh <- r:
				return true
			case <-timer.C:
				return false
			}
		}

		for {
			r := <-c.recCh

			// Processing behaviour of the resp[U] depends on id value of the correlatedChan
			// If 0, then this instancex is back in the pool, so drop the resp[U]
			// Otherwise, only forward if the resp.id matches the id of the correlatedChan; everything else is a ghost resp[U]
			id := c.getId()
			if id == 0 || id != r.id {
				continue
			}

			// Expected id, so attempt to forward to Requestor
			// Timeout and retries ensure that we stop attempting to send if the Requestor has disappeared
			retries := 0
			for !forwardedOK(r, sendTimeout) && retries < maxRetries {
				retries++
			}
		}

	}()

	return c
}

type correlatedChanPool[U any] struct {
	pool sync.Pool
}

// Get is a slight variance to idiomatic Go for sync.Pool,
// returning a correlatedChan instance set to handle only resp[U] with the specified id
func (p *correlatedChanPool[U]) Get(id uint64) *correlatedChan[U] {
	c := p.pool.Get().(*correlatedChan[U])
	c.setId(id)
	return c
}

// Put ensures that the returned instance is reset before reuse
func (p *correlatedChanPool[U]) Put(c *correlatedChan[U]) {
	c.setId(0)
	p.pool.Put(c)
}

func newCorrelatedChanPool[U any](maxRetries int, sendTimeout time.Duration, chanSize int) *correlatedChanPool[U] {
	return &correlatedChanPool[U]{
		pool: sync.Pool{
			New: func() any {
				return newCorrelatedChan[U](maxRetries, sendTimeout, chanSize)
			},
		},
	}
}
