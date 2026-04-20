package revproxy

import (
	"log/slog"
	"net/http"
)

func (p *Proxy) handleReload(w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(r.Context(), "Reload requested via admin API")
	if err := p.reloader.Reload(r.Context()); err != nil {
		slog.ErrorContext(r.Context(), "Reload failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
