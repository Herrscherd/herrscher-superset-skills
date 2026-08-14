package supersetskills

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts"
)

// cfgHome builds the one-setting config the factory reads.
func cfgHome(path string) contracts.PluginConfig {
	return contracts.PluginConfig{Settings: map[string]string{"home": path}}
}

// A machine without Superset gets no playbooks at all — the whole reason Skills
// is a factory rather than a fixed tree.
func TestSkills_NoInstall(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "no-superset-here")
	tree, err := Skills(context.Background(), cfgHome(absent))
	if err == nil {
		t.Fatal("want an error for a missing Superset home, got a tree")
	}
	if tree != nil {
		t.Fatalf("want no tree alongside the error, got %v", tree)
	}
	if !strings.Contains(err.Error(), absent) {
		t.Fatalf("error must name the path it looked at, got %q", err)
	}
}

// A file where the state dir should be is not an install either.
func TestSkills_HomeIsAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "superset")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Skills(context.Background(), cfgHome(file)); err == nil {
		t.Fatal("want an error when home is a file, got a tree")
	}
}

// With Superset present the tree is served rooted at the skills dir, so its
// top-level entries are the skill names the host installs.
func TestSkills_Installed(t *testing.T) {
	tree, err := Skills(context.Background(), cfgHome(t.TempDir()))
	if err != nil {
		t.Fatalf("Skills: %v", err)
	}
	entries, err := fs.ReadDir(tree, ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("embedded tree is empty")
	}
	for _, e := range entries {
		if !e.IsDir() {
			t.Fatalf("top level must hold skill directories, got file %q", e.Name())
		}
		if e.Name() == "skills" {
			t.Fatal("tree is not rooted at the skills dir: entry \"skills\" leaked")
		}
		if _, err := fs.Stat(tree, filepath.Join(e.Name(), "SKILL.md")); err != nil {
			t.Fatalf("skill %q has no SKILL.md: %v", e.Name(), err)
		}
	}
}

// An empty home setting falls back to ~/.superset rather than statting "".
func TestResolveHome_Defaults(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no user home dir on this machine")
	}
	for _, in := range []string{"", "  ", "~/.superset"} {
		got, err := resolveHome(in)
		if err != nil {
			t.Fatalf("resolveHome(%q): %v", in, err)
		}
		if want := filepath.Join(home, ".superset"); got != want {
			t.Fatalf("resolveHome(%q) = %q, want %q", in, got, want)
		}
	}
	if got, err := resolveHome("/opt/superset"); err != nil || got != "/opt/superset" {
		t.Fatalf("an absolute path must pass through unchanged, got (%q,%v)", got, err)
	}
}

// Every playbook must be loadable by an agent: frontmatter with a name that
// matches its directory (that is what the skill is invoked as) and a
// description, which is the only thing read before the skill is opened.
func TestEmbeddedSkills_Frontmatter(t *testing.T) {
	tree, err := fs.Sub(skillsFS, "skills")
	if err != nil {
		t.Fatal(err)
	}
	dirs, err := fs.ReadDir(tree, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		body, err := fs.ReadFile(tree, filepath.Join(d.Name(), "SKILL.md"))
		if err != nil {
			t.Fatalf("%s: %v", d.Name(), err)
		}
		front, err := frontmatter(string(body))
		if err != nil {
			t.Fatalf("%s: %v", d.Name(), err)
		}
		if front["name"] != d.Name() {
			t.Fatalf("%s: frontmatter name is %q, must match the directory", d.Name(), front["name"])
		}
		if strings.TrimSpace(front["description"]) == "" {
			t.Fatalf("%s: empty description", d.Name())
		}
		if !strings.HasPrefix(front["description"], "Use when") {
			t.Fatalf("%s: description must start with \"Use when\", got %q", d.Name(), front["description"])
		}
	}
}

// The registration is what a host sees: a skills-only plugin, no port.
func TestRegistered(t *testing.T) {
	var found int
	for _, p := range contracts.Default.Plugins() {
		if p.Manifest.Kind != "superset" {
			continue
		}
		found++
		if p.Manifest.Category != contracts.CategorySkills {
			t.Fatalf("category is %q, want %q", p.Manifest.Category, contracts.CategorySkills)
		}
		if p.Manifest.Status != contracts.StatusExperimental {
			t.Fatalf("status is %q, want experimental", p.Manifest.Status)
		}
		if p.Gateway != nil || p.Backend != nil || p.Memory != nil || p.Orchestrator != nil {
			t.Fatal("a skills plugin must declare no port factory")
		}
		if p.Skills == nil {
			t.Fatal("no skills factory registered")
		}
		if len(p.Manifest.Config) != 1 || p.Manifest.Config[0].Key != "home" {
			t.Fatalf("config is %+v, want the single home setting", p.Manifest.Config)
		}
		if s := p.Manifest.Config[0]; s.Env != "SUPERSET_HOME" || s.Default != defaultHome || s.Required {
			t.Fatalf("home setting is %+v, want SUPERSET_HOME, default %s, not required", s, defaultHome)
		}
	}
	if found != 1 {
		t.Fatalf("want exactly one registered superset plugin, got %d", found)
	}
}

// frontmatter parses the leading --- fenced YAML block as flat key: value pairs,
// which is all a SKILL.md header ever is.
func frontmatter(body string) (map[string]string, error) {
	rest, ok := strings.CutPrefix(body, "---\n")
	if !ok {
		return nil, fs.ErrInvalid
	}
	head, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return nil, fs.ErrInvalid
	}
	out := map[string]string{}
	for _, line := range strings.Split(head, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fs.ErrInvalid
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, nil
}
