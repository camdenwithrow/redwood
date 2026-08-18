---
name: redwood
description: Manage parallel Git worktrees and Redwood-owned tmux sessions with the `rw` CLI. Use when an agent needs to create isolated branches or worktrees, run concurrent project sessions, inspect Redwood worktree/session/port state, attach to or stop a worktree session, remove a worktree safely, or coordinate multiple features in a repository containing `redwood.toml`.
---

# Redwood

Use Redwood to coordinate Git worktrees, tmux sessions, and optional stable ports. Prefer `rw` over direct `git worktree` or tmux commands for operations Redwood supports.

## Establish context

1. Confirm the current directory belongs to the intended Git repository.
2. Read `redwood.toml` from the main checkout before changing configuration.
3. Run `rw help` when the installed CLI behavior may differ from this skill.
4. Run `rw list` before acting on existing worktrees.

Redwood commands may run from the main checkout or any linked worktree. Redwood always resolves repository configuration through the shared main checkout.

If `rw` is unavailable, report that clearly. Install it only when installation is within the user's requested scope:

```sh
go install github.com/camdenwithrow/redwood/cmd/rw@latest
```

## Create parallel work

Create one branch-backed worktree per independent task:

```sh
rw create feature/first-change
rw create feature/second-change
rw list
```

Use the paths printed by `rw create`; do not infer them when a command can provide the exact path. New branches start from the configured `base_branch`, not necessarily the caller's current `HEAD`.

Make changes, test, commit, and push inside each corresponding worktree. Keep unrelated tasks on separate branches and do not mix their diffs.

## Manage sessions

Start and enter a worktree session with:

```sh
rw start feature/first-change
rw attach feature/first-change
```

With no configured commands, `rw start` creates an interactive `shell` window rooted in the worktree. Configured commands create named windows. A matching port entry adds `RW_PORT`; commands without ports run normally.

When already inside tmux, `rw attach` switches the current client instead of nesting sessions. Return to the prior tmux session with:

```sh
tmux switch-client -l
```

Interpret `RW_SESSION=running` in `rw list` as a Redwood-managed tmux session. `RW_SESSION=none` does not mean the worktree, branch, terminal, or unrelated tmux session is stopped.

## Stop and remove

Inspect state before cleanup:

```sh
rw list
```

Stop only a running Redwood-managed session:

```sh
rw stop feature/first-change
```

`rw stop` reports an error when no managed session exists; treat that as already stopped only after confirming with `rw list`.

Before removal, inspect the target for uncommitted work and run the command from a different checkout:

```sh
git -C /exact/worktree/path status --short
rw remove feature/first-change
```

`rw remove` removes the worktree, releases its slot, and retains the branch. It passes through Git errors and does not force removal of dirty worktrees. Never discard changes or delete the retained branch unless the user explicitly requests it.

## Respect configuration

A portless project needs only:

```toml
worktree_path = "../worktrees/{repo}/{branch}"
```

Do not add ports, commands, or infrastructure unless the project needs them. When ports are configured, preserve matching command labels and let Redwood calculate each worktree's stable port from its slot.
