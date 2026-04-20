package huggingface

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

type fileInfo struct {
	RFilename string `json:"rfilename"`
	SHA256    string `json:"sha256"`
}

type modelInfo struct {
	Siblings []fileInfo `json:"siblings"`
}

func (c *Client) fetchFileInfo(ctx context.Context, repo, filename string) (*fileInfo, error) {
	url := fmt.Sprintf("%s/api/models/%s/revision/main", c.baseURL, repo)
	slog.DebugContext(ctx, "Fetching file info", "url", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch file info: %s", resp.Status)
	}

	var info modelInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	for _, f := range info.Siblings {
		if f.RFilename == filename {
			return &f, nil
		}
	}
	return nil, fmt.Errorf("file %s not found in repo %s", filename, repo)
}

func checksumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
