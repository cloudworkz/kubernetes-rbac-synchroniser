package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var address = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
var (
	roleUpdates = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "role_updates",
			Help: "Cumulative number of role update operations",
		},
		[]string{"count"},
	)

	roleUpdateErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "role_update_errors",
			Help: "Cumulative number of errors during role update operations",
		},
		[]string{"count"},
	)
)

func main() {
	flag.Parse()

	stopChan := make(chan struct{}, 1)

	go serveMetrics(address)
	go handleSigterm(stopChan)
	for {
		go updateRoles()
		time.Sleep(time.Second * 30)
	}
}

func handleSigterm(stopChan chan struct{}) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals
	log.Println("Received SIGTERM. Terminating...")
	close(stopChan)
}

func serveMetrics(address *string) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	prometheus.MustRegister(roleUpdates)
	prometheus.MustRegister(roleUpdateErrors)
	http.Handle("/metrics", promhttp.Handler())

	log.Printf("Server listing %v\n", *address)
	log.Fatal(http.ListenAndServe(*address, nil))
}

func updateRoles() {
	roleUpdates.WithLabelValues("updates").Inc()
	log.Println("Role updated!")
}
