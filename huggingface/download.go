package huggingface

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func (c *Client) downloadWithVerify(ctx context.Context, repo, filename, destPath, expectedSHA256 string) error {
	// キャッシュ済みの場合もチェックサムを検証
	if _, err := os.Stat(destPath); err == nil {
		slog.DebugContext(ctx, "Cache hit", "repo", repo, "filename", filename)
		sum, err := checksumFile(destPath)
		if err != nil {
			return err
		}
		if sum == expectedSHA256 {
			return nil // 正常
		}
		// 壊れているので再ダウンロード
		slog.InfoContext(ctx, "Checksum mismatch, redownloading", "repo", repo, "filename", filename)
		os.Remove(destPath)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", repo, filename)
	slog.DebugContext(ctx, "Downloading", "url", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	tmp := destPath + ".llamagatewaypartialdownload"
	slog.DebugContext(ctx, "Writing to", "tmp", tmp)
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(tmp) // エラー時の掃除（Rename後は空振り）
	}()

	h := sha256.New()
	// ファイルへの書き込みとハッシュ計算を同時に行う
	writer := io.MultiWriter(f, h)
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return err
	}
	slog.DebugContext(ctx, "Download complete")

	sum := hex.EncodeToString(h.Sum(nil))
	if expectedSHA256 != "" && sum != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, sum)
	}

	slog.DebugContext(ctx, "Moving to final location", "dest", destPath)
	return os.Rename(tmp, destPath)
}
