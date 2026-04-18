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
	dl      *model.Downloader
}

func NewProxy(targetURL string, mapper *model.Mapper, dl *model.Downloader) *Proxy {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil
	}
	if target == nil {
		panic("invalid target URL")
	}
	return &Proxy{
		target:  target,
		mapper:  mapper,
		reverse: httputil.NewSingleHostReverseProxy(target),
		dl:      dl,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/gateway/v1/models" {
		p.addModel(w, r)
		return
	}
	slog.DebugContext(r.Context(), "Reverse proxying request", "url", r.URL)
	p.reverse.ServeHTTP(w, r)
}
