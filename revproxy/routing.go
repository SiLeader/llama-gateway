package revproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

type ModelError int

const (
	ModelNotFound       ModelError = iota
	ModelBadRequest     ModelError = iota
	ModelLoadError      ModelError = iota
	ModelSerializeError ModelError = iota
)

func (p *Proxy) rewriteEntry(w http.ResponseWriter, r *http.Request) bool {
	//ctx := r.Context()
	//
	//if p.rewriteOpenAI(ctx, w, r) {
	//	return true
	//}
	//if p.rewriteAnthropic(ctx, w, r) {
	//	return true
	//}

	return false
}

func (p *Proxy) rewriteModelHelper(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	responseError func(m ModelError, w http.ResponseWriter) bool,
) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.DebugContext(ctx, "Failed to read request body", "error", err)
		return responseError(ModelBadRequest, w)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.DebugContext(ctx, "Failed to unmarshal request body", "error", err)
		return responseError(ModelBadRequest, w)
	}

	if model, ok := payload["model"].(string); ok {
		resolved := p.mapper.UseModel(model)
		if resolved != nil {
			payload["model"] = resolved
		} else {
			slog.DebugContext(ctx, "Model not found", "model", model)
			return responseError(ModelNotFound, w)
		}
	}

	newBody, err := json.Marshal(payload)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal payload", "error", err)
		return responseError(ModelSerializeError, w)
	}
	r.Body = io.NopCloser(bytes.NewReader(newBody))
	r.ContentLength = int64(len(newBody))
	return false
}
