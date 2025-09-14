package saferr

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/gford1000-go/saferr/mux"
	"github.com/gford1000-go/saferr/types"
)

func ExampleNewHandler() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := mux.NewHandler[int, float64, string](nil,
		&mux.Register[int, float64, string]{
			Key: "reciprical",
			Handler: func(ctx context.Context, input *int) (*float64, error) {
				var result float64 = math.Round(100/float64(*input)) / 100
				return &result, nil
			},
		}, &mux.Register[int, float64, string]{
			Key: "square",
			Handler: func(ctx context.Context, input *int) (*float64, error) {
				var result float64 = float64(*input * *input)
				return &result, nil
			},
		})

	requestor := Go(ctx, mux.Handler)

	v := 4
	input := types.Request[int, string, string]{
		Key:  "square",
		Data: &v,
	}

	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}

	input.Key = "reciprical"
	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}

	input.Key = "cube"
	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}

	// Output:
	// 16
	// 0.25
	// handler not found
}

func ExampleNewHandler_withKeyResolution() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resolver := mux.NewResolver(
		&mux.KeyResolver[string, string]{
			Key: "math/{func}",
			KeyResolver: func(key string, m *string) string {
				return fmt.Sprintf("math/%v", *m)
			},
		})

	mux := mux.NewHandler(resolver,
		&mux.Register[int, float64, string]{
			Key: "math/reciprical",
			Handler: func(ctx context.Context, input *int) (*float64, error) {
				var result float64 = math.Round(100/float64(*input)) / 100
				return &result, nil
			},
		}, &mux.Register[int, float64, string]{
			Key: "math/square",
			Handler: func(ctx context.Context, input *int) (*float64, error) {
				var result float64 = float64(*input * *input)
				return &result, nil
			},
		})

	requestor := Go(ctx, mux.Handler)

	v := 4
	input := types.Request[int, string, string]{
		Key:  "math/{func}",
		Meta: "square",
		Data: &v,
	}

	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}

	input.Meta = "reciprical"
	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}

	input.Meta = "cube"
	if response, err := requestor.Send(ctx, &input); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}

	// Output:
	// 16
	// 0.25
	// handler not found
}

func BenchmarkNewHandler(b *testing.B) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resolver := mux.NewResolver(
		&mux.KeyResolver[string, string]{
			Key: "math/{func}",
			KeyResolver: func(key string, m *string) string {
				return "math/" + *m
			},
		})

	mux := mux.NewHandler(resolver,
		&mux.Register[int, float64, string]{
			Key: "math/square",
			Handler: func(ctx context.Context, input *int) (*float64, error) {
				var result float64 = float64(*input * *input)
				return &result, nil
			},
		})

	i := 42
	requestor := Go(ctx, mux.Handler)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		input := types.Request[int, string, string]{
			Key:  "math/{func}",
			Meta: "square",
			Data: &i,
		}

		if response, err := requestor.Send(ctx, &input); err != nil {
			b.Fatal(err)
		} else if *response != float64(i*i) {
			b.Fatalf("Unexpected response returned: %v when should be %d", *response, i)
		}
	}
}
