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
	// CorrelatedChanSize sets the size of the buffer for chan that the Responder places resp[U] onto.
	CorrelatedChanSize int
	// CorrelatedChanRetries sets the number of times the correlatedChan will attempt to add a valid resp[U]
	// onto the Requestor chan, before assuming the Requestor has gone away and discarding
	CorrelatedChanRetries int
	// CorrelatedChanAddTimeout sets the duration that the correlationChan will wait for the Requestor to
	// receive the resp[U] on the Requestor chan, before timing out.  This is typically small, as the
	// Requestor should be blocked to receive the resp[U].
	CorrelatedChanAddTimeout time.Duration
}

var defaults Options = Options{
	RequestorTimeout:         30 * time.Second,
	RequestorGoneAwayTimeout: 2 * time.Minute,
	ResponderTimeout:         1 * time.Second, // Expect the Responder to spin on a for { } loop
	ChanSize:                 100,
	CorrelatedChanSize:       10,
	CorrelatedChanRetries:    5,
	CorrelatedChanAddTimeout: 100 * time.Millisecond,
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

// WithCorrelatedChanSize sets the size of the buffer of the non-block chan *resp[U],
// to which the Responder returns the result of the Handler call.  Default: 10
func WithCorrelatedChanSize(size int) func(*Options) {
	return func(o *Options) {
		if size > defaults.CorrelatedChanSize {
			o.CorrelatedChanSize = size
		}
	}
}

// WithCorrelatedChanRetries sets the number of retries that a correlatedChan[U] will
// attempt, to transfer a valid resp[U] to its Requestor, before assuming that the
// Requestor has gone away and discarding the resp[U].  Default: 5
func WithCorrelatedChanRetries(retries int) func(*Options) {
	return func(o *Options) {
		if retries > defaults.CorrelatedChanRetries {
			o.CorrelatedChanRetries = retries
		}
	}
}

// WithCorrelatedChanAddTimeout sets the timeout duration for retries for the correlatedChan,
// as it is attempting to add the resp[U] onto the chan that its Requestor should be blocked on.
// This timeout should be short, and hence is floored at 100ms and capped at 1min
func WithCorrelatedChanAddTimeout(d time.Duration) func(*Options) {
	return func(o *Options) {
		if d > defaults.CorrelatedChanAddTimeout && d < time.Minute {
			o.CorrelatedChanAddTimeout = d
		}
	}
}
