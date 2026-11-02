// Package supersetskills contributes the playbooks that teach an agent where a
// Herrscher session ends and a Superset workspace begins. It builds nothing and
// implements no port: what it adds to a host is what the agent knows.
package supersetskills

import (
	"context"
	"embed"
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

// Skills is the plugin's whole contribution: the embedded playbook tree, but
// only on a machine that actually runs Superset.
//
// The check is the point of the factory. These playbooks are about the seam
// between two harnesses, and on a host with only one of them they are advice
// about a tool that is not there, noise sitting in every agent's context
// forever, for a workflow it can never run. So the absence of the state dir is
// an error and not an empty tree: the host installs nothing and says why.
func Skills(_ context.Context, cfg contracts.PluginConfig) (fs.FS, error) {
	home, err := resolveHome(cfg.Get("home"))
	if err != nil {
		return nil, err
	}
	switch fi, err := os.Stat(home); {
	case os.IsNotExist(err):
		return nil, fmt.Errorf("no Superset install at %s", home)
	case err != nil:
		return nil, fmt.Errorf("no Superset install at %s: %w", home, err)
	case !fi.IsDir():
		return nil, fmt.Errorf("no Superset install at %s: not a directory", home)
	}
	return fs.Sub(skillsFS, "skills")
}

func resolveHome(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		v = supersetStateDir
	}
	if v != "~" && !strings.HasPrefix(v, "~/") {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expand %q: %w", v, err)
	}
	if v == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(v, "~/")), nil
}
