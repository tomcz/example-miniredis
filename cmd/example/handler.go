package main

import (
	"fmt"
	"net/http"
	"sync"

	workers "github.com/digitalocean/go-workers2"
	log "github.com/sirupsen/logrus"
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
