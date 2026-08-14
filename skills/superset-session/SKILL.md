---
name: superset-session
description: Use when a job arriving in a Herrscher session belongs in a Superset workspace, when you find yourself working inside `~/.superset/worktrees/…`, or when you must decide which of the two worktrees owns the branch and where the pull request is pushed from.
---

# A Herrscher session and a Superset workspace are two worktrees

Both harnesses isolate work the same way: a git worktree of the project, on its
own branch, with an agent living in it. They are never the same worktree, and a
job belongs to exactly one of them. Almost everything that goes wrong at this
seam is one job spread across both.

## Which one am I in?

A Superset workspace tells its agent so, through the environment:

```
SUPERSET_WORKSPACE_ID     the id every `superset workspaces …` command takes
SUPERSET_WORKSPACE_PATH   this worktree
SUPERSET_ROOT_PATH        the project's main checkout it was cut from
SUPERSET_HOME_DIR         Superset's own state dir (~/.superset)
SUPERSET_TERMINAL_ID      the terminal this agent is running in
```

`SUPERSET_WORKSPACE_ID` unset means you are not in a Superset workspace — do not
guess from the path. Git answers the other half:

```bash
git rev-parse --git-common-dir   # the project's real .git, wherever you stand
git worktree list                # the main checkout and every worktree of it
```

A Herrscher session's worktree is a directory named after the session under
`.herrscher-sessions/` inside the project, on a branch starting with `session/`.

## The two layouts

| | Herrscher session | Superset workspace |
|---|---|---|
| Worktree | `<project>/.herrscher-sessions/<instance>/<name>` | `~/.superset/worktrees/<project-id>/<workspace-name>` |
| Branch | `session/<instance>/<name>`, minted for it | the `--branch` you asked for, namespaced under the project's branch prefix unless `--skip-branch-prefix` |
| Opened by | `herrscher session create --name <name> [--project <name>]` | `superset workspaces create --project <id> --branch <b>` |
| Provisioning | none | the repo's `.superset/config.json` `setup` scripts run on create |
| On close | `herrscher session close` removes the worktree and **keeps the branch** | deleting the workspace is a separate, explicit, destructive step |

Both are ordinary worktrees of the *same* `.git`: `git worktree list` from either
one lists the other, and both share the project's remotes. Nothing is copied,
nothing is vendored, and there is no second clone.

## Which one owns the branch

The worktree the commits are made in — that is the whole answer, and it is
decided when the job starts, not later. Two worktrees on one branch is not a
thing git will do for you: `worktree add` refuses a branch that is already
checked out elsewhere. Take that refusal as the design, not as an obstacle;
detaching HEAD or cloning again to get around it buys you two agents writing one
branch.

Names never collide by construction — Herrscher mints `session/<instance>/<name>`
for itself, Superset uses the branch you named — so "who owns it" is never
ambiguous once you know where the commits went.

## Where the PR comes from

From the worktree that holds the commits, with the project's own remotes:

```bash
git push -u origin "$(git branch --show-current)"
gh pr create
```

That is the `pr-job` flow unchanged; being inside `~/.superset/worktrees/…`
changes nothing about it. What it does change is *where you must be standing*: a
Superset workspace's branch is pushed from the Superset worktree, a session's
branch from the session worktree. Pushing one from the other means committing to
a branch that is checked out somewhere else, where another agent is working.

Report the PR URL in the conversation the job arrived in. A PR nobody is told
about is work the operator has to go find.

## What not to do

1. **Do not point a Herrscher session at a Superset worktree.** `session create
   --project` is a single path segment (matched against
   `^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`) resolved under the daemon's configured
   workspace root. A path is rejected outright, and a bare name quietly resolves
   somewhere else entirely. No flag makes a session adopt an existing worktree.

2. **Do not read `--shared` as "here".** It skips the worktree and runs in the
   daemon's own checkout — its workspace root, or the directory the daemon was
   launched from when no root is configured. That is not your Superset worktree.

3. **Do not nest one inside the other.** If the daemon's workspace root, or its
   cwd, is a Superset worktree, `session create` adds
   `.herrscher-sessions/<instance>/<name>` *inside* it. That inner worktree is
   registered in the project's real `.git`, which Superset knows nothing about,
   so removing the workspace directory leaves a registration pointing at nothing
   and an orphaned `session/…` branch — and takes any uncommitted work in the
   inner worktree with it. If it has already happened:

   ```bash
   git -C <project> worktree list    # the dead one is marked "prunable"
   git -C <project> worktree prune   # clears the registration; keeps the branch
   ```

4. **Do not open a second home for a job that already has one.** Check first —
   `superset workspaces list --search <branch>` and `herrscher session list` —
   and if both already exist for one job, say so and let the operator pick which
   survives. Do not try to reconcile two worktrees yourself.

5. **Do not assume a session worktree is provisioned.** Superset runs the repo's
   setup scripts when it creates a workspace; a Herrscher worktree gets no such
   hook, so gitignored env files and installed dependencies are simply absent
   until you put them there. Do not report a build failure that is really an
   unprovisioned tree.

## Where a job belongs

- **Superset workspace** — several agents in parallel, a repo whose setup scripts
  matter, reviewing a pull request (`superset workspaces create --pr <n>`), or
  work that must outlive the conversation that started it.
- **Herrscher session** — the conversation *is* the work: it binds to the
  daemon's gateways, keeps its transcript, and answers where it was asked.
  `move-the-conversation` is the authority on opening one.
- **Neither** — a question, a read, a one-file fix on a branch you already have
  checked out. Spawning a worktree to answer it is the expensive way to type.

Whichever it is, it is one worktree, one branch, one agent. When the job has to
cross from one harness to the other anyway, that is a handoff: see
`superset-handoff`.
