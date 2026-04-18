package revproxy

import (
	"encoding/json"
	"net/http"

	"github.com/sileader/llama-gateway/model"
)

func (p *Proxy) addModel(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var info model.Info

	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := p.dl.AddModel(r.Context(), info); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
