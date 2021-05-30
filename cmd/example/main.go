package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	golog "log"
	"net/http"
	"os"

	"github.com/alicebob/miniredis/v2"
	workers "github.com/digitalocean/go-workers2"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func init() {
	if os.Getenv("ENV") == "dev" {
		log.SetFormatter(&log.TextFormatter{})
	} else {
		log.SetFormatter(&log.JSONFormatter{})
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

	// replace go-workers2 default logger with a structured logger
	// see https://github.com/Sirupsen/logrus#logger-as-an-iowriter
	ww := log.WithField("component", "workers").Writer()
	defer ww.Close()
	workers.Logger = golog.New(ww, "", 0)

	store := &dataStore{data: make(map[string]string)}

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

	var group errgroup.Group
	group.Go(func() error {
		log.Info("starting application on port ", *port)
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			log.Info("http server stopped")
			return nil
		}
		manager.Stop()
		return err
	})
	group.Go(func() error {
		log.Info("starting workers")
		manager.Run() // blocks waiting for exit signal
		log.Info("workers stopped")
		server.Shutdown(context.Background())
		return nil
	})
	return group.Wait()
}
