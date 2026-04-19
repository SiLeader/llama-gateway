package revproxy

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (p *Proxy) ListenAndServe() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:    p.config.ListenAddress(),
		Handler: accessLog(p),
	}
	slog.Info("Starting reverse proxy", "url", p.target, "port", p.config.ListenPort(), "host", p.config.ListenHost())
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln("Failed to start reverse proxy", err)
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down reverse proxy")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	slog.Info("Reverse proxy stopped")
	return nil
}
