package updater

import (
	"context"

	"github.com/waldirborbajr/sync/config"
	"github.com/waldirborbajr/sync/logger"
)

// RunUpdateFlow faz a checagem de versão e faz o download se configurado
func RunUpdateFlow(ctx context.Context, currentVersion string, cfg config.Config) (downloaded bool, filePath string, info UpdateInfo, err error) {
	log := logger.GetLogger()
	isNew, info, err := CheckForUpdateWithContext(ctx, currentVersion, cfg)
	if err != nil {
		return false, "", info, err
	}
	if !isNew {
		log.Debug().Str("current", currentVersion).Str("remote", info.Version).Msg("No newer version found")
		return false, "", info, nil
	}

	if cfg.AutoUpdate && info.URL != "" {
		log.Info().Msg("Auto-update enabled, downloading update...")
		path, err := DownloadUpdateWithContext(ctx, info.URL, cfg.UpdateDownloadDir)
		if err != nil {
			return false, "", info, err
		}
		log.Info().Msg("Installing update...")
		if err := InstallUpdateWithContext(ctx, path); err != nil {
			return true, path, info, err
		}
		return true, path, info, nil
	}

	return false, "", info, nil
}
