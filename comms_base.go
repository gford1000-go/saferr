package saferr

import (
	"context"
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
