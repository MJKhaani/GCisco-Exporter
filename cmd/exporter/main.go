package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mjkoot/GCiscoExporter/internal/cache"
	"github.com/mjkoot/GCiscoExporter/internal/config"
	"github.com/mjkoot/GCiscoExporter/internal/debugcapture"
	"github.com/mjkoot/GCiscoExporter/internal/metrics"
	"github.com/mjkoot/GCiscoExporter/internal/scheduler"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
	listenAddr = flag.String("listen", ":9427", "Address to listen on")
)

func main() {
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Loaded configuration with %d devices", len(cfg.Devices))

	var dc *debugcapture.Capture
	if cfg.DebugCapture.Enabled {
		dc, err = debugcapture.New(cfg.DebugCapture.Path)
		if err != nil {
			log.Fatalf("Failed to initialize debug capture: %v", err)
		}
	}

	mc := metrics.NewCollector()
	store := cache.New()

	sched := scheduler.New(cfg, store, mc, dc)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched.Start(ctx)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(mc.Registry(), promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	srv := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Starting exporter on %s", *listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
}
