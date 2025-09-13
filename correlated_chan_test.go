package saferr

import (
	"sync"
	"testing"
	"time"
)

func TestCorrelatedChan(t *testing.T) {

	p := newCorrelatedChanPool[int](5, 100*time.Millisecond, 10)

	// Basic test of processing: can a resp, sent with the correct id, reach the receiver
	var id uint64 = 42

	c := p.Get(id)
	defer p.Put(c)

	var data = 99

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.send(&resp[int]{
			id:   id,
			data: &data,
		})
	}()

	wg.Wait()

	<-time.After(200 * time.Millisecond) // Allow the goroutine in the correlatedChanPool to run

	select {
	case r := <-c.getReceiverChan():
		if r.id != id {
			t.Fatalf("unexpected delivery: resp.id should be: %d, got: %d", id, r.id)
		}
		if *r.data != data {
			t.Fatalf("unexpected data in delivery: resp.data should be: %d, got: %d", data, *r.data)
		}
	default:
		t.Fatal("should have received resp")
	}
}

func TestCorrelatedChan_1(t *testing.T) {

	p := newCorrelatedChanPool[int](5, 100*time.Millisecond, 10)

	// Tests for ghost values being discarded
	var id uint64 = 99

	c := p.Get(id)
	defer p.Put(c)

	var data = 99

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 100 {
			c.send(&resp[int]{
				id:   uint64(i),
				data: &data,
			})
		}
	}()

	wg.Wait()

	<-time.After(200 * time.Millisecond) // Allow the goroutine in the correlatedChanPool to run

	select {
	case r := <-c.getReceiverChan():
		if r.id != id {
			t.Fatalf("unexpected delivery: resp.id should be: %d, got: %d", id, r.id)
		}
		if *r.data != data {
			t.Fatalf("unexpected data in delivery: resp.data should be: %d, got: %d", data, *r.data)
		}
	default:
		t.Fatal("should have received resp")
	}
}
