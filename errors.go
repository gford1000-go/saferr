package saferr

import "errors"

var ErrSendTimeout = errors.New("request timedout exceeded")

var ErrSenderIsClosed = errors.New("sender closed")

var ErrReceiverIsClosed = errors.New("receiver closed")

var ErrCommsChannelIsClosed = errors.New("comms channel has been closed")

// ErrContextCompleted returned if the request is being attempted but the context has completed
var ErrContextCompleted = errors.New("context is completed")

var ErrUncaughtHandlerPanic = errors.New("recovered receiver panic during handling")

var ErrUncaughtSenderPanic = errors.New("recovered panic during send")

var ErrRequestorGoneAway = errors.New("requestor gone away")
