package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/alicebob/miniredis/v2"
	workers "github.com/digitalocean/go-workers2"
	"github.com/gorilla/mux"
)

var port = flag.Int("service", 3000, "service http port")

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
		// The "Add" value for a parameter named "class" feels a little strange to me.
		// Changing it to an arbitrary value seems to work just as well. My suspicion
		// is that it relates to the ruby Sidekiq implementation needing the name of
		// the worker class that it should invoke when the enqueued message gets picked
		// up for processing. I guess we have it available in case we are mixing it up
		// with ruby's Sidekiq workers and want them to be able to pick up any jobs
		// that we enqueue from a go producer.
		// See: https://github.com/mperham/sidekiq/wiki/The-Basics#client
		jobID, err := p.Enqueue("helloQueue", "Add", key)
		if err != nil {
			log.Println("enqueue failed with", err)
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

func stats(m *workers.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		stats, err := m.GetStats()
		if err != nil {
			log.Println("failed to get stats from manager:", err)
			http.Error(w, "worker stats unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(stats)
	}
}

func workerJob(q *queue) workers.JobFunc {
	return func(message *workers.Msg) error {
		log.Println("processing message", message.Jid())
		key, err := message.Args().String()
		if err != nil {
			log.Println("cannot get args from message:", err)
			return nil
		}
		q.enqueue(key, fmt.Sprintf("hello from job %s", message.Jid()))
		return nil
	}
}

func main() {
	flag.Parse()
	if err := realMain(); err != nil {
		log.Fatalln("application failed:", err)
	}
	log.Println("application stopped")
}

func realMain() error {
	ctx, cancelFunc := context.WithCancel(context.Background())

	q := &queue{data: make(map[string]string)}

	mr, err := miniredis.Run()
	if err != nil {
		return fmt.Errorf("failed to start miniredis: %w", err)
	}
	defer mr.Close()

	// ProcessID uniquely identifies this instance.
	// How do we set that up in a multi-node environment
	// like k8s where these things can go up & down???
	// Maybe we can't and therefore lose all the in-process
	// jobs and need something that will eventually re-queue
	// any jobs that have not completed. That means that we
	// need to keep track of this ourselves or peek into
	// queues that have not been updated after some time.
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
	r.HandleFunc("/stats", stats(manager)).Methods("GET")
	s := &http.Server{Addr: fmt.Sprintf(":%d", *port), Handler: r}

	shutdown := func() {
		cancelFunc() // stop waiting for exit
		s.Shutdown(context.Background())
	}
	waitForExit(shutdown,
		func() error {
			log.Println("starting application on port", *port)
			return s.ListenAndServe()
		},
		func() error {
			log.Println("starting workers")
			manager.Run() // blocks waiting for exit signal
			log.Println("workers stopped")
			return nil
		},
		waitForSignalAction(ctx),
	)
	return nil
}
