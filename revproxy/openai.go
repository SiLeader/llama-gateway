package revproxy

import (
	"context"
	"net/http"

	"github.com/sileader/llama-gateway/revproxy/route"
)

var openAiUrl = route.NewMatch(
	route.PostExact("/v1/chat/completions"),
	route.PostExact("/v1/responses"),
	route.PostExact("/v1/embeddings"),
)

func (p *Proxy) rewriteOpenAI(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	if !openAiUrl.IsMatch(r) {
		return false
	}

	return p.rewriteModelHelper(ctx, w, r, func(m ModelError, w http.ResponseWriter) bool {
		switch m {
		case ModelBadRequest:
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"error":{"message":"Invalid request body","type":"invalid_request_error","param":null,"code":"invalid_request"}}`))
			return true
		case ModelNotFound:
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"error":{"message":"Model not found","type":"not_found_error","param":null,"code":"not_found"}}`))
			return true
		case ModelLoadError:
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"error":{"message":"Failed to load model","type":"internal_server_error","param":null,"code":"internal_server_error"}}`))
			return true
		case ModelSerializeError:
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"error":{"message":"Failed to serialize model","type":"internal_server_error","param":null,"code":"internal_server_error"}}`))
			return true
		default:
			return false
		}
	})
}
