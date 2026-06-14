package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dollabilliam/github-actions-self-hosted-runner-exporter/internal/collector"
	"github.com/Dollabilliam/github-actions-self-hosted-runner-exporter/internal/config"
	gh "github.com/Dollabilliam/github-actions-self-hosted-runner-exporter/internal/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config.json", "path to exporter config")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	token := os.Getenv(cfg.GitHub.TokenEnv)
	if token == "" {
		logger.Error("missing GitHub token", "env", cfg.GitHub.TokenEnv)
		os.Exit(1)
	}

	client, err := gh.NewClient(gh.ClientConfig{
		BaseURL:   cfg.GitHub.BaseURL,
		Token:     token,
		Timeout:   cfg.GitHub.Timeout.Duration,
		UserAgent: "github-actions-self-hosted-runner-exporter/" + version,
	})
	if err != nil {
		logger.Error("create GitHub client", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	actionsCollector := collector.New(collector.Config{
		Client:            client,
		Repositories:      cfg.Repositories,
		RefreshInterval:   cfg.Scrape.RefreshInterval.Duration,
		RunsPerRepo:       cfg.Scrape.RunsPerRepository,
		Logger:            logger,
		CollectionTimeout: 2 * time.Minute,
	})
	actionsCollector.Start(ctx)

	registry := prometheus.NewRegistry()
	registry.MustRegister(actionsCollector)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	server := &http.Server{
		Addr:              cfg.Server.ListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("starting exporter", "listen_address", cfg.Server.ListenAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "err", err)
		os.Exit(1)
	}
}
