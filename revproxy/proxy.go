package revproxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/sileader/llama-gateway/model"
)

type Proxy struct {
	target  *url.URL
	mapper  *model.Mapper
	reverse *httputil.ReverseProxy
}

func NewProxy(targetURL string, mapper *model.Mapper) *Proxy {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil
	}
	return &Proxy{
		target:  target,
		mapper:  mapper,
		reverse: httputil.NewSingleHostReverseProxy(target),
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.DebugContext(r.Context(), "Reverse proxying request", "url", r.URL)
	p.reverse.ServeHTTP(w, r)
}
