// Package supersetskills is a plugin module. It contributes the playbooks that
// teach an agent where a Herrscher session ends and a Superset workspace
// begins. It builds nothing and implements no port: what it adds to a host is
// what the agent knows.
package supersetskills

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

//go:embed skills
var skillsFS embed.FS

const supersetStateDir = "~/.superset"

// ErrNotInstalled reports that no Superset state dir exists where the plugin looks for one.
var ErrNotInstalled = errors.New("no Superset install")

// ErrNotDirectory reports that the Superset home exists but is a file, not a directory.
var ErrNotDirectory = errors.New("superset home is not a directory")

// ErrUnsupportedHome reports a superset home that names another user's home (`~user/...`), which is not expanded.
var ErrUnsupportedHome = errors.New("only ~ and ~/ are expanded in the superset home")

func init() {
	contracts.Register(contracts.Plugin{
		Manifest: contracts.Manifest{
			Kind:     "superset",
			Category: contracts.CategorySkills,
			Status:   contracts.StatusExperimental,
			Config: []contracts.Setting{
				{Key: "home", Env: "SUPERSET_HOME", Help: "Superset state dir, whose existence is what proves Superset is installed here (default ~/.superset)", Default: supersetStateDir},
			},
		},
		Skills: Skills,
	})
}

// Skills returns the embedded playbook tree when the Superset home is a
// directory. A missing home returns ErrNotInstalled; a file at that path
// returns ErrNotInstalled wrapping ErrNotDirectory. A home naming another user
// (`~user/...`) returns ErrUnsupportedHome. Any other stat failure is returned
// with its cause and without ErrNotInstalled.
func Skills(_ context.Context, cfg contracts.PluginConfig) (fs.FS, error) {
	home, err := resolveHome(cfg.Get("home"))
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(home)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("%w at %s", ErrNotInstalled, home)
	case err != nil:
		return nil, fmt.Errorf("stat superset home %s: %w", home, err)
	case !info.IsDir():
		return nil, fmt.Errorf("%w at %s: %w", ErrNotInstalled, home, ErrNotDirectory)
	}
	return fs.Sub(skillsFS, "skills")
}

func resolveHome(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		value = supersetStateDir
	}
	if value != "~" && !strings.HasPrefix(value, "~/") {
		if strings.HasPrefix(value, "~") {
			return "", fmt.Errorf("%w: %q", ErrUnsupportedHome, value)
		}
		return value, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expand %q: %w", value, err)
	}
	if value == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(value, "~/")), nil
}
