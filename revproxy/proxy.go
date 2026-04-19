package revproxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/sileader/llama-gateway/model"
)

type Proxy struct {
	target  *url.URL
	reverse *httputil.ReverseProxy
	dl      *model.Downloader
	config  ServerConfig
}

func NewProxy(config ServerConfig, targetURL string, dl *model.Downloader) *Proxy {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil
	}
	if target == nil {
		panic("invalid target URL")
	}
	return &Proxy{
		target:  target,
		reverse: httputil.NewSingleHostReverseProxy(target),
		dl:      dl,
		config:  config,
	}
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
			p.addModel(w, r)
			return
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Add models is not enabled."))
			return
		}
	}
}
