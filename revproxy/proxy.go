package revproxy

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sileader/llama-gateway/model"
)

// UpstreamHandle represents a retired upstream that can be drained and closed.
type UpstreamHandle interface {
	Drain(ctx context.Context) error
	CloseIdleConnections()
}

// upstream holds a single llama-server backend, its transport, and an in-flight
// request counter. Each upstream owns its Transport so idle connections can be
// closed when it is retired.
type upstream struct {
	target    *url.URL
	reverse   *httputil.ReverseProxy
	transport *http.Transport
	wg        sync.WaitGroup
}

func (u *upstream) Drain(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		u.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (u *upstream) CloseIdleConnections() {
	u.transport.CloseIdleConnections()
}

// Reloader is implemented by orchestrator.Orchestrator to handle reload requests.
type Reloader interface {
	Reload(ctx context.Context) error
}

// Proxy is an HTTP reverse-proxy that supports zero-downtime upstream swaps.
type Proxy struct {
	current  atomic.Pointer[upstream]
	dl       *model.Downloader
	config   ServerConfig
	adminKey string
	reloader Reloader
}

func NewProxy(config ServerConfig, targetURL string, dl *model.Downloader, reloader Reloader) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, fmt.Errorf("invalid target url")
	}
	adminKey := ""
	if config.Apis.IsAdminApiEnabled() {
		adminKey, err = config.AdminKey()
		if err != nil {
			return nil, err
		}
	}

	p := &Proxy{
		dl:       dl,
		config:   config,
		adminKey: adminKey,
		reloader: reloader,
	}
	p.SetUpstream(target)
	return p, nil
}

func newUpstream(target *url.URL) *upstream {
	transport := &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          128,
		MaxIdleConnsPerHost:   128,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	pxy := httputil.NewSingleHostReverseProxy(target)
	pxy.ErrorLog = log.Default()
	pxy.FlushInterval = -1
	pxy.Transport = transport
	pxy.ErrorHandler = badGatewayError
	return &upstream{
		target:    target,
		reverse:   pxy,
		transport: transport,
	}
}

// SetReloader sets the Reloader after construction.
func (p *Proxy) SetReloader(r Reloader) {
	p.reloader = r
}

// SetDownloader sets the model.Downloader after construction.
func (p *Proxy) SetDownloader(dl *model.Downloader) {
	p.dl = dl
}

// SetUpstream atomically replaces the active upstream and returns the previous
// one so the caller can drain and close it. Returns nil if there was no previous
// upstream.
func (p *Proxy) SetUpstream(target *url.URL) UpstreamHandle {
	u := newUpstream(target)
	old := p.current.Swap(u)
	slog.Info("Upstream set", "url", target)
	if old == nil {
		return nil
	}
	return old
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/gateway/") {
		p.handleGatewayApi(w, r)
		return
	}
	u := p.current.Load()
	u.wg.Add(1)
	defer u.wg.Done()
	slog.DebugContext(r.Context(), "Reverse proxying request", "url", r.URL)
	u.reverse.ServeHTTP(w, r)
}

func (p *Proxy) handleGatewayApi(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/gateway/v1/models" {
		if p.config.Apis.AddModels {
			if !p.checkAdminKey(w, r) {
				return
			}
			p.addModel(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Add models is not enabled."))
		return
	}
	if r.Method == http.MethodPost && r.URL.Path == "/gateway/v1/reload" {
		if p.reloader == nil || !p.config.Apis.Reload {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Reload is not enabled."))
			return
		}
		if !p.checkAdminKey(w, r) {
			return
		}
		p.handleReload(w, r)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not found"))
}

func (p *Proxy) checkAdminKey(w http.ResponseWriter, r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == auth || subtle.ConstantTimeCompare([]byte(token), []byte(p.adminKey)) != 1 {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("operation not allowed"))
		return false
	}
	return true
}

func badGatewayError(w http.ResponseWriter, r *http.Request, err error) {
	if strings.HasPrefix(r.URL.Path, "/v1/messages") {
		// Anthropic Message API
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(529)
		w.Write([]byte(`{"type":"error", "error":{"type":"overloaded_error","message":"Backend server is temporary unavailable"}}`))
	} else {
		// Maybe OpenAI API
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":{"message":"Backend server is temporary unavailable","type":"service_unavailable_error"}}`))
	}
}
