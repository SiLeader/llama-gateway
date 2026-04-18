package model

import (
	"context"
	"path"

	"github.com/sileader/llama-gateway/huggingface"
)

type Info struct {
	Name    string `yaml:"name" json:"name"`
	Id      string `yaml:"id" json:"id"`
	File    string `yaml:"file" json:"file"`
	Context *int   `yaml:"context,omitempty" json:"context,omitempty"`
}

type Mapper struct {
	destination string
	mapping     map[string]Info
}

func NewModelMapper(models []Info, destination string) *Mapper {
	mapping := map[string]Info{}
	for _, model := range models {
		mapping[model.Name] = model
	}
	return &Mapper{
		destination: destination,
		mapping:     mapping,
	}
}

func (m *Mapper) UseModel(model string) *string {
	modelInfo, ok := m.mapping[model]
	if !ok {
		return nil
	}
	dest := modelInfo.DestinationPath(m.destination)
	return &dest
}

func (i Info) DestinationPath(destination string) string {
	return path.Join(destination, i.File)
}

func (i Info) Download(ctx context.Context, destination string, client *huggingface.Client) error {
	destPath := i.DestinationPath(destination)
	err := client.Download(ctx, i.Id, i.File, destPath)
	if err != nil {
		return err
	}
	return nil
}
