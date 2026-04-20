package revproxy

import (
	"crypto/subtle"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/sileader/llama-gateway/model"
)

type Proxy struct {
	target   *url.URL
	reverse  *httputil.ReverseProxy
	dl       *model.Downloader
	config   ServerConfig
	adminKey string
}

func NewProxy(config ServerConfig, targetURL string, dl *model.Downloader) (*Proxy, error) {
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

	pxy := httputil.NewSingleHostReverseProxy(target)
	pxy.ErrorLog = log.Default()
	pxy.FlushInterval = -1
	pxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          128,
		MaxIdleConnsPerHost:   128,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	pxy.ErrorHandler = badGatewayError

	p := &Proxy{
		target:   target,
		reverse:  pxy,
		dl:       dl,
		config:   config,
		adminKey: adminKey,
	}
	return p, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/gateway/") {
		p.handleGatewayApi(w, r)
		return
	}
	slog.DebugContext(r.Context(), "Reverse proxying request", "url", r.URL)
	p.reverse.ServeHTTP(w, r)
}

func (p *Proxy) handleGatewayApi(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/gateway/v1/models" {
		if p.config.Apis.AddModels {
			keys := r.Header.Values("X-Llama-Gateway-Api-Key")
			if len(keys) == 0 || subtle.ConstantTimeCompare([]byte(keys[0]), []byte(p.adminKey)) != 1 {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("operation not allowed"))
				return
			}
			p.addModel(w, r)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Add models is not enabled."))
		return
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not found"))
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
