package huggingface

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func sha256hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestDownloadWithVerify_Fresh(t *testing.T) {
	content := []byte("model weights data")
	expectedSHA := sha256hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "model.gguf")

	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	if err := c.downloadWithVerify(context.Background(), "org/model", "model.gguf", destPath, expectedSHA); err != nil {
		t.Fatalf("downloadWithVerify: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("file content = %q, want %q", got, content)
	}
}

func TestDownloadWithVerify_CacheHit(t *testing.T) {
	content := []byte("cached model weights")
	expectedSHA := sha256hex(content)

	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Write(content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "model.gguf")
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	if err := c.downloadWithVerify(context.Background(), "org/model", "model.gguf", destPath, expectedSHA); err != nil {
		t.Fatalf("downloadWithVerify: %v", err)
	}

	if requestCount != 0 {
		t.Errorf("expected 0 HTTP requests (cache hit), got %d", requestCount)
	}
}

func TestDownloadWithVerify_CacheCorrupt(t *testing.T) {
	content := []byte("correct model weights")
	expectedSHA := sha256hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "model.gguf")
	// Write corrupt content
	if err := os.WriteFile(destPath, []byte("corrupt data"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	if err := c.downloadWithVerify(context.Background(), "org/model", "model.gguf", destPath, expectedSHA); err != nil {
		t.Fatalf("downloadWithVerify: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("file content after re-download = %q, want %q", got, content)
	}
}

func TestDownloadWithVerify_ChecksumMismatch(t *testing.T) {
	content := []byte("model weights")
	wrongSHA := "0000000000000000000000000000000000000000000000000000000000000000"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "model.gguf")

	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	err := c.downloadWithVerify(context.Background(), "org/model", "model.gguf", destPath, wrongSHA)
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}
}

func TestDownloadWithVerify_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "model.gguf")

	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	err := c.downloadWithVerify(context.Background(), "org/model", "model.gguf", destPath, "anysha")
	if err == nil {
		t.Fatal("expected error for server 500, got nil")
	}
}

func TestDownloadWithVerify_CreatesDirectories(t *testing.T) {
	content := []byte("data")
	expectedSHA := sha256hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "nested", "deep", "model.gguf")

	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	if err := c.downloadWithVerify(context.Background(), "org/model", "model.gguf", destPath, expectedSHA); err != nil {
		t.Fatalf("downloadWithVerify: %v", err)
	}

	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("file not found at %q: %v", destPath, err)
	}
}
