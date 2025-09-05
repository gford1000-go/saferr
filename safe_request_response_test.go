package saferr

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"
)

func ExampleNewComms() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requestor, receiver := NewComms[int, float64](ctx)

	go func() {

		reciprical := func(ctx context.Context, input *int) (*float64, error) {
			var result float64 = math.Round(100/float64(*input)) / 100
			return &result, nil
		}

		var err error
		for err == nil {
			err = receiver.ListenAndHandle(ctx, reciprical)
		}
	}()

	input := 4
	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}

	// Output: 0.25
}

func ExampleNewComms_withStruct() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type results struct {
		square     float64
		cube       float64
		reciprical float64
	}

	requestor, receiver := NewComms[int, results](ctx)

	go func() {

		calc := func(ctx context.Context, input *int) (*results, error) {
			v := float64(*input)
			result := results{
				square:     v * v,
				cube:       v * v * v,
				reciprical: math.Round(100/v) / 100,
			}
			return &result, nil
		}

		var err error
		for err == nil {
			err = receiver.ListenAndHandle(ctx, calc)
		}
	}()

	input := 4
	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}

	// Output: {16 64 0.25}
}

func TestNewComms(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requestor, receiver := NewComms[int, int](ctx,
		WithRequestorTimeout(150*time.Millisecond))

	// This test simulates what should happen if the receiver is closed
	// Once closed, all requests should fail

	receiver.Close()

	input := 42
	response, err := requestor.Send(ctx, &input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !(errors.Is(err, ErrCommsChannelIsClosed) || errors.Is(err, ErrSendTimeout)) {
		t.Fatalf("unexpected error returned: %v", err)
	}
	if response != nil {
		t.Fatalf("unexpected response: %v", response)
	}
}

func TestNewComms_1(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// This test simulates what should happen if the receiver thinks the requestor has disappeared,
	// namely to close itself down.
	// The test goroutine will sleep to make it look like it has gone away, and then attempt to send a request
	// which will fail, as the chan has been closed by the receiver.

	requestor, receiver := NewComms[int, int](ctx,
		WithRequestorGoneWayTimeout(500*time.Millisecond),
		WithResponderTimeout(100*time.Millisecond),
		WithRequestorTimeout(150*time.Millisecond))

	go func() {
		defer receiver.Close() // Add this to close the channel

		reflect := func(ctx context.Context, input *int) (*int, error) {
			return input, nil
		}

		var err error
		for err == nil {
			err = receiver.ListenAndHandle(ctx, reflect)
		}
	}()

	<-time.After(550 * time.Millisecond) // Long enough so that goroutine has called Close()

	input := 42
	response, err := requestor.Send(ctx, &input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !(errors.Is(err, ErrCommsChannelIsClosed) || errors.Is(err, ErrSendTimeout)) {
		t.Fatalf("unexpected error returned: %v", err)
	}
	if response != nil {
		t.Fatalf("unexpected response: %v", response)
	}
}

func TestNewComms_2(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// In this test, the receiver goroutine has its own context, which ends early
	// Therefore any communication to the receiver should now time out
	childCtx, childCancel := context.WithCancel(ctx)

	// Here we set a short timeout on the requestor so the test runs quickly
	requestor, receiver := NewComms[int, int](ctx,
		WithRequestorTimeout(120*time.Millisecond))

	go func() {

		reflect := func(ctx context.Context, input *int) (*int, error) {
			return input, nil
		}

		var err error
		for err == nil {
			err = receiver.ListenAndHandle(childCtx, reflect)
		}
	}()

	// Context completion prevents any further processing
	childCancel()

	// Inherent race condition ... sleep will allow the goroutine to wake up and exit
	<-time.After(time.Millisecond)

	// Should see that Send exits after a 120ms timeout, observable with go test -v
	input := 42
	response, err := requestor.Send(ctx, &input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrSendTimeout) {
		t.Fatalf("unexpected error: %v\n", err)
	}
	if response != nil {
		fmt.Printf("unexpected response: %v\n", *response)
	}
}

func TestNewComms_3(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	requestor, receiver := NewComms[int, int](ctx)

	go func() {

		reflect := func(ctx context.Context, input *int) (*int, error) {
			return input, nil
		}

		var err error
		for err == nil {
			err = receiver.ListenAndHandle(ctx, reflect)
		}
	}()

	cancel() // Context completion prevents sending

	input := 42
	response, err := requestor.Send(ctx, &input)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrContextCompleted) {
		t.Fatalf("unexpected error: %v\n", err)
	}

	if response != nil {
		t.Fatalf("unexpected response: %v\n", response)
	}
}

func TestNewComms_4(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requestor, receiver := NewComms[int, int](ctx)

	go func() {

		// Confirm that the receiver handles a panic and continues ok
		var reqCount = new(int)

		reflectOrBoom := func(ctx context.Context, input *int) (*int, error) {
			(*reqCount)++
			if (*reqCount)%2 == 1 {
				return input, nil
			}
			panic(fmt.Sprintf("Count: %d: !Boom", *reqCount))
		}

		var err error
		for err == nil {
			err = receiver.ListenAndHandle(ctx, reflectOrBoom)
		}
	}()

	input := 42

	// Should be able to have successful communications despite some requests generating panicks
	for i := range 10 {
		response, err := requestor.Send(ctx, &input)

		if i%2 == 0 {
			if err != nil {
				t.Fatalf("unexpected error received, cycle %d: %v", i, err)
			}
			if response == nil {
				t.Fatalf("cycle %d: response is nil", i)
			}
		} else {
			if err == nil {
				t.Fatalf("cycle %d: expected error, got nil", i)
			}

			if !errors.Is(err, ErrUncaughtHandlerPanic) {
				t.Fatalf("cycle %d: unexpected error: %v\n", i, err)
			}

			if response != nil {
				t.Fatalf("cycle %d: unexpected response: %v\n", i, response)
			}
		}
	}
}
