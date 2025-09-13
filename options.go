package saferr

import (
	"context"
	"time"
)

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
	// GoPreStart called by Go() to initialise the Responder go routine prior to listening for requests.
	// GoPreStart returns the context that be used by ListenAndServe().
	// If GoPreStart returns an error then the go routine will exit immediately.
	GoPreStart func(context.Context) (context.Context, error)
	// GoPostEnd is invoked as a deferred call by Go() to support clean up of the Responder go routine on exit
	// If exit is triggered by an error from ListenAndServe(), this hook receives the error.
	GoPostEnd func(err error)
	// GoPostListen is called after ListenAndHandle() exits due to a timeout, allowing the Go() Responder
	// go routine to perform other tasks whilst waiting for a request.
	// If this returns an error, then the go routine exits with no further call to ListenAndHandle().
	// If ListenAndHandle() returns due to an error, GoPostListen will not be called.
	GoPostListen func(context.Context) error
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

// WithGoPreStart allows the PreStart hook to be set in a call to Go()
func WithGoPreStart(f func(context.Context) (context.Context, error)) func(*Options) {
	return func(o *Options) {
		o.GoPreStart = f
	}
}

// WithGoPostEnd allows the PostEnd hook to be set in a call to Go()
func WithGoPostEnd(f func(error)) func(*Options) {
	return func(o *Options) {
		o.GoPostEnd = f
	}
}

// WithGoPostListen allows the PostListen hook to be set in a call to Go()
func WithGoPostListen(f func(context.Context) error) func(*Options) {
	return func(o *Options) {
		o.GoPostListen = f
	}
}
