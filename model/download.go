package model

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/sileader/llama-gateway/huggingface"
	"github.com/sileader/llama-gateway/llamaserver"
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
) *Downloader {
	mappingMap := map[string]Info{}
	presets := map[string]llamaserver.Preset{}
	for _, m := range mapping {
		mappingMap[m.Name] = m
		presets[m.Name] = llamaserver.Preset{
			Model:   m.DestinationPath(destination),
			Context: m.Context,
		}
	}
	return &Downloader{
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
}

func (d *Downloader) DownloadAll() error {
	ctx := context.Background()

	d.models.m.Lock()
	defer d.models.m.Unlock()

	presets := map[string]llamaserver.Preset{}

	errChan := make(chan error, len(d.models.mapping))
	wg := sync.WaitGroup{}
	for _, m := range d.models.mapping {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slog.InfoContext(ctx, "Downloading model", "model", m.Id)
			var err error
			if err = m.Download(ctx, d.destination, d.client); err != nil {
				errChan <- err
			}
		}()
	}
	wg.Wait()

	select {
	case err := <-errChan:
		return err
	default:
	}

	ps := llamaserver.Presets{
		Global: nil,
		Models: presets,
	}

	if err := os.MkdirAll(filepath.Dir(d.presetFile), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(d.presetFile, []byte(ps.String()), 0644); err != nil {
		return err
	}
	d.models.presets = ps
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
		d.ls.ReloadServer()
	}
	return nil
}
