package revproxy

import (
	"context"
	"net/http"

	"github.com/sileader/llama-gateway/revproxy/route"
)

var llamaServerUrl = route.NewMatch(
	route.PostExact("/models/load"),
	route.PostExact("/models/unload"),
)

func (p *Proxy) rewriteLlamaServer(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	if !llamaServerUrl.IsMatch(r) {
		return false
	}

	return p.rewriteModelHelper(ctx, w, r, func(m ModelError, w http.ResponseWriter) bool {
		return false
	})
}
