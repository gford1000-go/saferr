[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://en.wikipedia.org/wiki/MIT_License)
[![Documentation](https://img.shields.io/badge/Documentation-GoDoc-green.svg)](https://godoc.org/github.com/gford1000-go/saferr)

# saferr | Safe Request/Response

This package is intended to make `chan` based communications as safe and easy as possible between goroutines.  Whilst the basic form of
the communication is easy to implement, there are plenty of edge cases that can occur, which the package aims to remove.

The communication pattern is the need for a requestor to make a request to a receiver running within a second goroutine, who
responds with an appropriate response.  The pattern includes support for either the requestor or the responder disappearing at any time.

The implementation uses `sync.Pool` for its internal entities to minimise the allocation overhead, so that heavy use does not generate GC pressure.

## Details

This package has two primary functions, `New` and `Go`.

`New` returns a pair of initialised `Requestor[T, U any]` and `Responder[T, U any]` instances, such that the `Requestor` can send requests to the `Responder` (which is expected to be running in another goroutine).

`Go` simplifies use by creating and managing the lifetime of the goroutine itself, as shown below:

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    reciprical := func(ctx context.Context, input *int) (*float64, error) {
        var result float64 = math.Round(100/float64(*input)) / 100
        return &result, nil
    }

    requestor := Go(ctx, reciprical)

    input := 4
    response, _ := requestor.Send(ctx, &input)
    fmt.Println(*response) // 0.25
}
```

The `Go` function supports `Options` that can alter the goroutine behaviour if the caller provides hooks for pre-start initialisation activity, post-end termination cleanup activity, and cyclic activity in-between.

Cyclic support is possible as the `Responder.ListenAndHandle` function will timeout if there are no requests from the `Requestor`, and
this gives an opportunity for the goroutine to perform other tasks.  These should be short duration, to ensure `Requestor`s do not have to wait too long for a response to their request.

Check `Options` to see the full set of configuration available.

## Generics based

Both `New` and `Go` support arbitrary types, provided that type instances are accessible as pointers.

## Features

* Zero internal allocation to minimise GC stress, through the use of internal object pooling
* `Requestor` can be safely copied across goroutines, allowing concurrent request submission
* Requests guaranteed to be processed in first-in, first-out (FIFO) sequence by the `Responder`
* `Requestor` will only receive the response corresponding for their request (i.e. no ghost response side effects due to pooling)
* `Responder` is non-blocked waiting for `Requestor`, to maximise throughput
* `Requestor` timeout for each request, to provide blocking forever
* Graceful detection that the `Requestor` has gone away
* Graceful detection that the `Responder` has gone away
* Runtime panics are caught and converted to error responses, but will not cause application crashes.
* Support for multiple context scenarios:
  * Overall (parent) context completed
  * `Responder` processing context completion
  * Per request context completion

For more details, see the Examples.
