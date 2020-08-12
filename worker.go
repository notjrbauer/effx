package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
)

type count32 int32

func (c *count32) inc() int32 {
	return atomic.AddInt32((*int32)(c), 1)
}

func (c *count32) get() int32 {
	return atomic.LoadInt32((*int32)(c))
}

func (c *count32) set(v int32) {
	atomic.StoreInt32((*int32)(c), v)
}

type worker struct {
	cli     *redis.Client
	ws      *websocket.Conn
	l       *log.Logger
	count   count32
	average count32
}

// NewWorker returns a new worker.
func NewWorker(cli *redis.Client, ws *websocket.Conn, l *log.Logger) *worker {
	var c count32
	var avg count32
	return &worker{cli, ws, l, c, avg}
}

// Start init's the worker
func (w *worker) Start(ctx context.Context) {
	go func() {
		lv := int32(0)
		c := int32(1)
		for {
			select {
			case <-time.Tick(time.Second):
				i := w.count.get()
				lv = ((i - lv) / c)
				w.average.set(lv)
				w.l.Println(lv * int32(60))
				c++
			}
		}
	}()

	for {
		_, message, err := w.ws.ReadMessage()
		w.count.inc()
		if err != nil {
			w.l.Println("read:", err)
			return
		}
		go func() {
			var data map[string]interface{}
			if err := json.Unmarshal(message, &data); err != nil {
			}
			v := fmt.Sprintf("%v", data["message"])
			tokens := strings.Split(strings.ToLower(strip(v)), " ")

			for _, v := range tokens {
				err = w.cli.ZIncrBy(leaderboard, 1, v).Err()
				if err != nil {
					w.l.Println("zincrby:", err)
				}
			}
		}()
	}
}

// Stop safely shutdowns the worker and its wss::connections.
func (w *worker) Stop(ctx context.Context) error {
	w.l.Println("stop() complete")
	if err := w.ws.Close(); err != nil {
		w.l.Println("closing ws:", err)
		return err
	}
	if err := w.cli.Close(); err != nil {
		w.l.Println("closing cli:", err)
		return err
	}
	return nil
}

func strip(s string) string {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if ('a' <= b && b <= 'z') ||
			('A' <= b && b <= 'Z') ||
			('0' <= b && b <= '9') ||
			b == ' ' {
			result.WriteByte(b)
		}
	}
	return result.String()
}
