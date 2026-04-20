package model

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/sileader/llama-gateway/huggingface"
	"github.com/sileader/llama-gateway/llamaserver"
	"golang.org/x/sync/errgroup"
)

type Downloader struct {
	models      models
	destination string
	presetFile  string
	client      *huggingface.Client
	ls          *llamaserver.Manager
}

type models struct {
	m       sync.Mutex
	mapping map[string]Info
	presets llamaserver.Presets
}

func NewDownloader(
	mapping []Info,
	destination string,
	presetFile string,
	client *huggingface.Client,
	ls *llamaserver.Manager,
) (*Downloader, error) {
	mappingMap := map[string]Info{}
	presets := map[string]llamaserver.Preset{}
	for _, m := range mapping {
		if err := m.Validate(); err != nil {
			return nil, err
		}
		mappingMap[m.Name] = m
		presets[m.Name] = llamaserver.Preset{
			Model:   m.DestinationPath(destination),
			Context: m.Context,
		}
	}
	dl := &Downloader{
		models: models{
			mapping: mappingMap,
			presets: llamaserver.Presets{
				Global: nil,
				Models: presets,
			},
		},
		destination: destination,
		presetFile:  presetFile,
		client:      client,
		ls:          ls,
	}
	return dl, nil
}

func (d *Downloader) DownloadAll() error {
	ctx := context.Background()

	d.models.m.Lock()
	defer d.models.m.Unlock()

	eg := errgroup.Group{}
	for _, m := range d.models.mapping {
		eg.Go(func() error {
			slog.InfoContext(ctx, "Downloading model", "model", m.Id)
			return m.Download(ctx, d.destination, d.client)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(d.presetFile), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(d.presetFile, []byte(d.models.presets.String()), 0644); err != nil {
		return err
	}
	return nil
}

func (d *Downloader) AddModel(ctx context.Context, info Info) error {
	d.models.m.Lock()
	defer d.models.m.Unlock()

	if _, exists := d.models.mapping[info.Name]; exists {
		return nil
	}
	slog.InfoContext(ctx, "Downloading model", "model", info.Id)
	err := info.Download(ctx, d.destination, d.client)
	if err != nil {
		return err
	}
	destPath := info.DestinationPath(d.destination)

	d.models.mapping[info.Name] = info
	d.models.presets.Models[info.Name] = llamaserver.Preset{
		Model:   destPath,
		Context: info.Context,
	}
	if err := os.MkdirAll(filepath.Dir(d.presetFile), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(d.presetFile, []byte(d.models.presets.String()), 0644); err != nil {
		return err
	}
	if d.ls != nil {
		d.ls.RestartServer()
	}
	return nil
}
