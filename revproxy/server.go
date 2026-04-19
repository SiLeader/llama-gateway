package revproxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (p *Proxy) ListenAndServe(host string, port int) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: accessLog(p),
	}
	slog.Info("Starting reverse proxy", "url", p.target, "port", port, "host", host)
	go srv.ListenAndServe()

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
