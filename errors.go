package saferr

import "errors"

// ErrSendTimeout returned if the Requestor.Send times out
var ErrSendTimeout = errors.New("request timedout exceeded")

// ErrRequestorIsClosed returned if the Requestor is closed but a transmission is attempted
var ErrRequestorIsClosed = errors.New("requestor closed")

// ErrResponderIsClosed returned if the Responder has closed
var ErrResponderIsClosed = errors.New("receiver closed")

// ErrCommsChannelIsClosed returned when it is known that transmission is not possible
var ErrCommsChannelIsClosed = errors.New("comms channel has been closed")

// ErrContextCompleted returned if the request is being attempted but the context has completed
var ErrContextCompleted = errors.New("context is completed")

// ErrUncaughtHandlerPanic returned if a panic occurs when handling a request
var ErrUncaughtHandlerPanic = errors.New("recovered receiver panic during handling")

// ErrUncaughtSendPanic returned if a send attempt generates a panic
var ErrUncaughtSendPanic = errors.New("recovered panic during send")

// ErrRequestorGoneAway returned when the Responder decides that its Requestor has gone
var ErrRequestorGoneAway = errors.New("requestor gone away")

// ErrUnableToSendRequest returned when a request cannot be sent after multiple attempts
var ErrUnableToSendRequest = errors.New("unable to send request")
