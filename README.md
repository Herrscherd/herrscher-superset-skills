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

It is optional, and a leading `~/` is expanded against the user's home directory.

## What it ships

| Skill | Answers |
|---|---|
| `superset-session` | A job arrived and belongs in a worktree, but which one? How to tell where you are standing, which harness owns the branch, where the PR comes from, and the four ways to nest the two worktrees wrongly |
| `superset-handoff` | Moving work between a Herrscher session and a Superset agent in either direction: the commands each side takes, what the brief must carry, and where the answer lands once it is done |

Both are grounded in mechanisms verified against the two CLIs and against a live
Superset install. Neither restates the eight skills Superset installs and updates
itself.

## When it installs nothing

`Skills` stats the configured home and returns `no Superset install at <path>`
when it is absent. The host reports that and installs nothing. It is not fatal,
and every other plugin in the build is unaffected.

One naming note: Superset's own agents export `SUPERSET_HOME_DIR`, not
`SUPERSET_HOME`. The setting is named for this plugin's config surface, and it
defaults to `~/.superset`, so a normal install needs no configuration at all.

## Further reading

- [contracts](https://github.com/Herrscherd/herrscher-contracts), for
  `SkillsFactory` and `CategorySkills`
- [Superset CLI docs](https://docs.superset.sh/cli)
