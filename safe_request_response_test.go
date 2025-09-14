package saferr

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gford1000-go/saferr/types"
)

func ExampleNew() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requestor, receiver := New[int, float64](ctx)

	go func() {
		defer receiver.Close()

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

func ExampleNew_copySemantics() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requestor, receiver := New[string, string](ctx)

	go func() {
		defer receiver.Close()

		reflect := func(ctx context.Context, input *string) (*string, error) {
			return input, nil
		}

		var err error
		for err == nil {
			err = receiver.ListenAndHandle(ctx, reflect)
		}
	}()

	// Having the same Requestor instance shared between multiple variables is allowed
	// as New() returns an interface rather than a struct.
	// The same is true for the Response instance returned by New()

	// Create goroutines that concurrently interact with Responder with their own instance of Requestor
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var newRequestor types.Requestor[string, string] = requestor

			input := "Hello World"
			if response, err := newRequestor.Send(ctx, &input); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(*response)
			}
		}()
	}
	// Parent goroutine also concurrently interacts with the Responder
	input := "Hello World"
	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}
	wg.Wait()

	// Output:
	// Hello World
	// Hello World
	// Hello World
	// Hello World
	// Hello World
	// Hello World
}

func ExampleNew_withStruct() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type results struct {
		square     float64
		cube       float64
		reciprical float64
	}

	requestor, receiver := New[int, results](ctx)

	go func() {
		defer receiver.Close()

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

func ExampleNew_withMultiplex() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type choice int
	const (
		square choice = iota
		cube
	)

	type input struct {
		value  int
		choice choice
	}

	// Since the types used in New[T, U] are arbitrary, multiplexing can be done by using a struct
	// which also ensures a more robust contract between the requestor and responder
	requestor, receiver := New[input, int](ctx)

	go func() {
		defer receiver.Close()

		calc := func(ctx context.Context, input *input) (*int, error) {
			var result int
			switch input.choice {
			case square:
				result = input.value * input.value
			case cube:
				result = input.value * input.value * input.value
			default:
				return nil, fmt.Errorf("unknown choice: %d", input.choice)
			}
			return &result, nil
		}

		var err error
		for err == nil {
			err = receiver.ListenAndHandle(ctx, calc)
		}
	}()

	inputs := []*input{
		{
			value:  4,
			choice: square,
		},
		{
			value:  4,
			choice: cube,
		},
	}

	for _, input := range inputs {
		if response, err := requestor.Send(ctx, input); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(*response)
		}
	}

	// Output:
	// 16
	// 64
}

func ExampleGo() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reciprical := func(ctx context.Context, input *int) (*float64, error) {
		var result float64 = math.Round(100/float64(*input)) / 100
		return &result, nil
	}

	requestor := Go(ctx, reciprical)

	input := 4
	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}

	// Output: 0.25
}

func ExampleGo_withStruct() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type results struct {
		square     float64
		cube       float64
		reciprical float64
	}

	calc := func(ctx context.Context, input *int) (*results, error) {
		v := float64(*input)
		result := results{
			square:     v * v,
			cube:       v * v * v,
			reciprical: math.Round(100/v) / 100,
		}
		return &result, nil
	}

	requestor := Go(ctx, calc)

	input := 4
	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}

	// Output: {16 64 0.25}
}

func ExampleGo_withMultiplex() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Since the types used in New[T, U] are arbitrary, multiplexing can be done by using a struct
	// which also ensures a more robust contract between the requestor and responder

	type choice int
	const (
		square choice = iota
		cube
	)

	type input struct {
		value  int
		choice choice
	}

	calc := func(ctx context.Context, input *input) (*int, error) {
		var result int
		switch input.choice {
		case square:
			result = input.value * input.value
		case cube:
			result = input.value * input.value * input.value
		default:
			return nil, fmt.Errorf("unknown choice: %d", input.choice)
		}
		return &result, nil
	}

	requestor := Go(ctx, calc)

	inputs := []*input{
		{
			value:  4,
			choice: square,
		},
		{
			value:  4,
			choice: cube,
		},
	}

	for _, input := range inputs {
		if response, err := requestor.Send(ctx, input); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(*response)
		}
	}

	// Output:
	// 16
	// 64
}

func ExampleGo_withHooks() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reciprical := func(ctx context.Context, input *int) (*float64, error) {
		var result float64 = math.Round(100/float64(*input)) / 100
		return &result, nil
	}

	// Builder is used to capture messages across all goroutines, as this isn't
	// natively provided via fmt.Println() for Example tests.
	// Unfortunately Builder isn't threadsafe, so need to create threadsafe reader/writer funcs.
	var sb strings.Builder
	var lck sync.Mutex
	write := func(s string) {
		lck.Lock()
		defer lck.Unlock()
		sb.WriteString(s)
	}
	read := func() string {
		lck.Lock()
		defer lck.Unlock()
		return sb.String()
	}

	requestor := Go(ctx, reciprical,
		WithResponderTimeout(10*time.Millisecond),         // Duration before ListenAndServe() will exit
		WithRequestorGoneWayTimeout(250*time.Millisecond), // Duration between requests before Responder believes Requestor has gone
		WithGoPreStart(func(ctx context.Context) (context.Context, error) {
			write("In PreStart\n")
			return ctx, nil
		}),
		WithGoPostEnd(func(err error) {
			write("In PostEnd")
		}))

	input := 4
	if response, err := requestor.Send(ctx, &input); err != nil {
		write(fmt.Sprintf("%v", err))
	} else {
		write(fmt.Sprintf("%v\n", *response))
	}

	// Pause to allow Go()'s goroutine to exit and demonstrate PostEnd being called
	<-time.After(300 * time.Millisecond)

	fmt.Println(read())

	// Output:
	// In PreStart
	// 0.25
	// In PostEnd
}

func TestNewComms(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requestor, receiver := New[int, int](ctx,
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

	requestor, receiver := New[int, int](ctx,
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
	requestor, receiver := New[int, int](ctx,
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

	requestor, receiver := New[int, int](ctx)

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

	requestor, receiver := New[int, int](ctx)

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

func BenchmarkGo_0(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reflect := func(ctx context.Context, input *int) (*int, error) {
		return input, nil
	}

	i := 42
	requestor := Go(ctx, reflect)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		if response, err := requestor.Send(ctx, &i); err != nil {
			b.Fatal(err)
		} else if *response != i {
			b.Fatalf("Unexpected response returned: %d when should be %d", *response, i)
		}
	}
}

func BenchmarkGo_1(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reflect := func(ctx context.Context, input *int) (*int, error) {
		return input, nil
	}

	i := 42
	n := runtime.NumCPU()
	fmt.Println("NumCPU = ", n)

	requestor := Go(ctx, reflect, WithChanSize(n))

	var wg sync.WaitGroup
	wg.Add(n)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		defer wg.Done()
		for pb.Next() {
			if response, err := requestor.Send(ctx, &i); err != nil {
				b.Fatal(err)
			} else if *response != i {
				b.Fatalf("Unexpected response returned: %d when should be %d", *response, i)
			}
		}
	})

	wg.Wait()
}
