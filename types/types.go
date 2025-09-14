package types

import (
	"context"
)

// Requestor issues requests of type *T and receives responses of type *U
type Requestor[T any, U any] interface {
	// Send implements the sending of a single request
	Send(ctx context.Context, t *T) (*U, error)
}

// Handler processes a request of type *T into the result *U or an error
type Handler[T any, U any] func(ctx context.Context, t *T) (*U, error)

// Responder handles requests from the associated Requestor
type Responder[T any, U any] interface {
	// ListenAndHandle invokes the requestHandler to generate the response
	ListenAndHandle(ctx context.Context, requestHandler Handler[T, U]) error
	// Close allows resources to be tidied away
	Close()
}

// Request is a request issued by a Requestor, providing the key to the handler to be used
// to process the supplied value of type T.
type Request[T, M any, K comparable] struct {
	Key  K
	Meta M
	Data *T
}
