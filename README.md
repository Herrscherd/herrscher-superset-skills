# herrscher-superset-skills

**Playbooks for the seam between Herrscher and Superset.** A plugin that builds
nothing and implements no port. Its whole contribution is two skills, teaching an
agent how a Herrscher session and a [Superset](https://superset.sh) workspace line
up: which worktree owns the branch, where the PR is pushed from, and how a job
crosses from one harness to the other without ending up half-done in both.

It installs them **only on a machine that actually runs Superset**. A playbook
about a tool that is not there is noise in every agent's context forever, so the
skills factory refuses rather than shipping advice nobody can follow.

Category: `contracts.CategorySkills`, skills only, no port. Registered by
`contracts.Register` in `init()` under kind `superset`. It implements no port and
consumes none. Status: experimental.

## Install

```bash
herrscher plugin add github.com/Herrscherd/herrscher-superset-skills
```

Blank-import the package into a herrscher host, the xcaddy pattern:

```go
import _ "github.com/Herrscherd/herrscher-superset-skills"
```

## Configuration

| Setting | Environment | Default | What it is |
|---|---|---|---|
| `home` | `SUPERSET_HOME` | `~/.superset` | the Superset state directory, whose existence proves Superset is installed here |

It is optional. `~` and `~/` are expanded against the user's home directory;
`~user/...` is refused with an error rather than read as a relative path.

## What it ships

| Skill | Answers |
|---|---|
| `superset-session` | A job arrived and belongs in a worktree, but which one? How to tell where you are standing, which harness owns the branch, where the PR comes from, and the four ways to nest the two worktrees wrongly |
| `superset-handoff` | Moving work between a Herrscher session and a Superset agent in either direction: the commands each side takes, what the brief must carry, and where the answer lands once it is done |

## When it installs nothing

`Skills` stats the configured home. When it is absent, or is a file, it returns
`ErrNotInstalled`; the host reports that and installs nothing. It is not fatal,
and every other plugin in the build is unaffected. Any other stat failure, such
as a permission error, is returned with its cause and is not `ErrNotInstalled`.

One naming note: Superset's own agents export `SUPERSET_HOME_DIR`, not
`SUPERSET_HOME`. The setting is named for this plugin's config surface, and it
defaults to `~/.superset`, so a normal install needs no configuration at all.

## Packages

| Package | Kind | What it is |
|---|---|---|
| `github.com/Herrscherd/herrscher-superset-skills` (root) | plugin module | The `superset` skills plugin: the embedded playbooks and their registration |
| `tools/commentcheck` | build tooling | Refuses stray comments and doc comments in test files; run by `scripts/check.sh` |

## Build and check

```bash
go build ./...
scripts/check.sh
```

`scripts/check.sh` runs the module tidy and verify checks, the formatter and
linter, the comment check, the regenerate-from-scratch diff, the dash check,
`go test -race -shuffle=on`, the build and `govulncheck`. CI runs the same script.

## Further reading

- [contracts](https://github.com/Herrscherd/herrscher-contracts), for
  `SkillsFactory` and `CategorySkills`
- [Superset CLI docs](https://docs.superset.sh/cli)
