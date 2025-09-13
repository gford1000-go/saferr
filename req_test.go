package saferr

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewReqPool(t *testing.T) {

	cp := newCorrelatedChanPool[int](5, 100*time.Millisecond, 10)

	p := newReqPool[int](cp, getIncrementer())

	var size = 2
	var usages = 10000
	var errs = make([]error, size)

	var wg sync.WaitGroup

	for i := range size {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errs[n] = fmt.Errorf("caught panic for %d: %v", n, r)
				}
			}()

			for j := range usages {
				r := p.Get(&j)
				p.Put(r)
			}
		}(i)
	}

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkIncrementer(b *testing.B) {

	incrementer := getIncrementer()

	b.ResetTimer()

	for range b.N {
		incrementer()
	}
}
