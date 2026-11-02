package supersetskills

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

func cfgHome(path string) contracts.PluginConfig {
	return contracts.PluginConfig{Settings: map[string]string{"home": path}}
}

func TestSkills_NoInstall(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "no-superset-here")
	tree, err := Skills(context.Background(), cfgHome(absent))
	if err == nil {
		t.Fatal("want an error for a missing Superset home, got a tree")
	}
	if tree != nil {
		t.Fatalf("want no tree alongside the error, got %v", tree)
	}
	if !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("error must be ErrNotInstalled, got %v", err)
	}
	if !strings.Contains(err.Error(), absent) {
		t.Fatalf("error must name the path it looked at, got %q", err)
	}
}

func TestSkills_HomeIsAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "superset")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Skills(context.Background(), cfgHome(file))
	if !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("a file at home must be ErrNotInstalled, got %v", err)
	}
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("a file at home must be ErrNotDirectory, got %v", err)
	}
}

func TestSkills_StatFailureKeepsItsCause(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	parent := t.TempDir()
	home := filepath.Join(parent, "superset")
	if err := os.Mkdir(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	_, err := Skills(context.Background(), cfgHome(home))
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("a permission failure must keep its cause, got %v", err)
	}
	if errors.Is(err, ErrNotInstalled) {
		t.Fatalf("a permission failure must not be reported as not installed, got %v", err)
	}
}

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

func TestResolveHome_DefaultAndAbsolute(t *testing.T) {
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

func TestRegistered(t *testing.T) {
	var found int
	for _, p := range contracts.Default.Plugins() {
		if p.Manifest.Kind != "superset" {
			continue
		}
		found++
		checkSkillsPlugin(t, p)
	}
	if found != 1 {
		t.Fatalf("want exactly one registered superset plugin, got %d", found)
	}
}

func checkSkillsPlugin(t *testing.T, p contracts.Plugin) {
	t.Helper()
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
	if s := p.Manifest.Config[0]; s.Env != "SUPERSET_HOME" || s.Default != supersetStateDir || s.Required {
		t.Fatalf("home setting is %+v, want SUPERSET_HOME, default %s, not required", s, supersetStateDir)
	}
}

func frontmatter(body string) (map[string]string, error) {
	rest, ok := strings.CutPrefix(body, "---\n")
	if !ok {
		return nil, fs.ErrInvalid
	}
	head, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return nil, fs.ErrInvalid
	}
	out := make(map[string]string)
	for _, line := range strings.Split(head, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fs.ErrInvalid
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, nil
}
