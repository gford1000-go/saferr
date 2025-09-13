package saferr

import "time"

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
