// Command api runs the Online Tickets HTTP API, the booking expiry job and the mock
// payment provider page.
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Vanady/online-tickets/backend/internal/app"
	"github.com/Vanady/online-tickets/backend/internal/courier"
	"github.com/Vanady/online-tickets/backend/internal/httpapi"
	"github.com/Vanady/online-tickets/backend/internal/payment"
	"github.com/Vanady/online-tickets/backend/internal/seed"
	"github.com/Vanady/online-tickets/backend/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// secret reads a secret from the environment or, for local runs, generates a random one
// (tokens and payment signatures then do not survive a restart).
func secret(log *slog.Logger, key string) []byte {
	if v := os.Getenv(key); v != "" {
		return []byte(v)
	}
	log.Warn("secret not set, using a random one", "env", key)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	addr := env("ADDR", ":8080")
	publicURL := env("PUBLIC_URL", "http://localhost:8080") // where browsers reach the mock provider page
	webURL := env("WEB_URL", "http://localhost:5173")       // frontend, for returning after payment

	st, err := openWithRetry(ctx, log, dbURL)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	now := time.Now
	accounts := &app.AccountService{Repo: st, JWTSecret: secret(log, "JWT_SECRET"), Now: now}
	pay := payment.NewMock(secret(log, "PAYMENT_SECRET"), publicURL,
		func(orderID int64) string { return fmt.Sprintf("%s/orders/%d", webURL, orderID) }, log)
	orders := &app.OrderService{Repo: st, Payment: pay, Courier: &courier.Mock{Log: log}, Now: now, Log: log}
	pay.OnResult = orders.ConfirmPayment

	if env("SEED_DEMO", "false") == "true" {
		if err := seed.Run(ctx, st, accounts, env("SEED_PASSWORD", "demo-pass-123"), now()); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
	}

	expiry := &app.ExpiryJob{Repo: st, Now: now, Log: log}
	go expiry.Run(ctx, time.Minute)

	srv := &http.Server{
		Addr: addr,
		Handler: (&httpapi.Server{
			Accounts:  accounts,
			Catalog:   &app.CatalogService{Repo: st, Now: now},
			Orders:    orders,
			BoxOffice: &app.BoxOfficeService{Repo: st, Now: now},
			Back:      &app.BackOfficeService{Repo: st, Now: now},
			Extra:     pay.Handler(),
			Log:       log,
		}).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// openWithRetry waits up to 30 s for the database (it may still be starting in Docker).
func openWithRetry(ctx context.Context, log *slog.Logger, url string) (*store.Store, error) {
	var lastErr error
	for i := 0; i < 30; i++ {
		st, err := store.Open(ctx, url)
		if err == nil {
			return st, nil
		}
		lastErr = err
		log.Info("waiting for database", "err", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, fmt.Errorf("database: %w", lastErr)
}
