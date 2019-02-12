package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/TimothyCole/timcole.me/pkg"
	"github.com/TimothyCole/timcole.me/pkg/firehose"
	"github.com/TimothyCole/timcole.me/pkg/ping"
	"github.com/TimothyCole/timcole.me/pkg/security"
	"github.com/TimothyCole/timcole.me/pkg/sockets"
	"github.com/TimothyCole/timcole.me/pkg/spotify"
	"github.com/go-redis/redis"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	router = mux.NewRouter()
	err    error
	store  *redis.Client
)

func init() {
	store = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_AUTH"),
	})

	if _, err := store.Ping().Result(); err != nil {
		panic(err)
	}

	log.Printf("Connected to Redis [%s]\n", store.Options().Addr)
}

func main() {
	router.Use(middleware)

	router.HandleFunc("/login", security.UserLogin).Methods("POST")
	router.HandleFunc("/stats", pkg.SBStats).Methods("GET")

	router.HandleFunc("/spotify/playing", spotify.GetPlaying).Methods("GET")

	// WebSockets
	pubsub := sockets.New()
	go pubsub.Start()
	pubsub.AddHandler((ping.New(pubsub)).Handler, "ping")
	if os.Getenv("TWITCH_OAUTH") != "" {
		pubsub.AddHandler((firehose.New(pubsub)).Handler, "firehose")
	}
	router.Handle("/ws", security.WSMiddleWare(
		http.HandlerFunc(pubsub.Handler),
	)).Methods("GET")

	// Admin API Router
	var admin = router.PathPrefix("/admin").Subrouter()
	admin.Use(security.UserMiddleWare)
	admin.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) }).Methods("GET")

	// API 404 Handler
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"status": 404, "error": "StatusNotFound"}`))
	})

	r := handlers.CombinedLoggingHandler(os.Stdout, router)
	r = handlers.ProxyHeaders(r)
	var httpServer = &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      r,
		Addr:         ":6969",
	}

	log.Printf("HTTP Server Started [%s]\n", httpServer.Addr)
	panic(httpServer.ListenAndServe())
}
