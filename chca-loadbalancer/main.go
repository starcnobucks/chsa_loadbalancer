package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"chca-loadbalancer/backend"
	"chca-loadbalancer/balancer"
	"chca-loadbalancer/config"
	"chca-loadbalancer/hashring"
	"chca-loadbalancer/logger"
)

func main() {
	// ── Load configuration ──────────────────────────────────────────────
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	// ── Initialise logger ───────────────────────────────────────────────
	logger.Init(cfg.Logging.Level)
	logger.Log.Info("starting CHCA load balancer")

	// ── Create backends ─────────────────────────────────────────────────
	backends := make([]*backend.Backend, 0, len(cfg.Backends))
	for _, bc := range cfg.Backends {
		b, err := backend.NewBackend(bc)
		if err != nil {
			logger.Log.WithError(err).Fatal("failed to create backend")
		}
		backends = append(backends, b)
		logger.Log.WithField("url", b.Address()).Info("registered backend")
	}

	// ── Initialise hash ring ────────────────────────────────────────────
	ring := hashring.NewRing(cfg.Ring.VirtualNodes)
	for _, b := range backends {
		ring.Add(b.Address())
	}
	logger.Log.WithField("members", ring.Members()).Info("hash ring initialised")

	// ── Start background monitor ────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor := backend.NewMonitor(backends, ring, cfg)
	monitor.Start(ctx)

	// ── Build the CHCA balancer ─────────────────────────────────────────
	lb := balancer.NewCHCABalancer(backends, ring, cfg)

	// ── HTTP server ─────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.Handle("/", lb)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful shutdown ───────────────────────────────────────────────
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Log.WithField("signal", sig.String()).Info("shutting down")

		cancel() // stop monitor goroutines

		shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutCancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			logger.Log.WithError(err).Error("server shutdown error")
		}
	}()

	logger.Log.WithField("addr", addr).Info("listening")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Log.WithError(err).Fatal("server error")
	}
	logger.Log.Info("server stopped")
}
