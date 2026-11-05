package main

import (
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
)

func init() {
	addr := os.Getenv("PPROF_ADDR")
	if addr == "" {
		return
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		log.Fatalf("invalid PPROF_ADDR %q: %v", addr, err)
	}
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		log.Fatalf("PPROF_ADDR must use a loopback host, got %q", host)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen for pprof on %s: %v", addr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	for _, name := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		mux.Handle("GET /debug/pprof/"+name, pprof.Handler(name))
	}

	go func() {
		log.Printf("temporary pprof listener on http://%s/debug/pprof/", addr)
		if err := http.Serve(listener, mux); err != nil {
			log.Printf("pprof listener stopped: %v", err)
		}
	}()
}
