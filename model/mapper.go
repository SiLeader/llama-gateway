package model

import (
	"context"
	"path/filepath"

	"github.com/sileader/llama-gateway/huggingface"
)

type Info struct {
	Name    string `yaml:"name" json:"name"`
	Id      string `yaml:"id" json:"id"`
	File    string `yaml:"file" json:"file"`
	Context *int   `yaml:"context,omitempty" json:"context,omitempty"`
}

func (i Info) DestinationPath(destination string) string {
	cleaned := filepath.Join("/", i.File)
	return filepath.Join(destination, cleaned)
}

func (i Info) Download(ctx context.Context, destination string, client *huggingface.Client) error {
	destPath := i.DestinationPath(destination)
	err := client.Download(ctx, i.Id, i.File, destPath)
	if err != nil {
		return err
	}
	return nil
}
