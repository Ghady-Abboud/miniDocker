# miniDock

A container runtime built from scratch in Go. No Docker, no containerd, no abstractions — just raw Linux kernel primitives doing what they've always done.

## What it does

Spins up an isolated container running Alpine Linux with:

- **PID, UTS, and mount namespace isolation** via Linux `clone` flags
- **Filesystem isolation** using `pivot_root` into an Alpine root FS
- **Resource limits** enforced through cgroups v2 — 50MB memory cap, 20% CPU quota
- **`/proc` mounted** inside the container so it sees only its own processes

The parent process re-executes itself as a child across the namespace boundary (the same pattern `runc` uses), sets up the cgroup, then waits. The child does the rootFS pivot and drops into a shell.

## Stack

- Go
- Linux namespaces (`CLONE_NEWPID`, `CLONE_NEWUTS`, `CLONE_NEWNS`)
- cgroups v2
- Alpine Linux root filesystem

## Why

Wanted to understand what Docker actually does under the hood. Turns out it's just namespace syscalls, a `pivot_root`, and some cgroup writes. This is that, stripped down to the minimum.

## Usage

Needs Linux (uses Linux-specific syscalls). Run as root.

```bash
go build -o miniDock
sudo ./miniDock
```

You'll get a shell inside an isolated Alpine container.

## What you could use this for

- Base for a custom container runtime or sandbox
- Lightweight isolated execution environment
- Learning how container security boundaries actually work
- Starting point for adding seccomp filtering, network namespaces, or rootless support
