package updater

import (
	"fmt"
	"strings"
)

// isNewerVersion compara semver simples (vMAJOR.MINOR.PATCH) — só lida com prefixo 'v'
func isNewerVersion(current, remote string) bool {
	if current == "" || remote == "" {
		return false
	}
	c := normalizeVersion(current)
	r := normalizeVersion(remote)

	cParts := strings.Split(c, ".")
	rParts := strings.Split(r, ".")

	for i := 0; i < 3; i++ {
		var ci, ri int
		if i < len(cParts) {
			_, err := fmt.Sscanf(cParts[i], "%d", &ci)
			if err != nil {
				ci = 0
			}
		}
		if i < len(rParts) {
			_, err := fmt.Sscanf(rParts[i], "%d", &ri)
			if err != nil {
				ri = 0
			}
		}
		if ri > ci {
			return true
		}
		if ri < ci {
			return false
		}
	}
	return false
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	return v
}
