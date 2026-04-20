package model

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

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

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
var idRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?(/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?)?$`)

func (i Info) Validate() error {
	if i.Name == "" || i.Id == "" || i.File == "" {
		return fmt.Errorf("empty content")
	}
	if !nameRegex.MatchString(i.Name) {
		return fmt.Errorf("invalid name")
	}
	if !idRegex.MatchString(i.Id) {
		return fmt.Errorf("invalid id")
	}
	if !isValidPath(i.File) {
		return fmt.Errorf("invalid file")
	}
	return nil
}

func isValidPath(p string) bool {
	if p == "" {
		return false
	}

	cleaned := path.Clean(p)

	if path.IsAbs(cleaned) {
		return false
	}

	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return false
	}

	return true
}
