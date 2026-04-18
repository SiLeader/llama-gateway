package huggingface

import (
	"context"
	"fmt"
	"log/slog"
)

type Client struct {
	token string
}

func NewClient(token string) *Client {
	return &Client{token: token}
}

func (c *Client) Download(ctx context.Context, repo, filename, destPath string) error {
	info, err := c.fetchFileInfo(ctx, repo, filename)
	if err != nil {
		return fmt.Errorf("failed to fetch file info: %w", err)
	}
	slog.InfoContext(ctx, "File metadata downloaded from Hugging Face", "repo", repo, "filename", filename)

	return c.downloadWithVerify(ctx, repo, filename, destPath, info.SHA256)
}
