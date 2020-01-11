package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/alicebob/miniredis/v2"
	workers "github.com/digitalocean/go-workers2"
	"github.com/gorilla/mux"
)

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

func enqueue(q *workers.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.FormValue("key")
		if key == "" {
			http.Error(w, "no message", http.StatusBadRequest)
			return
		}
		res, err := q.Enqueue("helloQueue", "Add", key)
		if err != nil {
			log.Println("enqueue failed with", err)
			http.Error(w, "enqueue failed", http.StatusInternalServerError)
			return
		}
		log.Println("enqueue response:", res)
		http.Error(w, "accepted", http.StatusAccepted)
	}
}

func dequeue(q *queue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "no message", http.StatusBadRequest)
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
		log.Println("processing message", message.Jid())
		key, err := message.Args().String()
		if err != nil {
			log.Println("cannot get args:", err)
			return nil
		}
		q.enqueue(key, fmt.Sprintf("hello from job %s", message.Jid()))
		return nil
	}
}

func main() {
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
		return fmt.Errorf("failed to start redis: %w", err)
	}
	defer mr.Close()

	manager, err := workers.NewManager(workers.Options{
		ProcessID:  "1",
		Namespace:  "example",
		ServerAddr: mr.Addr(),
	})
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}
	manager.AddWorker("helloQueue", 2, workerJob(q))
	p := manager.Producer()

	r := mux.NewRouter()
	r.HandleFunc("/enqueue", enqueue(p)).Methods("POST")
	r.HandleFunc("/dequeue", dequeue(q)).Methods("GET")
	s := &http.Server{Addr: ":3000", Handler: r}

	shutdown := func() {
		cancelFunc()
		s.Shutdown(context.Background())
	}
	waitForExit(shutdown,
		func() error { return s.ListenAndServe() },
		func() error { manager.Run(); return fmt.Errorf("manager stopped") },
		waitForSignalAction(ctx),
	)
	return nil
}
