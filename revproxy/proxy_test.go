package revproxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sileader/llama-gateway/model"
)

func testDownloader() *model.Downloader {
	dl, err := model.NewDownloader([]model.Info{}, "/models", "", nil, nil)
	if err != nil {
		panic(err)
	}
	return dl
}

func defaultConfig(targetURL string) (ServerConfig, string) {
	return ServerConfig{}, targetURL
}

func TestNewProxy_ValidURL(t *testing.T) {
	cfg, url := defaultConfig("http://localhost:8081")
	p, err := NewProxy(cfg, url, testDownloader(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil Proxy")
	}
}

func TestNewProxy_InvalidURL(t *testing.T) {
	cfg, _ := defaultConfig("")
	// url.Parse fails on strings containing control characters
	p, _ := NewProxy(cfg, "http://[::1]:namedport", testDownloader(), nil)
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

	cfg, _ := defaultConfig(backend.URL)
	p, err := NewProxy(cfg, backend.URL, testDownloader(), nil)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
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

	cfg, _ := defaultConfig(backend.URL)
	p, err := NewProxy(cfg, backend.URL, testDownloader(), nil)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set(customHeader, customValue)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		body, _ := io.ReadAll(rec.Body)
		t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, body)
	}
}

func proxyWithAdminKey(t *testing.T, targetURL, adminKey string) *Proxy {
	t.Helper()
	t.Setenv("TEST_ADMIN_KEY", adminKey)
	envName := "TEST_ADMIN_KEY"
	cfg := ServerConfig{
		Apis: api{AddModels: true},
		Auth: auth{AdminKeyEnv: &envName},
	}
	p, err := NewProxy(cfg, targetURL, testDownloader(), nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	return p
}

func TestHandleGatewayApi_Forbidden_NoKey(t *testing.T) {
	p := proxyWithAdminKey(t, "http://localhost:9999", "mysecret")
	req := httptest.NewRequest(http.MethodPost, "/gateway/v1/models", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHandleGatewayApi_Forbidden_WrongKey(t *testing.T) {
	p := proxyWithAdminKey(t, "http://localhost:9999", "mysecret")
	req := httptest.NewRequest(http.MethodPost, "/gateway/v1/models", nil)
	req.Header.Set("Authorization", "Bearer wrongkey")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHandleGatewayApi_AddModels_Disabled(t *testing.T) {
	cfg := ServerConfig{Apis: api{AddModels: false}}
	p, err := NewProxy(cfg, "http://localhost:9999", testDownloader(), nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/gateway/v1/models", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (disabled gateway API)", rec.Code, http.StatusNotFound)
	}
}

func TestHandleGatewayApi_UnknownPath(t *testing.T) {
	cfg := ServerConfig{Apis: api{AddModels: false}}
	p, err := NewProxy(cfg, "http://localhost:9999", testDownloader(), nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/gateway/unknown", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleGatewayApi_AddModel_ValidKey(t *testing.T) {
	// Downloader with existing model: AddModel returns nil without downloading
	existing := model.Info{Name: "existing", Id: "org/existing", File: "existing.gguf"}
	dl, err := model.NewDownloader([]model.Info{existing}, "/tmp", "/tmp/presets.ini", nil, nil)
	if err != nil {
		t.Fatalf("NewDownloader: %v", err)
	}
	t.Setenv("TEST_ADMIN_KEY", "secret")
	envName := "TEST_ADMIN_KEY"
	cfg := ServerConfig{
		Apis: api{AddModels: true},
		Auth: auth{AdminKeyEnv: &envName},
	}
	p, err := NewProxy(cfg, "http://localhost:9999", dl, nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}

	body, _ := json.Marshal(existing)
	req := httptest.NewRequest(http.MethodPost, "/gateway/v1/models", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestHandleGatewayApi_AddModel_InvalidBody(t *testing.T) {
	p := proxyWithAdminKey(t, "http://localhost:9999", "secret")
	req := httptest.NewRequest(http.MethodPost, "/gateway/v1/models", strings.NewReader("not json"))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleGatewayApi_AddModel_InvalidModelName(t *testing.T) {
	p := proxyWithAdminKey(t, "http://localhost:9999", "secret")
	body := `{"name":"bad name!","id":"org/repo","file":"model.gguf"}`
	req := httptest.NewRequest(http.MethodPost, "/gateway/v1/models", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for invalid model name", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleGatewayApi_AddModel_PathTraversal(t *testing.T) {
	p := proxyWithAdminKey(t, "http://localhost:9999", "secret")
	body := `{"name":"safe","id":"org/repo","file":"../../../etc/passwd"}`
	req := httptest.NewRequest(http.MethodPost, "/gateway/v1/models", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for path traversal", rec.Code, http.StatusBadRequest)
	}
}

func TestBadGatewayError_MessagesPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	rec := httptest.NewRecorder()
	badGatewayError(rec, req, nil)
	if rec.Code != 529 {
		t.Errorf("status = %d, want 529 for /v1/messages", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestBadGatewayError_OtherPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()
	badGatewayError(rec, req, nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}
