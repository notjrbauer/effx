package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type key int

const (
	requestIDKey key = 0
	leaderboard      = "top"
)

var (
	// Version is the version of the app
	Version string = ""
	// GitTag is the git tag of the app
	GitTag string = ""
	// GitCommit is the commit hash of the app
	GitCommit string = ""
	// GitTreeState represents dirty or clean states
	GitTreeState string = ""

	listenAddr string = ""
	remoteAddr string = ""
	healthy    int32
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	flag.StringVar(&listenAddr, "listen-addr", ":3000", "server listen address")
	flag.StringVar(&remoteAddr, "remote-addr", "chaotic-stream.herokuapp.com", "remote listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Word Service")
	logger.Println("Version:", Version)
	logger.Println("GitTag:", GitTag)
	logger.Println("GitCommit:", GitCommit)
	logger.Println("GitTreeState:", GitTreeState)

	logger.Println("Server is starting ...")

	client := redis.NewClient(&redis.Options{
		Addr:     "db:6379",
		Password: "",
		DB:       0,
	})

	u := url.URL{Scheme: "wss", Host: remoteAddr}
	logger.Printf("connecting to %s", u.String())

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	w := NewWorker(client, ws, logger)
	go func() {
		w.Start(context.Background())
	}()

	svc := NewService(client, logger)

	router := mux.NewRouter()
	router.Handle("/health", health())
	router.Handle("/api/standing/top", top(svc))
	router.Handle("/api/standing/{id}", standing(svc))

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	lh := logging(logger)
	th := tracing(nextRequestID)
	tracingLogger := th(lh(router))

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      tracingLogger,
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		ranks, _ := svc.TopRanks()
		jsonData, err := json.MarshalIndent(ranks, "", "    ")
		if err != nil {
		}

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		if err := w.Stop(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the worker: %v\n", err)
		}
		logger.Println(string(jsonData))
		logger.Println("Total Event Count", w.count.get())
		logger.Println("Average event per minute", w.average.get()*int32(60))

		close(done)
	}()

	logger.Println("Server is ready to handle requests at", listenAddr)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %b\n", listenAddr, err)
	}
	<-done
	logger.Println("Server stopped")
}
