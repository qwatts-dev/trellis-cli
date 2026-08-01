# trellis-cli (WSL2 Fork)

> **This is a fork of [roots/trellis-cli](https://github.com/roots/trellis-cli)** that adds native **WSL2 virtual machine support for Windows**. The upstream CLI supports Lima (macOS/Linux). This fork adds a `wsl` backend that manages WSL2 distros via `wsl.exe`, giving Windows developers a first-class Trellis development experience.

[![Upstream](https://img.shields.io/badge/upstream-roots%2Ftrellis--cli-blue?style=flat-square)](https://github.com/roots/trellis-cli)

---

## What's New: WSL2 VM Backend

### Overview

Windows developers can now run `trellis vm start` to get a fully provisioned Trellis development environment powered by WSL2. Each project gets its own isolated Ubuntu distro with nginx, PHP-FPM, MariaDB, and all Trellis services — no manual WSL setup required.

### Requirements

- **Windows 11** with WSL2 enabled
- **VS Code** with the [WSL extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-wsl) (required for editing project files)
- **trellis-cli** (this fork)

> **Important:** Project files live on WSL2's native ext4 filesystem for performance. You must use an editor that supports WSL remote development. VS Code with the WSL extension is the recommended (and automated) path. JetBrains IDEs also support WSL remoting but are not automated by `vm open`.

### Quick Start

```powershell
# Create a new Trellis project (from PowerShell)
trellis new mysite.com

# Start the VM (imports Ubuntu, bootstraps Ansible, provisions everything)
cd mysite.com
trellis vm start

# Open VS Code connected to the WSL distro
trellis vm open

# From the VS Code integrated terminal (inside WSL):
trellis provision development    # Re-provision
trellis db open --app=tableplus  # Open database in TablePlus
```

### Windows/WSL Development Workflow

> **Important for Windows developers:** The development workflow differs slightly from macOS/Lima. On macOS, project files live on a shared filesystem, so dependency installs (composer, yarn) and frontend build tools run on the host. On WSL2, project files live on the distro's native ext4 filesystem for performance. **All dependency installs and build commands should be run inside the WSL terminal** (via `trellis vm open` or `trellis vm shell`).

After `vm start` and `vm open`, run your project's setup steps from the VS Code integrated terminal:

```bash
# Example: typical Bedrock + Sage project
cd site && composer install
cd web/app/themes/my-theme && composer install && yarn install
yarn dev   # Frontend asset watcher
```

Node.js LTS and Corepack (yarn/pnpm) are pre-installed in every WSL distro. You do **not** need Node.js on Windows for Trellis development. If your project requires additional CLI tools, you can install them directly in your WSL distro via `trellis vm shell` or the VS Code terminal.

### Commands

**New commands** (WSL2 only):

| Command    | Run From | Description                                                   |
|------------|----------|---------------------------------------------------------------|
| `vm open`  | Windows  | Opens VS Code connected to the WSL distro at the project root |
| `vm sync`  | Windows  | Manually syncs project files from WSL back to Windows         |
| `vm trust` | Windows  | Re-imports self-signed SSL certs into the Windows trust store |

**Enhanced for WSL2** (existing commands with added Windows-specific behavior):

| Command           | What Changed                                                                      |
|-------------------|-----------------------------------------------------------------------------------|
| `vm start`        | WSL2 backend: imports Ubuntu distro, bootstraps to ext4, auto-stops other distros |
| `vm stop`         | Auto SyncBack (rsync ext4 → Windows) before terminating the distro                |
| `vm delete`       | Cleans up Windows hosts file entries and SSL certs                                |
| `vm shell`        | Routes to WSL distro; detects when run from wrong host                            |
| `db open`         | Works from both Windows and WSL; uses direct `mysql://` URI (no SSH tunnels)      |
| `provision`       | Detects Windows host and redirects to WSL terminal                                |
| `deploy`          | Detects Windows host and redirects to WSL terminal                                |
| `vault *`         | Detects Windows host and redirects to WSL terminal                                |
| `galaxy install`  | Detects Windows host and redirects to WSL terminal                                |
| `xdebug-tunnel *` | Detects Windows host and redirects to WSL terminal                                |

### How It Works

1. **`vm start`** imports an Ubuntu rootfs into a dedicated WSL2 distro (e.g., `trellis-mysite-com`), installs Python/Ansible, copies the project to ext4, runs `ansible-playbook dev.yml`, tunes opcache, trusts SSL certs, and updates the Windows hosts file.

2. **Project files live on ext4** at `/home/admin/<project>/` inside the distro. This gives native filesystem performance (~80ms page loads vs ~14 seconds with Windows filesystem mounts). The `site/` directory is bind-mounted to `/srv/www/<site>/current` as Trellis expects.

3. **`vm open`** launches VS Code with `--folder-uri vscode-remote://wsl+<distro>/home/admin/<project>`, connecting directly to the WSL distro. The developer sees the full project (trellis/ + site/ + .git/) and uses git normally from the VS Code terminal.

4. **`vm stop`** runs an incremental rsync from WSL ext4 back to the Windows filesystem before stopping the distro, keeping the Windows-side repo up to date for GitHub Desktop or other Windows git tools.

5. **Smart command routing** — Ansible-dependent commands (provision, deploy, vault, etc.) detect when you're on the Windows host and tell you to run them from the WSL terminal instead. VM management commands detect when you're inside WSL and redirect you to Windows.

### Features

- **Ext4-native performance** — ~80ms TTFB (vs ~14s with DrvFS/9p bind mounts)
- **Automatic hosts file management** — Adds/removes `*.test` entries in the Windows hosts file (UAC elevation, only when entries change)
- **SSL certificate trust** — Self-signed certs auto-imported into the Windows Trusted Root CA store (sites must have `ssl.enabled: true` in `wordpress_sites.yml`)
- **Bi-directional file sync** — Auto sync on stop; manual `vm sync`; config auto-sync on any Windows-side command
- **Database GUI support** — `db open --app=tableplus` works from both Windows and WSL terminals, using direct `mysql://` URIs (no SSH tunnels needed)
- **Cross-compiled Linux binary** — Automatically deployed into distros so `trellis` commands work from the WSL terminal
- **Distro isolation** — Each project gets its own WSL distro; multiple projects can run simultaneously
- **Resilient lifecycle** — Detects unprovisioned distros and auto-cleans; keepalive process prevents WSL idle shutdown

### Architecture

```
Windows Host                          WSL2 Distro (trellis-mysite-com)
─────────────                         ─────────────────────────────────
trellis vm start ──────────────────── wsl --import → bootstrap → provision
trellis vm open  ──────────────────── code --folder-uri vscode-remote://wsl+...
trellis vm stop  ── rsync ext4→Win ── wsl -t (terminate)
trellis vm trust ── certutil ───────── reads /etc/nginx/ssl/*.cert
trellis db open  ── rundll32 URI ───── ansible-playbook → JSON credentials

C:\Users\...\mysite.com\             /home/admin/mysite.com/
  trellis/  (config, read by Win)       trellis/  (config, used by Ansible)
  site/     (Windows backup)            site/     (ext4, served by nginx)
  .git/     (Windows backup)            .git/     (ext4, used by VS Code)
```

### Configuration

The WSL backend is auto-selected on Windows. You can explicitly set it in `trellis.cli.yml`:

```yaml
vm:
  manager: "wsl"    # "auto" also works (selects wsl on Windows, lima on macOS)
  ubuntu: "24.04"   # Ubuntu version for the rootfs (22.04 or 24.04)
```

### Differences from Lima (macOS)

| Aspect             | Lima (macOS)            | WSL2 (Windows)                |
|--------------------|-------------------------|-------------------------------|
| VM technology      | QEMU/Lima               | WSL2 (Hyper-V)                |
| Filesystem         | virtiofs (FUSE)         | ext4 native                   |
| Networking         | Lima port forwarding    | WSL2 NAT (automatic)          |
| Editor requirement | Any (shared filesystem) | VS Code + WSL extension       |
| SSH                | Lima manages SSH keys   | Not needed (local connection) |
| Ansible connection | `local`                 | `local`                       |

### Known Limitations

- **One project at a time** — All WSL2 distros share a single network stack ([by design](https://github.com/microsoft/WSL/issues/4304)), so services like MariaDB (3306), nginx (80/443), and openssh-server (22) conflict if multiple distros run simultaneously. `vm start` automatically stops other `trellis-*` distros before starting yours, with an optional SyncBack prompt so you can sync unsaved work back to Windows first. Your data is safe — stopped distros resume exactly where they left off.
- **VS Code is required** for editing project files (they live on WSL2 ext4, not the Windows filesystem)
- **Windows-side files are a backup** — the Windows copy is kept in sync by `vm stop` and `vm sync` but is not the source of truth during development
- **One UAC prompt** per `vm start` (for hosts file and SSL cert trust) — subsequent starts skip UAC if entries haven't changed

---

## Upstream documentation

This fork adds a native WSL2 backend for Windows. For everything else - installation, configuration, the full command reference, provisioning, and deployment - see the upstream project this fork is based on:

**[roots/trellis-cli README](https://github.com/roots/trellis-cli#readme)**

