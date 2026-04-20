package revproxy

import (
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

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
		log.Fatalln("invalid target URL")
	}
	adminKey, err := config.AdminKey()
	if err != nil {
		return nil, err
	}
	p := &Proxy{
		target:   target,
		reverse:  httputil.NewSingleHostReverseProxy(target),
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
			if len(keys) == 0 || keys[0] != p.adminKey {
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
