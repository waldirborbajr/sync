package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"path"
	"strings"
	"time"

	"github.com/waldirborbajr/sync/logger"
)

// DownloadUpdateWithContext baixa o binário da URL de download para o diretório informado com contexto
func DownloadUpdateWithContext(ctx context.Context, downloadURL, destDir string) (string, error) {
	log := logger.GetLogger()

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// Use request with context to allow cancellations
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error fetching download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code when downloading: %d", resp.StatusCode)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("error creating download directory: %w", err)
	}

	fname := determineFilename(resp.Request.URL.Path)
	destPath := filepath.Join(destDir, fname)

	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("error creating destination file: %w", err)
	}
	defer func() { _ = f.Close() }()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error writing to file: %w", err)
	}
	log.Info().Str("file", destPath).Int64("bytes", n).Msg("Downloaded update file")
	return destPath, nil
}

func determineFilename(p string) string {
	// Extract the last path component from the URL path.
	base := path.Base(p)

	// Generate a safe fallback name if the base is empty or clearly invalid.
	generated := fmt.Sprintf("sync-%d.bin", time.Now().Unix())
	if base == "" || base == "." || base == "/" {
		return generated
	}

	// Ensure the filename is a single component without directory traversal.
	if strings.Contains(base, "/") || strings.Contains(base, "\\") || strings.Contains(base, "..") {
		return generated
	}

	return base
}
