# Redwood

Redwood is a small Git worktree and tmux project manager written in Go. It is
invoked as `rw` from the main repository checkout and coordinates worktrees,
stable development ports, and detached tmux sessions.

## Configuration

Redwood reads a committed `redwood.toml` from the repository root:

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
backend = "just dev-server --port $RW_PORT"
simulator = "just dev-mobile --port $RW_PORT"
```

The labels are not built into Redwood; users may define any commands their
project needs. Each label creates one tmux window and must have one matching
base port. Redwood sets that command's `RW_PORT` environment variable to
`base port + slot * port_stride` before starting it.

Each worktree receives a stable numeric slot, allowing the same command set to
run in several worktrees without port conflicts. Base ports must have different
remainders when divided by `port_stride`, which prevents one command's port in
one slot from colliding with another command in a different slot. Redwood only
supplies `RW_PORT`; Doppler remains responsible for secrets.

## Commands

```text
rw create feature/foo   Create a worktree and assign its ports
rw start feature/foo    Start its commands in a detached tmux session
rw attach feature/foo   Attach to its tmux session
rw stop feature/foo     Stop its tmux session
rw list                 Show worktrees, ports, and running state
```

## TODO

### 1. CLI foundation

- [x] Create the `rw` executable and command dispatcher in Go.
- [x] Add consistent usage text, validation, and actionable error messages.
- [x] Locate and validate the main repository checkout for every command.
- [x] Load the committed `redwood.toml` and validate its required fields.
- [x] Add focused tests for command parsing and invalid configuration.

### 2. Repository and worktree discovery

- [ ] Wrap Git command execution so failures include useful context.
- [ ] Discover existing worktrees with `git worktree list --porcelain`.
- [ ] Resolve the repository name, main checkout, and shared Git directory.
- [ ] Represent each worktree with its path, branch, and current commit.
- [ ] Add parser tests using representative Git worktree output.

### 3. Stable slot and port allocation

- [ ] Define a small allocation file stored under the shared Git directory.
- [ ] Assign each new worktree the lowest available stable numeric slot.
- [ ] Preserve a worktree's slot across repeated Redwood invocations.
- [ ] Reconcile stored allocations with worktrees discovered from Git.
- [ ] Calculate every configured service port as `base + slot * port_stride`.
- [x] Define and document the port environment variable names passed to commands.
- [ ] Test stable allocation, multiple worktrees, and port calculations.

### 4. `rw create`

- [ ] Validate the requested branch name and reject duplicate worktrees.
- [ ] Expand `{repo}` and `{branch}` in `worktree_path` safely for the filesystem.
- [ ] Create or check out the branch from `base_branch` using `git worktree add`.
- [ ] Allocate and persist the worktree's slot after successful creation.
- [ ] Print the created path, slot, and assigned service ports.
- [ ] Roll back partial state when worktree creation or allocation fails.

### 5. tmux session lifecycle

- [ ] Generate a deterministic, collision-safe tmux session name per worktree.
- [ ] Implement `rw start` with one detached session per worktree.
- [ ] Create one tmux window per configured command.
- [ ] Run every command from the selected worktree directory.
- [ ] Supply only the calculated port environment variables to each command.
- [ ] Make `rw start` report an already-running session without duplicating it.
- [ ] Implement `rw attach` for the selected worktree's session.
- [ ] Implement `rw stop` by terminating the selected worktree's session.
- [ ] Report clear errors when tmux is unavailable or a session is not running.

### 6. `rw list`

- [ ] Combine discovered Git worktrees with stored slot allocations.
- [ ] Check tmux to determine whether each worktree session is running.
- [ ] Display branch, path, slot, service ports, and running state.
- [ ] Keep output readable and useful in scripts without adding a TUI.

### 7. End-to-end verification and documentation

- [ ] Add integration coverage around temporary Git repositories where practical.
- [ ] Verify all commands can be invoked from the main checkout.
- [ ] Verify two worktrees receive different ports and run concurrently.
- [ ] Verify processes remain running after detaching from tmux.
- [ ] Document installation, prerequisites, configuration, and the core workflow.
- [ ] Run formatting, tests, static analysis, and a clean build.

## MVP acceptance scenario

```sh
rw create feature/a
rw create feature/b
rw start feature/a
rw start feature/b
rw list
```

The two worktrees must run simultaneously on separate ports. Their processes
must continue running in detached tmux sessions until explicitly stopped.

## Deferred scope

The MVP will not manage databases, env vars, provide a proxy or TUI, integrate with
Docker, or delete branches. These features should not be introduced until the
core worktree, port allocation, and tmux workflow is complete and reliable.
