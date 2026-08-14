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

	"github.com/Herrscherd/herrscher-contracts"
)

// skillsFS carries the playbooks. One directory per skill, each holding a
// SKILL.md, which is the layout the host installs into ~/.claude/skills.
//
//go:embed skills
var skillsFS embed.FS

// defaultHome is where the Superset desktop app keeps its state — its CLI shim,
// its host service socket, and the worktrees it hands to agents.
const defaultHome = "~/.superset"

// init self-registers the plugin. A blank import of this package (in the host's
// generated plugins.go) is enough to make the playbooks discoverable — there is
// nothing else to wire, since the plugin has no port.
func init() {
	contracts.Register(contracts.Plugin{
		Manifest: contracts.Manifest{
			Kind:     "superset",
			Category: contracts.CategorySkills,
			Status:   contracts.StatusExperimental,
			Config: []contracts.Setting{
				{Key: "home", Env: "SUPERSET_HOME", Help: "Superset state dir, whose existence is what proves Superset is installed here (default ~/.superset)", Default: defaultHome},
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
// about a tool that is not there — noise sitting in every agent's context
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
	// Rooted at the skills dir, so the entries are the skill names the host
	// installs ("superset-session/SKILL.md") and not a "skills/" prefix leaking
	// this repo's layout into ~/.claude/skills.
	return fs.Sub(skillsFS, "skills")
}

// resolveHome expands a leading ~ against the user's home dir. The setting is
// read from an env var and from config.json, both of which people write with a
// tilde, and neither of which the shell expands for us.
func resolveHome(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		v = defaultHome
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
