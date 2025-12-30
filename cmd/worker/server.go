package main

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"hauslet/config"
	platformQueue "hauslet/internal/platform/queue"
	"hauslet/internal/queue"

	"github.com/go-pkgz/lgr"
)

func startWorkerServer(cfg *config.GlobalConfig, registry *queue.Registry, log *lgr.Logger, ready *atomic.Bool) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, _ *http.Request) {
		if ready.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not_ready"))
	})

	for _, route := range platformQueue.QueueRouteList(cfg.YAML.Queue.Subjects) {
		mux.HandleFunc(route.Path, func(w http.ResponseWriter, r *http.Request) {
			if !ready.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			queueHeader := strings.TrimSpace(r.Header.Get("X-CloudTasks-QueueName"))
			if queueHeader != "" && queueHeader != route.Name {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if err := registry.Handle(r.Context(), route.Name, body); err != nil {
				log.Logf("ERROR task failed queue=%s: %v", route.Name, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
		})
	}

	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Logf("ERROR worker server failed: %v", err)
		}
	}()

	return srv
}
