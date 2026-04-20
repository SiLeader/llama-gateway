package revproxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"
)

func (p *Proxy) ListenAndServe(shutdownCtx context.Context) error {
	srv := &http.Server{
		Addr:         p.config.ListenAddress(),
		Handler:      accessLog(p),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Minute,
		ErrorLog:     log.Default(),
	}
	slog.Info("Starting reverse proxy", "url", p.target, "port", p.config.ListenPort(), "host", p.config.ListenHost())

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("reverse proxy listen failed: %w", err)
		}
		return nil
	case <-shutdownCtx.Done():
	}

	slog.Info("Shutting down reverse proxy")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	slog.Info("Reverse proxy stopped")
	return nil
}
