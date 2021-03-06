package waitgroup

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// some code refer golang.org/x/sync/errgroup

var (
	waitgroupCounter int32
	goroutineCounter int32
)

type WaitGroup struct {
	cancel func()
	waiter int32
	sync.Mutex
	fastFail bool
	wg       sync.WaitGroup

	Errs []error
}

type optionFunc func(*WaitGroup) error

func WithFastFail() optionFunc {
	return func(o *WaitGroup) error {
		o.fastFail = true
		return nil
	}
}

func NewWithContext(ctx context.Context, opts ...optionFunc) (*WaitGroup, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	wg := &WaitGroup{cancel: cancel}
	for _, opt := range opts {
		opt(wg)
	}
	return wg, ctx
}

func New(opts ...optionFunc) (*WaitGroup, context.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &WaitGroup{cancel: cancel}
	for _, opt := range opts {
		opt(wg)
	}
	return wg, ctx
}

func (g *WaitGroup) WaitTimeout(d time.Duration) {
	timer := time.NewTimer(d)
	defer func() {
		timer.Stop()
		g.close()
	}()

	sig := make(chan bool, 0)
	go func() {
		g.Wait()
		sig <- true
	}()

	select {
	case <-timer.C:
		return
	case <-sig:
		return
	}
}

// Wait wait block until all goroutine exit
func (g *WaitGroup) Wait() []error {
	g.wg.Wait()
	g.close()
	return g.Errs
}

func (g *WaitGroup) IsError() bool {
	return len(g.Errs) != 0
}

func (g *WaitGroup) close() {
	if g.cancel != nil {
		g.cancel()
	}
}

func (g *WaitGroup) GetWaiter() int {
	return int(atomic.LoadInt32(&g.waiter))
}

func (g *WaitGroup) Async(fn func() error) {
	g.run(fn)
}

func (g *WaitGroup) AsyncMany(fn func() error, count int) {
	if count <= 0 { // match uint32
		panic("invalid count")
	}

	for i := 0; i < count; i++ {
		g.run(fn)
	}
}

func (g *WaitGroup) incr() int32 {
	atomic.AddInt32(&goroutineCounter, 1)
	n := atomic.AddInt32(&g.waiter, 1)
	if n == 0 { // exception ?
		g.cancel()
	}
	if n == 1 { // first add
		atomic.AddInt32(&waitgroupCounter, 1)
	}
	return n
}

func (g *WaitGroup) decr() int32 {
	atomic.AddInt32(&goroutineCounter, -1)
	n := atomic.AddInt32(&g.waiter, -1)
	if n == 0 {
		g.close()
		atomic.AddInt32(&waitgroupCounter, -1)
	}
	return n
}

func (g *WaitGroup) run(fn func() error) {
	g.wg.Add(1)
	g.incr()

	go func() {
		defer func() {
			g.decr()
			g.wg.Done()
		}()

		if err := fn(); err != nil {
			g.Lock()
			g.Errs = append(g.Errs, err)
			if g.fastFail {
				g.close()
			}
			g.Unlock()
		}
	}()
}

func AsyncManyFunc(fn func(), callback func(), count int) {
	var wg = sync.WaitGroup{}

	for i := 0; i < count; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			fn()
		}()
	}
	if callback == nil {
		return
	}

	go func() {
		wg.Wait()
		callback()
	}()
}

func ConvertQueue(l interface{}) chan interface{} {
	s := reflect.ValueOf(l)
	c := make(chan interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		c <- s.Index(i).Interface()
	}
	close(c)
	return c
}

func incrWaitgroups() int32 {
	return atomic.AddInt32(&waitgroupCounter, 1)
}

func decrWaitgroups() int32 {
	return atomic.AddInt32(&waitgroupCounter, -1)
}

func GetRunningWaitgroups() int32 {
	return atomic.LoadInt32(&waitgroupCounter)
}

func incrGoroutines() int32 {
	return atomic.AddInt32(&goroutineCounter, 1)
}

func decrGoroutines() int32 {
	return atomic.AddInt32(&goroutineCounter, -1)
}

func GetRunningGoroutines() int32 {
	return atomic.LoadInt32(&goroutineCounter)
}
