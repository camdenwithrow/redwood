# Redwood

Redwood is a small Git worktree and tmux project manager written in Go. It is
invoked as `rw` from the main repository checkout and coordinates worktrees,
detached tmux sessions, and optional stable development ports.

## Prerequisites

- Git with worktree support.
- tmux available on `PATH`.
- Go 1.26.5 or newer to install from source.
- Any tools referenced by the project's configured commands, such as `just` or
  Doppler.

Redwood does not load or store secrets. Put secret management in the configured
command, such as `doppler run -- just dev-server`, when the project needs it.

## Installation

Install the latest version with Go:

```sh
go install github.com/camdenwithrow/redwood/cmd/rw@latest
```

Ensure Go's binary directory is on `PATH`, then verify the installation:

```sh
rw help
```

When developing Redwood itself, install the current checkout with
`go install ./cmd/rw` from the repository root.

## Configuration

Redwood reads a committed `redwood.toml` from the repository root. Projects
that only need a worktree and an interactive tmux shell can use:

```toml
# Optional; auto-detects a local "main" or "master" when omitted.
base_branch = "main"
worktree_path = "../{repo}-{branch}"
```

With no commands configured, `rw start` creates a `shell` window rooted in the
worktree. Projects can also define command windows without assigning ports:

```toml
worktree_path = "../{repo}-{branch}"

[commands]
tests = "go test ./..."
```

Projects that run development servers can opt into stable ports:

```toml
# Optional; auto-detects a local "main" or "master" when omitted.
base_branch = "main"
worktree_path = "../{repo}-{branch}"
port_stride = 100

# Labels are user-defined and must match between the two tables.
[ports]
frontend = 3000
backend = 8080
simulator = 8081

[commands]
frontend = "just dev-web --port $RW_PORT"
backend = "just dev-server --port $RW_PORT --frontend-port $RW_PORT_FRONTEND"
simulator = "just dev-mobile --port $RW_PORT"
```

Projects can run setup commands immediately after a worktree is created:

```toml
worktree_path = "../{repo}-{branch}"

[hooks]
post_create = [
  "go mod download",
  "cp .env.example .env",
]
```

Post-create hooks run sequentially through `/bin/sh -c`, with the new worktree
as the working directory. Redwood inherits the invoking process's environment
and streams each command's input and output. If any hook fails, later hooks do
not run and Redwood rolls back the worktree, its slot allocation, and any branch
created by the command. An existing branch is retained during rollback.

The labels are not built into Redwood; users may define any commands their
project needs. Each label creates one tmux window. When a command has a
matching port label, Redwood sets that window's `RW_PORT` environment variable
to `base port + slot * port_stride` before starting it. Commands without a
matching port run without `RW_PORT`. Every command also receives every
configured service port as `RW_PORT_<LABEL>`, so services in the same worktree
can discover one another. Labels are uppercased and non-alphanumeric runs become
underscores; for example, `user-api` becomes `RW_PORT_USER_API`. Labels that
would produce the same environment variable are rejected.

Each worktree receives a stable numeric slot, allowing the same command set to
run in several worktrees without port conflicts. Base ports must have different
remainders when divided by `port_stride`, which prevents one command's port in
one slot from colliding with another command in a different slot. Redwood only
supplies these port variables; Doppler remains responsible for secrets.

The configuration fields are:

- `base_branch`: the branch used when creating a new branch. It may be omitted
  when exactly one local `main` or `master` branch exists.
- `worktree_path`: the path template, resolved from the main checkout when
  relative. `{repo}` and `{branch}` expand to filesystem-safe values.
- `port_stride`: the amount added to every base port for each worktree slot;
  required only when `[ports]` is configured.
- `[ports]`: optional command labels and their base ports. Every port label must
  have a matching command.
- `[commands]`: optional labels and the commands Redwood runs in tmux windows.
- `[hooks].post_create`: optional shell commands run in order after creating a
  worktree and assigning its slot.

Slot allocations are kept in `.git/redwood/allocations.toml`, inside the shared
Git directory rather than the committed repository.

## Commands

```text
rw create feature/foo   Create a worktree and assign its slot
rw remove feature/foo   Remove a worktree while keeping its branch
rw start feature/foo    Start its commands in a detached tmux session
rw attach feature/foo   Attach to its tmux session
rw stop feature/foo     Stop its tmux session
rw list                 Show worktrees, ports, and running state
```

## Core workflow

Run Redwood commands from the main checkout or any linked worktree. Redwood
uses the shared Git directory to find the main checkout, configuration, and all
other worktrees:

```sh
cd /path/to/main-checkout
rw create feature/foo
rw start feature/foo
rw list
rw attach feature/foo
```

When invoked from inside tmux, `rw attach` switches the current client to the
worktree's session instead of attempting a nested attachment. Use
`tmux switch-client -l` to return to the previous session.

Detach from the tmux session with tmux's configured detach binding, `Ctrl-b d`
by default. The commands continue running after detaching. Stop the whole
worktree session explicitly when it is no longer needed:

```sh
rw stop feature/foo
```

Remove a worktree after stopping its session:

```sh
rw remove feature/foo
```

The branch is retained. Git refuses the removal when the worktree contains
uncommitted changes; Redwood reports that Git error without forcing removal.

`rw list` prints one tab-separated row per worktree with its branch, slot,
Redwood-managed session state, calculated ports, and path. `RW_SESSION` is
`running` when that worktree has a Redwood-managed tmux session and `none`
otherwise; it does not describe unrelated tmux sessions. Redwood does not
delete the worktree or branch when stopping a session.
