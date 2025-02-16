package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	golog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alicebob/miniredis/v2"
	workers "github.com/digitalocean/go-workers2"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func init() {
	if os.Getenv("ENV") == "production" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{})
	}
}

var port = flag.Int("port", 3000, "service http port")

func main() {
	flag.Parse()
	if err := realMain(); err != nil {
		log.WithError(err).Fatal("application failed")
	}
	log.Info("application stopped")
}

func realMain() error {
	mr, err := miniredis.Run()
	if err != nil {
		return fmt.Errorf("failed to start miniredis: %w", err)
	}
	defer mr.Close()

	store := &dataStore{data: make(map[string]string)}

	// replace go-workers2 default logger with a structured logger
	// see https://github.com/Sirupsen/logrus#logger-as-an-iowriter
	ww := log.WithField("component", "workers").Writer()
	defer ww.Close()

	workers.Logger = golog.New(ww, "", 0)
	manager, err := workers.NewManager(workers.Options{
		ProcessID:  "1",
		Namespace:  "example",
		ServerAddr: mr.Addr(),
	})
	if err != nil {
		return fmt.Errorf("failed to create worker manager: %w", err)
	}
	registerWorker(store, manager)
	producer := manager.Producer()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: createHandler(store, producer),
	}

	ctx, cancel := context.WithCancel(context.Background())

	var group errgroup.Group
	group.Go(func() error {
		defer cancel()
		log.Info("starting server on port ", *port)
		return server.ListenAndServe()
	})
	group.Go(func() error {
		defer cancel()
		log.Info("starting manager")
		manager.Run()
		return nil
	})
	group.Go(func() error {
		<-ctx.Done()
		log.Info("stopping server")
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer waitCancel()
		return server.Shutdown(waitCtx)
	})
	group.Go(func() error {
		defer func() {
			// manager.Stop can panic
			_ = recover()
		}()
		<-ctx.Done()
		log.Info("stopping manager")
		manager.Stop()
		return nil
	})
	group.Go(func() error {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-signalChan:
			log.Info("shutting down")
			cancel()
			return nil
		case <-ctx.Done():
			log.Info("errored out")
			return nil
		}
	})
	err = group.Wait()
	if errors.Is(err, http.ErrServerClosed) {
		return nil // expected
	}
	return err
}
