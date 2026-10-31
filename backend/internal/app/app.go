// Package app wires the HTTP router to PostgreSQL, Redis, object storage, and push.
package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/anniversaries"
	"github.com/Liao-Jun-Tao/couple-daily/internal/auth"
	"github.com/Liao-Jun-Tao/couple-daily/internal/checkins"
	"github.com/Liao-Jun-Tao/couple-daily/internal/config"
	"github.com/Liao-Jun-Tao/couple-daily/internal/dbmigrate"
	"github.com/Liao-Jun-Tao/couple-daily/internal/media"
	"github.com/Liao-Jun-Tao/couple-daily/internal/objectstore"
	"github.com/Liao-Jun-Tao/couple-daily/internal/pairing"
	"github.com/Liao-Jun-Tao/couple-daily/internal/posts"
	"github.com/Liao-Jun-Tao/couple-daily/internal/push"
	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
	"github.com/Liao-Jun-Tao/couple-daily/migrations"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

// Services are the mounted HTTP handlers.
type Services struct {
	Auth          *auth.Handler
	Pairing       *pairing.Handler
	Media         *media.Handler
	Posts         *posts.Handler
	Checkins      *checkins.Handler
	Anniversaries *anniversaries.Handler
	Devices       *push.Handler
}

// App is a running API process.
type App struct {
	Handler http.Handler
	DB      *store.DB
	cancel  context.CancelFunc
	rdb     *redis.Client
	wg      sync.WaitGroup
}

// Router mounts health and /v1 contract routes.
func Router(s Services) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/healthz", healthz)
	r.Route("/v1", func(r chi.Router) {
		auth.Mount(r, s.Auth)
		pairing.Mount(r, s.Pairing)
		media.Mount(r, s.Media)
		posts.Mount(r, s.Posts)
		checkins.Mount(r, s.Checkins)
		anniversaries.Mount(r, s.Anniversaries)
		push.Mount(r, s.Devices)
	})
	return r
}

// UnconnectedRouter mounts the contract without a database.
// It is used to prove routes answer 401/400 instead of 501.
func UnconnectedRouter() http.Handler {
	const secret = "test-secret"
	authn := auth.Middleware(secret)
	signer := objectstore.Unsigned{Endpoint: "http://localhost:9000", Bucket: "couple-daily"}
	return Router(Services{
		Auth:          &auth.Handler{Issuer: auth.NewIssuer(secret, time.Hour), Secret: secret, AccessTTL: time.Hour, RefreshTTL: 720 * time.Hour},
		Pairing:       &pairing.Handler{Authn: authn},
		Media:         &media.Handler{Authn: authn, Signer: signer},
		Posts:         &posts.Handler{Authn: authn, Signer: signer},
		Checkins:      &checkins.Handler{Authn: authn},
		Anniversaries: &anniversaries.Handler{Authn: authn},
		Devices:       &push.Handler{Authn: authn},
	})
}

// New migrates the database and starts the reminder loop.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.AllowsInsecureDev() && (strings.EqualFold(cfg.JWTSecret, config.DefaultJWTSecret) || len(cfg.JWTSecret) < config.MinJWTSecretLen) {
		log.Printf("warning: APP_ENV=%s is using a weak JWT_SECRET; do not expose this process beyond local development", cfg.AppEnv)
	}
	if err := dbmigrate.Apply(ctx, cfg.DatabaseURL, migrations.Files); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	db, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	rdb := openRedis(cfg.RedisURL)
	signer := objectstore.New(cfg)
	authn := auth.Middleware(cfg.JWTSecret)
	notifier := push.NewNotifier(cfg)
	services := Services{
		Auth: &auth.Handler{
			DB:         db,
			Issuer:     auth.NewIssuer(cfg.JWTSecret, cfg.JWTAccessTTL),
			Secret:     cfg.JWTSecret,
			AccessTTL:  cfg.JWTAccessTTL,
			RefreshTTL: cfg.JWTRefreshTTL,
			Redis:      rdb,
			Login:      auth.NewLoginGate(),
		},
		Pairing:       &pairing.Handler{DB: db, Authn: authn},
		Media:         &media.Handler{DB: db, Authn: authn, Signer: signer},
		Posts:         &posts.Handler{DB: db, Authn: authn, Signer: signer},
		Checkins:      &checkins.Handler{DB: db, Authn: authn},
		Anniversaries: &anniversaries.Handler{DB: db, Authn: authn},
		Devices:       &push.Handler{DB: db, Authn: authn},
	}
	workerCtx, cancel := context.WithCancel(context.Background())
	application := &App{Handler: Router(services), DB: db, cancel: cancel, rdb: rdb}
	var dedupe push.Dedupe = &push.MemoryDedupe{}
	if rdb != nil {
		dedupe = push.RedisDedupe{Client: rdb}
	}
	application.wg.Add(1)
	go func() {
		defer application.wg.Done()
		push.RunReminders(workerCtx, db, notifier, dedupe)
	}()
	return application, nil
}

// Close stops the reminder loop and closes connections.
func (a *App) Close() {
	if a == nil {
		return
	}
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
	if a.rdb != nil {
		_ = a.rdb.Close()
	}
	if a.DB != nil {
		a.DB.Close()
	}
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func openRedis(raw string) *redis.Client {
	opt, err := redis.ParseURL(raw)
	if err != nil {
		log.Printf("redis disabled: %v", err)
		return nil
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("redis unavailable, auth continues without it: %v", err)
		_ = client.Close()
		return nil
	}
	log.Printf("redis connected")
	return client
}
