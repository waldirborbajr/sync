package updater

import "testing"

func TestParseGithubOwnerRepo(t *testing.T) {
	cases := []struct {
		url       string
		wantOwner string
		wantRepo  string
		wantOK    bool
	}{
		{"https://github.com/waldirborbajr/sync/releases/new", "waldirborbajr", "sync", true},
		{"https://github.com/waldirborbajr/sync/releases/latest", "waldirborbajr", "sync", true},
		{"https://github.com/waldirborbajr/sync/releases", "waldirborbajr", "sync", true},
		{"https://example.com/foo/bar", "", "", false},
		{"", "", "", false},
	}

	for _, c := range cases {
		owner, repo, ok := parseGithubOwnerRepo(c.url)
		if ok != c.wantOK || owner != c.wantOwner || repo != c.wantRepo {
			t.Fatalf("parseGithubOwnerRepo(%q) = (%q, %q, %v); want (%q, %q, %v)", c.url, owner, repo, ok, c.wantOwner, c.wantRepo, c.wantOK)
		}
	}
}
