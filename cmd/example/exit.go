package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"

	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"
)

type action func() error

func runAndWaitForExit(shutdown func(), runList ...action) error {
	ctx, shutdown := withCancel(shutdown)
	runList = append(runList, waitForSignalAction(ctx))
	var res sync.Map
	var once sync.Once
	var wg sync.WaitGroup
	run := func(idx int, item action) {
		err := invoke(item)
		if err != nil {
			res.Store(idx, err)
		}
		once.Do(shutdown)
		wg.Done()
	}
	wg.Add(len(runList))
	for i, item := range runList {
		go run(i, item)
	}
	wg.Wait()
	var fail error
	res.Range(func(key, value interface{}) bool {
		if fail == nil {
			fail = value.(error)
		} else {
			fail = multierror.Append(fail, value.(error))
		}
		return true
	})
	return fail
}

func withCancel(shutdown func()) (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return ctx, func() {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				log.WithField("panic", r).WithField("stack", stack).Error("shutdown panic")
			}
		}()
		cancel()
		shutdown()
	}
}

func invoke(item action) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			err = fmt.Errorf("action panic: %v, stack: %s", r, stack)
		}
	}()
	return item()
}

func waitForSignalAction(ctx context.Context) action {
	return func() error {
		return waitForSignal(ctx)
	}
}

func waitForSignal(ctx context.Context) error {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-signalChan:
		log.Info("shutdown received")
		return nil
	case <-ctx.Done():
		return nil
	}
}
