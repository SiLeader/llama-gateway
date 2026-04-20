package model

import (
	"context"
	"testing"
)

func TestNewDownloader_Valid(t *testing.T) {
	models := []Info{
		{Name: "model-a", Id: "org/model-a", File: "a.gguf"},
	}
	dl, err := NewDownloader(models, "/tmp", "/tmp/presets.ini", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dl == nil {
		t.Fatal("expected non-nil Downloader")
	}
}

func TestNewDownloader_Empty(t *testing.T) {
	dl, err := NewDownloader([]Info{}, "/tmp", "/tmp/presets.ini", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dl == nil {
		t.Fatal("expected non-nil Downloader")
	}
}

func TestNewDownloader_InvalidModel(t *testing.T) {
	models := []Info{
		{Name: "bad name!", Id: "org/model", File: "a.gguf"},
	}
	_, err := NewDownloader(models, "/tmp", "/tmp/presets.ini", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid model name")
	}
}

func TestNewDownloader_EmptyModelName(t *testing.T) {
	models := []Info{
		{Name: "", Id: "org/model", File: "a.gguf"},
	}
	_, err := NewDownloader(models, "/tmp", "/tmp/presets.ini", nil, nil)
	if err == nil {
		t.Fatal("expected error for empty model name")
	}
}

func TestDownloadAll_NoModels(t *testing.T) {
	dl, err := NewDownloader([]Info{}, t.TempDir(), t.TempDir()+"/presets.ini", nil, nil)
	if err != nil {
		t.Fatalf("NewDownloader: %v", err)
	}
	if err := dl.DownloadAll(context.Background()); err != nil {
		t.Errorf("DownloadAll with no models: %v", err)
	}
}

func TestDownloadAll_CancelledContext(t *testing.T) {
	dl, err := NewDownloader([]Info{}, t.TempDir(), t.TempDir()+"/presets.ini", nil, nil)
	if err != nil {
		t.Fatalf("NewDownloader: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// No models to download, so a cancelled context still succeeds
	if err := dl.DownloadAll(ctx); err != nil {
		t.Errorf("unexpected error with cancelled context and no models: %v", err)
	}
}

func TestAddModel_Duplicate(t *testing.T) {
	info := Info{Name: "model-a", Id: "org/model-a", File: "a.gguf"}
	dl, err := NewDownloader([]Info{info}, t.TempDir(), t.TempDir()+"/presets.ini", nil, nil)
	if err != nil {
		t.Fatalf("NewDownloader: %v", err)
	}
	// Adding the same model again should be a no-op (nil error, no download)
	if err := dl.AddModel(context.Background(), info); err != nil {
		t.Errorf("AddModel duplicate: expected nil error, got %v", err)
	}
}
