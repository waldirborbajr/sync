package updater

import (
	"context"

	"github.com/waldirborbajr/sync/config"
)

// CheckForUpdate mantém compatibilidade com a versão anterior sem contexto
func CheckForUpdate(currentVersion string, cfg config.Config) (bool, UpdateInfo, error) {
	return CheckForUpdateWithContext(context.Background(), currentVersion, cfg)
}

// DownloadUpdate mantém compatibilidade com a versão anterior sem contexto
func DownloadUpdate(downloadURL, destDir string) (string, error) {
	return DownloadUpdateWithContext(context.Background(), downloadURL, destDir)
}
