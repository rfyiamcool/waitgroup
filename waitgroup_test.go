package waitgroup

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_MultiForLeak(t *testing.T) {
	Test_Simple(t)
	Test_WaitTimeout(t)
	Test_Ctx(t)
	Test_FastFail(t)

	assert.Equal(t, int(GetRunningGoroutines()), 0)
	assert.Equal(t, int(GetRunningWaitgroups()), 0)
}

func Test_Simple(t *testing.T) {
	counter := 0
	lock := sync.Mutex{}
	fn := func() error {
		lock.Lock()
		defer lock.Unlock()

		time.Sleep(1e6)
		counter++
		return nil
	}

	wg, ctx := New()
	wg.Async(fn)
	wg.AsyncMany(fn, 5)

	assert.Equal(t, int(GetRunningGoroutines()), 6)
	assert.Equal(t, int(GetRunningWaitgroups()), 1)

	<-ctx.Done()
	wg.Wait()

	assert.Equal(t, wg.IsError(), false)
	assert.Equal(t, counter, 6)

	assert.Equal(t, int(GetRunningGoroutines()), 0)
	assert.Equal(t, int(GetRunningWaitgroups()), 0)
}

func Test_Ctx(t *testing.T) {
	counter := 0
	lock := sync.Mutex{}
	fn := func() error {
		lock.Lock()
		defer lock.Unlock()

		counter++
		return nil
	}

	errval := errors.New("errfn")
	errfn := func() error {
		lock.Lock()
		defer lock.Unlock()

		counter++
		return errval
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg, cctx := NewWithContext(ctx)
	wg.Async(errfn)
	wg.AsyncMany(fn, 5)

	select {
	case <-cctx.Done():
	case <-ctx.Done():
	}

	assert.Equal(t, counter, 6)
	assert.Equal(t, wg.IsError(), true)
	assert.Equal(t, errval, wg.Errs[0])
}

func Test_WaitTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg, cctx := NewWithContext(ctx)
	// _ = cctx

	wg.Async(
		func() error {
			time.Sleep(3 * time.Second)
			return io.ErrClosedPipe
		},
	)
	assert.Greater(t, int(GetRunningGoroutines()), 0)
	assert.Greater(t, int(GetRunningWaitgroups()), 0)

	// force to exit with timeout
	wg.WaitTimeout(1 * time.Second)
	select {
	case <-cctx.Done():
	}

	assert.Equal(t, int(GetRunningGoroutines()), 1)
	assert.Equal(t, int(GetRunningWaitgroups()), 1)

	// wait all g exit
	wg.Wait()
	assert.Equal(t, int(GetRunningGoroutines()), 0)
	assert.Equal(t, int(GetRunningWaitgroups()), 0)
}

func Test_AsyncManyFunc(t *testing.T) {
	counter := 0
	lock := sync.Mutex{}
	fn := func() {
		lock.Lock()
		defer lock.Unlock()

		counter++
	}

	q := make(chan bool, 1)
	cb := func() {
		close(q)
	}

	AsyncManyFunc(fn, cb, 10)
	<-q

	assert.Equal(t, counter, 10)
}

func Test_FastFail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg, cctx := NewWithContext(ctx, WithFastFail())
	// _ = cctx

	start := time.Now()
	wg.Async(
		func() error {
			time.Sleep(2 * time.Second)
			return nil
		},
	)

	wg.Async(
		func() error {
			time.Sleep(1 * time.Second)
			return io.ErrClosedPipe
		},
	)

	<-cctx.Done()
	assert.LessOrEqual(t, time.Since(start).Seconds(), float64(2))

	assert.Equal(t, int(GetRunningGoroutines()), 1)
	assert.Equal(t, int(GetRunningWaitgroups()), 1)

	time.Sleep(2 * time.Second)

	assert.Equal(t, int(GetRunningGoroutines()), 0)
	assert.Equal(t, int(GetRunningWaitgroups()), 0)
}
