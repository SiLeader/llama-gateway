package huggingface

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:   token,
		baseURL: "https://huggingface.co",
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	}
}

func (c *Client) Download(ctx context.Context, repo, filename, destPath string) error {
	info, err := c.fetchFileInfo(ctx, repo, filename)
	if err != nil {
		return fmt.Errorf("failed to fetch file info: %w", err)
	}
	slog.InfoContext(ctx, "File metadata downloaded from Hugging Face", "repo", repo, "filename", filename)

	return c.downloadWithVerify(ctx, repo, filename, destPath, info.SHA256)
}
