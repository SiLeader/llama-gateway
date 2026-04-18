package huggingface

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("hf_mytoken")
	if c == nil {
		t.Fatal("expected non-nil Client")
	}
	if c.token != "hf_mytoken" {
		t.Errorf("token = %q, want %q", c.token, "hf_mytoken")
	}
	if c.baseURL != "https://huggingface.co" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://huggingface.co")
	}
	if c.httpClient != http.DefaultClient {
		t.Error("httpClient should default to http.DefaultClient")
	}
}

func TestNewClient_EmptyToken(t *testing.T) {
	c := NewClient("")
	if c == nil {
		t.Fatal("expected non-nil Client")
	}
	if c.token != "" {
		t.Errorf("token = %q, want empty", c.token)
	}
}

func TestClientDownload(t *testing.T) {
	const repo = "org/mymodel"
	const filename = "weights.gguf"
	fileContent := []byte("fake model weights for testing")
	expectedSHA := sha256hex(fileContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/models/" + repo + "/revision/main":
			resp := modelInfo{
				Siblings: []fileInfo{
					{RFilename: filename, SHA256: expectedSHA},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/" + repo + "/resolve/main/" + filename:
			w.Write(fileContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, filename)

	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	if err := c.Download(context.Background(), repo, filename, destPath); err != nil {
		t.Fatalf("Download: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(fileContent) {
		t.Errorf("file content = %q, want %q", got, fileContent)
	}
}

func TestClientDownload_MetadataError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	err := c.Download(context.Background(), "org/model", "f.gguf", filepath.Join(dir, "f.gguf"))
	if err == nil {
		t.Fatal("expected error when metadata fetch fails, got nil")
	}
}
