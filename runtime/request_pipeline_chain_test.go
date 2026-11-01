package runtime

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
)

func TestCompiledMiddlewareChainOrdering(t *testing.T) {
	var order []string
	chain := compileMiddlewareChain([]httpadapter.Middleware{
		func(_ httpadapter.Context, next httpadapter.Next) error {
			order = append(order, "global before")
			if err := next(); err != nil {
				return err
			}
			order = append(order, "global after")
			return nil
		},
		func(_ httpadapter.Context, next httpadapter.Next) error {
			order = append(order, "route before")
			if err := next(); err != nil {
				return err
			}
			order = append(order, "route after")
			return nil
		},
	})
	if err := chain(&capturedContext{}, func() error {
		order = append(order, "handler")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(order, ","), "global before,route before,handler,route after,global after"; got != want {
		t.Fatalf("middleware order: got %s, want %s", got, want)
	}
}

func TestCompiledMiddlewareChainShortCircuitAndError(t *testing.T) {
	called := false
	chain := compileMiddlewareChain([]httpadapter.Middleware{
		func(_ httpadapter.Context, _ httpadapter.Next) error {
			return nil
		},
		func(_ httpadapter.Context, next httpadapter.Next) error {
			return next()
		},
	})
	if err := chain(&capturedContext{}, func() error {
		called = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("middleware short-circuit must not invoke downstream middleware or handler")
	}

	sentinel := errors.New("downstream failed")
	chain = compileMiddlewareChain([]httpadapter.Middleware{
		func(_ httpadapter.Context, next httpadapter.Next) error { return next() },
		func(_ httpadapter.Context, next httpadapter.Next) error { return next() },
	})
	if err := chain(&capturedContext{}, func() error { return sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("downstream error must propagate unchanged: %v", err)
	}
}

func TestCompiledMiddlewareChainRejectsRepeatedNext(t *testing.T) {
	for _, count := range []int{1, 3} {
		t.Run(fmt.Sprintf("layers_%d", count), func(t *testing.T) {
			visited := 0
			stack := make([]httpadapter.Middleware, count)
			for i := range stack {
				stack[i] = func(_ httpadapter.Context, next httpadapter.Next) error {
					if err := next(); err != nil {
						return err
					}
					if err := next(); err == nil || !strings.Contains(err.Error(), "more than once") {
						return fmt.Errorf("repeated next() must fail, got %v", err)
					}
					return nil
				}
			}
			chain := compileMiddlewareChain(stack)
			for i := 0; i < 3; i++ {
				if err := chain(&capturedContext{}, func() error {
					visited++
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			if visited != 3 {
				t.Fatalf("handler must execute once per request; got %d", visited)
			}
		})
	}
}

func TestCompiledMiddlewareChainConcurrentRequests(t *testing.T) {
	chain := compileMiddlewareChain([]httpadapter.Middleware{
		func(_ httpadapter.Context, next httpadapter.Next) error { return next() },
		func(_ httpadapter.Context, next httpadapter.Next) error { return next() },
		func(_ httpadapter.Context, next httpadapter.Next) error { return next() },
	})
	const requests = 256
	var wg sync.WaitGroup
	errorsCh := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			called := 0
			err := chain(&capturedContext{}, func() error {
				called++
				return nil
			})
			if err != nil {
				errorsCh <- err
			} else if called != 1 {
				errorsCh <- fmt.Errorf("handler invoked %d times", called)
			}
		}()
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Error(err)
	}
}
