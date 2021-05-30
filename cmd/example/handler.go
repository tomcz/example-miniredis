package main

import (
	"fmt"
	"net/http"
	"sync"

	workers "github.com/digitalocean/go-workers2"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const queueName = "helloQueue"

type dataStore struct {
	sync.Mutex
	data map[string]string
}

func (s *dataStore) put(key, value string) {
	s.Lock()
	s.data[key] = value
	s.Unlock()
}

func (s *dataStore) pop(key string) (string, bool) {
	s.Lock()
	defer s.Unlock()

	if value, ok := s.data[key]; ok {
		delete(s.data, key)
		return value, ok
	}
	return "", false
}

func registerWorker(s *dataStore, m *workers.Manager) {
	m.AddWorker(queueName, 2, workerJob(s))
}

func workerJob(s *dataStore) workers.JobFunc {
	return func(message *workers.Msg) error {
		log.Info("processing message ", message.Jid())
		key, err := message.Args().String()
		if err != nil {
			log.WithError(err).Error("cannot get args from message")
			return nil
		}
		s.put(key, fmt.Sprintf("hello from job %s", message.Jid()))
		return nil
	}
}

func createHandler(s *dataStore, p *workers.Producer) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/enqueue", enqueue(p)).Methods("POST")
	r.HandleFunc("/dequeue", dequeue(s)).Methods("GET")

	workerAPI := http.NewServeMux()
	workers.RegisterAPIEndpoints(workerAPI)
	r.PathPrefix("/workers").Handler(http.StripPrefix("/workers", workerAPI))

	return r
}

func enqueue(p *workers.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.FormValue("key")
		if key == "" {
			http.Error(w, "no key", http.StatusBadRequest)
			return
		}
		jobID, err := p.Enqueue(queueName, "Add", key)
		if err != nil {
			log.WithError(err).Error("enqueue failed")
			http.Error(w, "enqueue failed", http.StatusInternalServerError)
			return
		}
		res := fmt.Sprintf("accepted job %s", jobID)
		http.Error(w, res, http.StatusAccepted)
	}
}

func dequeue(s *dataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "no key", http.StatusBadRequest)
			return
		}
		if value, ok := s.pop(key); ok {
			http.Error(w, value, http.StatusOK)
			return
		}
		http.Error(w, "no answer for you", http.StatusNotFound)
	}
}
