package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/waldirborbajr/sync/config"
	"github.com/waldirborbajr/sync/logger"
)

// UpdateInfo traz a versão mais recente e a URL para download
type UpdateInfo struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// CheckForUpdateWithContext consulta o endpoint configurado e informa se há uma nova versão com contexto
func CheckForUpdateWithContext(ctx context.Context, currentVersion string, cfg config.Config) (bool, UpdateInfo, error) {
	log := logger.GetLogger()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	info, err := fetchUpdateInfo(ctx, cfg.UpdateCheckURL)
	if err != nil {
		return false, UpdateInfo{}, err
	}

	log.Debug().Str("remote_version", info.Version).Str("download_url", info.URL).Msg("Update info retrieved")

	if isNewerVersion(currentVersion, info.Version) {
		return true, info, nil
	}
	return false, info, nil
}

func fetchUpdateInfo(ctx context.Context, urlStr string) (UpdateInfo, error) {
	// If empty, default to the GitHub releases page for this repo
	if strings.TrimSpace(urlStr) == "" {
		urlStr = "https://github.com/waldirborbajr/sync/releases/latest"
	}

	// Detect GitHub releases URL and use the API when applicable
	if owner, repo, ok := parseGithubOwnerRepo(urlStr); ok {
		return fetchFromGitHubAPI(ctx, owner, repo)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("error while checking update: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{}, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var info UpdateInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return UpdateInfo{}, fmt.Errorf("error decoding update info: %w", err)
	}
	return info, nil
}

// parseGithubOwnerRepo tenta extrair owner e repo de URLs do GitHub relacionadas a releases
func parseGithubOwnerRepo(u string) (owner, repo string, ok bool) {
	// exemplos válidos:
	// https://github.com/owner/repo/releases
	// https://github.com/owner/repo/releases/new
	// https://github.com/owner/repo/releases/latest
	parts := strings.Split(strings.TrimPrefix(u, "https://"), "/")
	if len(parts) < 3 {
		return "", "", false
	}
	if parts[0] != "github.com" {
		return "", "", false
	}
	owner = parts[1]
	repo = parts[2]
	return owner, repo, true
}

// fetchFromGitHubAPI usa a API pública para obter o último release
func fetchFromGitHubAPI(ctx context.Context, owner, repo string) (UpdateInfo, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("error creating request to GitHub API: %w", err)
	}
	// Set a user agent to avoid being rejected
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sync-updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("error while calling GitHub API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{}, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Minimal struct to decode fields we need
	var gh struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			BrowserDownloadURL string `json:"browser_download_url"`
			Name               string `json:"name"`
		} `json:"assets"`
		ZipballURL string `json:"zipball_url"`
		TarballURL string `json:"tarball_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return UpdateInfo{}, fmt.Errorf("error decoding GitHub release info: %w", err)
	}

	info := UpdateInfo{Version: gh.TagName}

	if url := selectAssetURL(gh.Assets, runtime.GOOS, runtime.GOARCH); url != "" {
		info.URL = url
		return info, nil
	}
	// Fallback to zipball URL
	if gh.ZipballURL != "" {
		info.URL = gh.ZipballURL
		return info, nil
	}
	// No useful URL found
	return info, nil
}

func selectAssetURL(assets []struct {
	BrowserDownloadURL string `json:"browser_download_url"`
	Name               string `json:"name"`
}, goos, goarch string) string {
	if len(assets) == 0 {
		return ""
	}

	osTokens := osMatchTokens(goos)
	archTokens := archMatchTokens(goarch)

	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if name == "" || a.BrowserDownloadURL == "" {
			continue
		}
		if matchesAnyToken(name, osTokens) && matchesAnyToken(name, archTokens) {
			return a.BrowserDownloadURL
		}
	}

	for _, a := range assets {
		if a.BrowserDownloadURL != "" {
			return a.BrowserDownloadURL
		}
	}

	return ""
}

func osMatchTokens(goos string) []string {
	switch strings.ToLower(goos) {
	case "darwin":
		return []string{"darwin", "macos", "mac", "osx"}
	case "windows":
		return []string{"windows", "win"}
	case "linux":
		return []string{"linux"}
	default:
		return []string{strings.ToLower(goos)}
	}
}

func archMatchTokens(goarch string) []string {
	switch strings.ToLower(goarch) {
	case "amd64":
		return []string{"amd64", "x86_64"}
	case "arm64":
		return []string{"arm64", "aarch64"}
	case "386":
		return []string{"386", "x86"}
	default:
		return []string{strings.ToLower(goarch)}
	}
}

func matchesAnyToken(name string, tokens []string) bool {
	for _, t := range tokens {
		if strings.Contains(name, t) {
			return true
		}
	}
	return false
}
