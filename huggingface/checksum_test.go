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

func TestChecksumFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")
	content := []byte("hello world\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	want := sha256hex(content)
	got, err := checksumFile(path)
	if err != nil {
		t.Fatalf("checksumFile: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestChecksumFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.bin")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	want := sha256hex([]byte{})
	got, err := checksumFile(path)
	if err != nil {
		t.Fatalf("checksumFile: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestChecksumFile_NotFound(t *testing.T) {
	_, err := checksumFile("/nonexistent/path/file.bin")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestFetchFileInfo_Success(t *testing.T) {
	const repo = "org/mymodel"
	const filename = "model.gguf"
	const sha256 = "abc123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/api/models/" + repo + "/revision/main"
		if r.URL.Path != want {
			http.NotFound(w, r)
			return
		}
		resp := modelInfo{
			Siblings: []fileInfo{
				{RFilename: "other.gguf", SHA256: "999"},
				{RFilename: filename, SHA256: sha256},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	info, err := c.fetchFileInfo(context.Background(), repo, filename)
	if err != nil {
		t.Fatalf("fetchFileInfo: %v", err)
	}
	if info.SHA256 != sha256 {
		t.Errorf("SHA256 = %q, want %q", info.SHA256, sha256)
	}
	if info.RFilename != filename {
		t.Errorf("RFilename = %q, want %q", info.RFilename, filename)
	}
}

func TestFetchFileInfo_FileNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := modelInfo{Siblings: []fileInfo{{RFilename: "other.gguf", SHA256: "aaa"}}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	_, err := c.fetchFileInfo(context.Background(), "org/model", "missing.gguf")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestFetchFileInfo_WithToken(t *testing.T) {
	const token = "hf_testtoken"
	gotAuth := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		resp := modelInfo{Siblings: []fileInfo{{RFilename: "f.gguf", SHA256: "bbb"}}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &Client{token: token, baseURL: srv.URL, httpClient: srv.Client()}
	c.fetchFileInfo(context.Background(), "org/model", "f.gguf")

	if gotAuth != "Bearer "+token {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer "+token)
	}
}

func TestFetchFileInfo_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Client{token: "", baseURL: srv.URL, httpClient: srv.Client()}
	_, err := c.fetchFileInfo(context.Background(), "org/model", "f.gguf")
	if err == nil {
		t.Error("expected error for server error response, got nil")
	}
}
