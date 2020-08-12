package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/gorilla/mux"
)

func top(s *service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		words, err := s.TopRanks()
		if err != nil {
		}
		json.NewEncoder(w).Encode(&words)
	}
}

func standing(s *service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		s, err := s.Standing(id)
		if err != nil {
		}
		json.NewEncoder(w).Encode(&s)
	}
}

// convient
func health() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// quick logger
func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// quick tracer
func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(context.Background(), requestIDKey, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
