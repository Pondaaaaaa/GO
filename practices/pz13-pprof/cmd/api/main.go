package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz13-pprof/internal/api"
)

// Overridden at link time to build the before and after variants from one source.
var defaultMode = "fast"

func main() {
	port := flag.Int("port", 0, "loopback port; 0 chooses a free port")
	n := flag.Int("n", 38, "Fib input, 0..40")
	flag.Parse()
	if *port < 0 || *port > 65535 || *n < 0 || *n > 40 {
		log.Fatal("invalid port or n")
	}
	if defaultMode != "slow" && defaultMode != "fast" {
		log.Fatal("invalid linked mode")
	}
	serve(*port, *n)
}

func serve(port, n int) {
	listener, err := listenLoopback(port)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("mode=%s n=%d listening=%s", defaultMode, n, listener.Addr())
	srv := &http.Server{Handler: api.NewMux(defaultMode, n), ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.Serve(listener))
}
