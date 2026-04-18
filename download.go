package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sileader/llama-gateway/huggingface"
	"github.com/sileader/llama-gateway/llamaserver"
	"github.com/sileader/llama-gateway/model"
)

type ModelDownloader struct {
	mapping     []model.Info
	destination string
	client      *huggingface.Client
}

func NewModelDownloader(mapping []model.Info, destination string, client *huggingface.Client) *ModelDownloader {
	return &ModelDownloader{
		mapping:     mapping,
		destination: destination,
		client:      client,
	}
}

func (d *ModelDownloader) DownloadAll(presetFile string) {
	ctx := context.Background()

	presets := map[string]llamaserver.Preset{}

	for _, m := range d.mapping {
		slog.InfoContext(ctx, "Downloading model", "model", m.Id)
		var destPath string
		var err error
		if destPath, err = m.Download(ctx, d.destination, d.client); err != nil {
			panic(err)
		}
		p := llamaserver.Preset{
			Model:   destPath,
			Context: m.Context,
		}
		presets[m.Name] = p
	}

	ps := llamaserver.Presets{
		Global: nil,
		Models: presets,
	}

	if err := os.MkdirAll(filepath.Dir(presetFile), 0755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(presetFile, []byte(ps.String()), 0644); err != nil {
		panic(err)
	}
}
