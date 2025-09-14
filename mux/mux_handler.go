package mux

import (
	"context"
	"errors"

	"github.com/gford1000-go/saferr/types"
)

// ErrHandlerNotFound is returned if the key cannot be found in the MuxHandler
var ErrHandlerNotFound = errors.New("handler not found")

// Resolver encapsulates a map of KeyResolver information, created by NewResolver
// Calling Resolve will attempt to find a match, and if found the KeyResolver
// is used to attempt to resolve the key using the additional data
type Resolver[M any, K comparable] struct {
	Resolve func(key K, m *M) K
}

// KeyResolver maps a partially completed Key (i.e. with parameters or missing values) to a resolution
// fumction, which can generate a fully defined Key using the additional information supplied
// For example, this allows for Keys such as /customerSegment/{segment} to be fully resolved to /customerSegment/premier
// using the information provided by the Request.Meta, providing fine grain Handler selection
type KeyResolver[M any, K comparable] struct {
	// Key contains the partially completed key details
	Key K
	// KeyResolver defines a function that can fully resolve Keys
	KeyResolver func(key K, m *M) K
}

// NewResolver creates a new instance of Resolver containing the specified KeyResolver details
func NewResolver[M any, K comparable](resolvers ...*KeyResolver[M, K]) *Resolver[M, K] {

	// map is in closure only, to ensure readonly behaviour after creation
	m := map[K]func(K, *M) K{}

	for _, resolver := range resolvers {
		if resolver.KeyResolver != nil {
			m[resolver.Key] = resolver.KeyResolver
		}
	}

	return &Resolver[M, K]{
		Resolve: func(key K, meta *M) K {
			if r, ok := m[key]; !ok {
				return key
			} else {
				return r(key, meta)
			}
		},
	}
}

// Handler provides the Handler to present to Go() or Responder.ListenAndHandle()
type Handler[T, U, M any, K comparable] struct {
	Handler func(ctx context.Context, t *types.Request[T, M, K]) (*U, error)
}

// Register associate the specifed Key to a Handler.
// Keys for Handlers should always be fully defined - see the examples for NewHandler()
type Register[T, U any, K comparable] struct {
	// Fully defined Key against which the Handler can be found
	Key K
	// Handler for Requests with this Key
	Handler types.Handler[T, U]
}

// NewHandler initialises a new Handler instance with the specified resolver and set of handlers
// The resolver can be nil if none of the keys to the handlers require resolution
func NewHandler[T, U, M any, K comparable](resolver *Resolver[M, K], handlers ...*Register[T, U, K]) *Handler[T, U, M, K] {

	// Map is used inside a closure to enforce readonly behaviour after creation
	m := map[K]types.Handler[T, U]{}

	for _, v := range handlers {
		if v.Handler != nil {
			m[v.Key] = v.Handler
		}
	}

	return &Handler[T, U, M, K]{

		Handler: func(ctx context.Context, r *types.Request[T, M, K]) (*U, error) {
			h, ok := m[r.Key]
			if ok {
				return h(ctx, r.Data)
			}

			if resolver != nil && resolver.Resolve != nil {
				h, ok := m[resolver.Resolve(r.Key, &r.Meta)]
				if ok {
					return h(ctx, r.Data)
				}
			}

			return nil, ErrHandlerNotFound
		},
	}
}
