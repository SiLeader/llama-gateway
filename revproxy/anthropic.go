package revproxy

import (
	"context"
	"net/http"

	"github.com/sileader/llama-gateway/revproxy/route"
)

var anthropicUrl = route.NewMatch(
	route.PostPrefix("/v1/messages"),
)

func (p *Proxy) rewriteAnthropic(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	if !anthropicUrl.IsMatch(r) {
		return false
	}

	return p.rewriteModelHelper(ctx, w, r, func(m ModelError, w http.ResponseWriter) bool {
		switch m {
		case ModelBadRequest:
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"Invalid request body"}}`))
			return true
		case ModelNotFound:
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"type":"error","error":{"type":"not_found_error","message":"Model not found"}}`))
			return true
		case ModelLoadError:
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"Model load error"}}`))
			return true
		case ModelSerializeError:
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"Model serialization error"}}`))
			return true
		default:
			return false
		}
	})
}

type anthropicError struct {
	Type  string                `json:"type"`
	Error anthropicErrorContent `json:"error"`
}

type anthropicErrorContent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
