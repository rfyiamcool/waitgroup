# waitgroup

extend std waitgroup faeture

- easy api
- add timeout
- add context
- concurrency go func

## usage:

see more example in test

[example](waitgroup_test.go)

```go
func Test_Simple(t *testing.T) {
	counter := 0
	lock := sync.Mutex{}
	fn := func() error {
		lock.Lock()
		defer lock.Unlock()

		counter++
		return nil
	}

	wg, ctx := New()
	wg.Async(fn)
	wg.AsyncMany(fn, 5)

	<-ctx.Done()
	wg.Wait()

	assert.Equal(t, wg.IsError(), false)
	assert.Equal(t, counter, 6)
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
			time.Sleep(5 * time.Second)
			return io.ErrClosedPipe
		},
	)

	wg.WaitTimeout(1 * time.Second)
	<-cctx.Done()
}
```
