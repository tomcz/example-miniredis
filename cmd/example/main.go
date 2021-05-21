package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	logger "log"
	"net/http"
	"sync"

	"github.com/alicebob/miniredis/v2"
	workers "github.com/digitalocean/go-workers2"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var port = flag.Int("port", 3000, "service http port")

type queue struct {
	sync.Mutex
	data map[string]string
}

func (q *queue) enqueue(key, value string) {
	q.Lock()
	defer q.Unlock()

	q.data[key] = value
}

func (q *queue) dequeue(key string) (string, bool) {
	q.Lock()
	defer q.Unlock()

	if value, ok := q.data[key]; ok {
		delete(q.data, key)
		return value, ok
	}
	return "", false
}

func enqueue(p *workers.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.FormValue("key")
		if key == "" {
			http.Error(w, "no key", http.StatusBadRequest)
			return
		}
		jobID, err := p.Enqueue("helloQueue", "Add", key)
		if err != nil {
			log.WithError(err).Error("enqueue failed")
			http.Error(w, "enqueue failed", http.StatusInternalServerError)
			return
		}
		res := fmt.Sprintf("accepted job %s", jobID)
		http.Error(w, res, http.StatusAccepted)
	}
}

func dequeue(q *queue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "no key", http.StatusBadRequest)
			return
		}
		if value, ok := q.dequeue(key); ok {
			http.Error(w, value, http.StatusOK)
			return
		}
		http.Error(w, "no answer for you", http.StatusNotFound)
	}
}

func workerJob(q *queue) workers.JobFunc {
	return func(message *workers.Msg) error {
		log.Info("processing message ", message.Jid())
		key, err := message.Args().String()
		if err != nil {
			log.WithError(err).Error("cannot get args from message")
			return nil
		}
		q.enqueue(key, fmt.Sprintf("hello from job %s", message.Jid()))
		return nil
	}
}

func main() {
	flag.Parse()
	if err := realMain(); err != nil {
		log.WithError(err).Fatal("application failed")
	}
	log.Info("application stopped")
}

func realMain() error {
	q := &queue{data: make(map[string]string)}

	mr, err := miniredis.Run()
	if err != nil {
		return fmt.Errorf("failed to start miniredis: %w", err)
	}
	defer mr.Close()

	// replace go-workers2 default logger with a structured logger
	// see https://github.com/Sirupsen/logrus#logger-as-an-iowriter
	ww := log.WithField("component", "workers").Writer()
	defer ww.Close()
	workers.Logger = logger.New(ww, "", 0)

	manager, err := workers.NewManager(workers.Options{
		ProcessID:  "1",
		Namespace:  "example",
		ServerAddr: mr.Addr(),
	})
	if err != nil {
		return fmt.Errorf("failed to create worker manager: %w", err)
	}
	manager.AddWorker("helloQueue", 2, workerJob(q))
	p := manager.Producer()

	r := mux.NewRouter()
	r.HandleFunc("/enqueue", enqueue(p)).Methods("POST")
	r.HandleFunc("/dequeue", dequeue(q)).Methods("GET")

	workerAPI := http.NewServeMux()
	workers.RegisterAPIEndpoints(workerAPI)
	r.PathPrefix("/workers").Handler(http.StripPrefix("/workers", workerAPI))

	s := &http.Server{Addr: fmt.Sprintf(":%d", *port), Handler: r}

	var group errgroup.Group
	group.Go(func() error {
		log.Info("starting application on port ", *port)
		err := s.ListenAndServe()
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
		s.Shutdown(context.Background())
		return nil
	})
	return group.Wait()
}
