package revproxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sileader/llama-gateway/model"
)

func testMapper() *model.Mapper {
	return model.NewModelMapper([]model.Info{
		{Name: "test-model", Id: "org/test", File: "test.gguf"},
	}, "/models")
}

func testDownloader() *model.Downloader {
	return model.NewDownloader([]model.Info{}, "/models", "", nil, nil)
}

func TestNewProxy_ValidURL(t *testing.T) {
	p := NewProxy("http://localhost:8081", testMapper(), testDownloader())
	if p == nil {
		t.Fatal("expected non-nil Proxy")
	}
}

func TestNewProxy_InvalidURL(t *testing.T) {
	// url.Parse fails on strings containing control characters
	p := NewProxy("http://[::1]:namedport", testMapper(), testDownloader())
	if p != nil {
		t.Errorf("expected nil Proxy for invalid URL, got non-nil")
	}
}

func TestServeHTTP(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	p := NewProxy(backend.URL, testMapper(), testDownloader())
	if p == nil {
		t.Fatal("failed to create proxy")
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/responses", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "backend response" {
		t.Errorf("body = %q, want %q", string(body), "backend response")
	}
}

func TestServeHTTP_HeaderForwarding(t *testing.T) {
	const customHeader = "X-Test-Header"
	const customValue = "hello-proxy"

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get(customHeader)
		if got != customValue {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing header"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("header received"))
	}))
	defer backend.Close()

	p := NewProxy(backend.URL, testMapper(), testDownloader())
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set(customHeader, customValue)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		body, _ := io.ReadAll(rec.Body)
		t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, body)
	}
}
