package machine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/skovtunenko/machine"
)

func Test(t *testing.T) {
	m, closeFn := machine.New()
	var count = 0

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer closeFn()
	m.Go(ctx, func(ctx context.Context) error {
		return m.Subscribe(ctx, "testing.*", func(ctx context.Context, msg machine.Message) (bool, error) {
			t.Logf("(%s) got message: %v", msg.Channel, msg.Body)
			if !strings.Contains(msg.Channel, "testing") {
				t.Fatal("expected channel to contain 'testing'")
			}
			count++
			return count < 3, nil
		})
	})
	time.Sleep(1 * time.Second)
	for i := 0; i < 3; i++ {
		m.Publish(ctx, machine.Message{
			Channel: "testing",
			Body:    "hello world",
		})
	}
	if err := m.Wait(); err != nil {
		t.Fatal(err)
	}
	if count < 3 {
		t.Fatal("count < 3", count)
	}
}

func TestWithThrottledRoutines(t *testing.T) {
	max := 3
	m, closeFn := machine.New(machine.WithThrottledRoutines(max))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer closeFn()
	for i := 0; i < 100; i++ {
		i := i
		m.Go(ctx, func(ctx context.Context) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if current := m.Current(); current > max {
				t.Fatalf("more routines running %v than max threshold: %v", current, max)
			}
			t.Logf("(%v)", i)
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}
	m.Wait()
}

func Benchmark(b *testing.B) {
	b.Run("publish", func(b *testing.B) {
		m, closeFn := machine.New()
		defer closeFn()
		go func() {
			m.Subscribe(context.Background(), "testing.*", func(ctx context.Context, _ machine.Message) (bool, error) {
				return true, nil
			})
		}()
		for i := 0; i < b.N; i++ {
			m.Publish(context.Background(), machine.Message{
				Channel: "testing",
				Body:    i,
			})
		}
	})
}
