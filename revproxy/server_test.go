package revproxy

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sileader/llama-gateway/model"
)

func TestListenAndServe_PortAlreadyInUse(t *testing.T) {
	// Occupy a port first
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	dl, _ := model.NewDownloader([]model.Info{}, "/tmp", "/tmp/presets.ini", nil, nil)
	cfg := ServerConfig{Listen: listen{Host: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port}}
	p, err := NewProxy(cfg, "http://localhost:9999", dl)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	_ = addr

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = p.ListenAndServe(ctx)
	if err == nil {
		t.Error("expected error when port is already in use")
	}
}

func TestListenAndServe_GracefulShutdown(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	dl, _ := model.NewDownloader([]model.Info{}, "/tmp", "/tmp/presets.ini", nil, nil)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	cfg := ServerConfig{Listen: listen{Host: "127.0.0.1", Port: port}}
	p, err := NewProxy(cfg, backend.URL, dl)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- p.ListenAndServe(ctx)
	}()

	// Give the server time to start
	time.Sleep(50 * time.Millisecond)

	// Trigger graceful shutdown
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ListenAndServe returned error on graceful shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ListenAndServe did not shut down in time")
	}
}
