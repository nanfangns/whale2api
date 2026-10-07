package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"whale2api/internal/config"
	"whale2api/internal/poolui"
	"whale2api/internal/server"
)

func main() {
	if err := run(); err != nil {
		config.Logger.Error("Vercel service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	config.RefreshLogger()
	for _, name := range []string{
		"WHALE2API_DATABASE_URL",
		"POOL_UI_ADMIN_TOKEN",
		"DEVICE_HARVEST_URL",
		"DEVICE_HARVEST_TOKEN",
	} {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			return fmt.Errorf("required Vercel environment variable %s is not set", name)
		}
	}

	app, err := server.NewApp()
	if err != nil {
		return fmt.Errorf("initialize gateway: %w", err)
	}
	defer app.Close()

	admin, err := poolui.NewServer(app.PoolDB)
	if err != nil {
		return fmt.Errorf("initialize pool UI: %w", err)
	}
	handler := vercelHandler{api: app.Router, admin: admin.Router}
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "80"
	}
	srv := &http.Server{
		Addr:              "0.0.0.0:" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	config.Logger.Info("starting Vercel container service", "addr", srv.Addr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

type vercelHandler struct {
	api   http.Handler
	admin http.Handler
}

func (h vercelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/", "/api", "/app.css", "/app.js", "/favicon.ico":
		h.admin.ServeHTTP(w, r)
	case "/admin", "/admin/":
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	default:
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.admin.ServeHTTP(w, r)
			return
		}
		h.api.ServeHTTP(w, r)
	}
}
