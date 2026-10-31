package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/anniversaries"
	"github.com/Liao-Jun-Tao/couple-daily/internal/auth"
	"github.com/Liao-Jun-Tao/couple-daily/internal/checkins"
	"github.com/Liao-Jun-Tao/couple-daily/internal/config"
	"github.com/Liao-Jun-Tao/couple-daily/internal/media"
	"github.com/Liao-Jun-Tao/couple-daily/internal/pairing"
	"github.com/Liao-Jun-Tao/couple-daily/internal/posts"
	"github.com/Liao-Jun-Tao/couple-daily/internal/push"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load()
	if cfg.JWTSecret == config.DefaultJWTSecret {
		log.Println("warning: JWT_SECRET is the example default; set it before any shared deployment")
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           newRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("couple-daily api listening on %s", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}
}

func newRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", healthz)
	r.Route("/v1", func(r chi.Router) {
		auth.Mount(r)
		pairing.Mount(r)
		media.Mount(r)
		posts.Mount(r)
		checkins.Mount(r)
		anniversaries.Mount(r)
		push.Mount(r)
	})
	return r
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
