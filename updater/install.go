package updater

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/waldirborbajr/sync/logger"
)

// InstallUpdateWithContext replaces the current executable with the downloaded file.
func InstallUpdateWithContext(ctx context.Context, downloadPath string) error {
	log := logger.GetLogger()

	if runtime.GOOS == "windows" {
		return fmt.Errorf("auto-install is not supported on Windows")
	}
	if downloadPath == "" {
		return fmt.Errorf("empty download path")
	}
	if isArchiveFile(downloadPath) {
		return fmt.Errorf("downloaded file appears to be an archive, not a binary")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error resolving executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("error resolving symlinks: %w", err)
	}

	exeInfo, err := os.Stat(exePath)
	if err != nil {
		return fmt.Errorf("error stating current executable: %w", err)
	}

	tmpPath := exePath + ".tmp"
	if err := copyFileWithContext(ctx, downloadPath, tmpPath, exeInfo.Mode()); err != nil {
		return err
	}

	backupPath := exePath + ".bak"
	_ = os.Remove(backupPath)
	if err := os.Rename(exePath, backupPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("error creating backup: %w", err)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		_ = os.Rename(backupPath, exePath)
		_ = os.Remove(tmpPath)
		return fmt.Errorf("error replacing executable: %w", err)
	}

	log.Info().Str("path", exePath).Str("backup", backupPath).Msg("Update installed")
	return nil
}

// InstallUpdate maintains compatibility with the previous version without context.
func InstallUpdate(downloadPath string) error {
	return InstallUpdateWithContext(context.Background(), downloadPath)
}

func copyFileWithContext(ctx context.Context, src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}
	defer func() { _ = out.Close() }()

	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := in.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return fmt.Errorf("error writing temp file: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("error reading source file: %w", readErr)
		}
	}

	return nil
}

func isArchiveFile(path string) bool {
	name := strings.ToLower(path)
	return strings.HasSuffix(name, ".zip") ||
		strings.HasSuffix(name, ".tar.gz") ||
		strings.HasSuffix(name, ".tgz") ||
		strings.HasSuffix(name, ".tar")
}
