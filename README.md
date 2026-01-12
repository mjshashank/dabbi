# dabbi

**Isolated Linux environments for AI coding agents - and everything else.**

```
     _       _     _     _
  __| | __ _| |__ | |__ (_)
 / _` |/ _` | '_ \| '_ \| |
| (_| | (_| | |_) | |_) | |
 \__,_|\__,_|_.__/|_.__/|_|
        (dub-bee)
```

AI coding agents are powerful but dangerous. They hallucinate commands, install untrusted packages, and modify system files with confidence. Running them on your actual machine is playing with fire.

dabbi gives you disposable sandboxes where AI agents can go full YOLO. Each sandbox is a lightweight, persistent Linux VM with its own filesystem and network. Restrict network access to specific hosts, or cut it off entirely. Snapshot before experiments, roll back when things break. Your host stays untouched.

But dabbi isn't just for AI agents. It's your personal cloud - isolated VMs you can access from any browser, share with anyone, and run on your laptop, a home server, or a cheap VPS.

## Let AI agents run wild

Let [opencode](https://opencode.ai), [Claude Code](https://docs.anthropic.com/en/docs/claude-code), [Aider](https://aider.chat), or any AI agent do whatever they want. `rm -rf /`? Sure. `curl | bash`? Go ahead. Random npm packages from sketchy repos? Have at it.

When (not if) things go sideways:

```bash
dabbi snapshot restore agent clean-slate  # Undo everything in seconds
```

## But also...

**Code from anywhere** - Access any VM from your phone, tablet, or browser. Full PTY terminal (vim, tmux, everything works). File browser with upload/download. No SSH keys, no VPN.

**Share apps with a link** - Running a server on port 3000 in a VM called `demo`? It's at `http://demo-3000.localhost`. On a VPS with a domain, your friends get a real HTTPS URL.

**Clone working setups** - Perfect dev environment? Clone it. Need 10 machines for a workshop? Clone them. Snapshot before letting an AI agent loose.

**Zero cost when idle** - VMs auto-pause after a configurable timeout. Paused VMs use no CPU or memory - only disk. They wake automatically when accessed.

## Features

| Feature                 | Description                                       |
| ----------------------- | ------------------------------------------------- |
| **AI Agent (opencode)** | Built-in coding agent accessible from any device  |
| **Web Terminal**        | Full PTY terminal in your browser                 |
| **File Browser**        | Browse, upload, download files from any device    |
| **Auto-Routing**        | `vm-port.localhost` routes to your VM             |
| **Wake-on-Request**     | Stopped VMs auto-start when accessed              |
| **Auto-Pause**          | VMs sleep after idle timeout                      |
| **Snapshots**           | Create, restore, delete - instant rollback        |
| **Cloning**             | Duplicate VMs with all their state                |
| **Mounts**              | Share host directories with VMs                   |
| **TCP Tunnels**         | Forward VM ports to localhost                     |
| **Network Control**     | Allowlist, blocklist, or fully isolate VM network |
| **HTTPS**               | Automatic Let's Encrypt certificates              |
| **Mobile-First UI**     | Manage everything from your phone                 |
| **Single Binary**       | Embedded UI, zero dependencies (except multipass) |

## Quick Start

**1. Install multipass** (the VM engine dabbi uses):

```bash
# macOS
brew install --cask multipass

# Linux
sudo snap install multipass
```

**2. Install dabbi:**

```bash
# Quick install
curl -fsSL https://raw.githubusercontent.com/mjshashank/dabbi/main/install.sh | bash

# Or with Homebrew
brew tap mjshashank/tap
brew install dabbi
```

**3. Start the daemon:**

```bash
dabbi serve
```

**4. Open the UI** at http://localhost and create your first VM.

Or use the CLI:

```bash
dabbi create dev
dabbi shell dev
```

## Examples

### AI Agent Sandbox

```bash
dabbi create agent --cpu 4 --mem 8G
dabbi snapshot create agent clean-slate
dabbi shell agent
# Let Claude Code do whatever it wants...
# Things went wrong?
dabbi snapshot restore agent clean-slate
```

## AI Agent (opencode)

Each dabbi VM comes with [opencode](https://opencode.ai) pre-installed and running. Click the **Agent** button in the UI to launch it. No setup required.

opencode's web UI is optimized for mobile - a great way to code from your phone or tablet.

### Claude Code

[Claude Code](https://docs.anthropic.com/en/docs/claude-code) is also pre-installed in every VM. Access it via the terminal:

```bash
dabbi shell myvm
claude   # Start Claude Code CLI
```

### Why run AI agents in a VM?

- **Safety** - AI agents run arbitrary commands. In a VM, they can't touch your host system.
- **Snapshots** - Take a snapshot before letting the agent loose. Roll back if things go wrong.
- **Access anywhere** - Work on your code from your phone, tablet, or any browser.
- **Isolation** - Each project gets its own VM with its own environment.

### Tips

- Mount your project directory: `dabbi mount add myvm ~/projects/myapp /home/ubuntu/myapp`
- Create a snapshot before experiments: `dabbi snapshot create myvm pre-experiment`
- Use network restrictions for extra safety: `dabbi network set myvm --mode allowlist --allow api.anthropic.com`

### Network-Restricted VM

```bash
dabbi create sandbox --network-mode isolated          # No internet at all
dabbi create build --network-mode allowlist --allow github.com  # Only GitHub
dabbi network set myvm --mode blocklist --block facebook.com    # Block specific sites
```

### Remote Dev Environment

Run dabbi on a VPS, access from anywhere:

```bash
# On your VPS
dabbi serve --domain dev.yourdomain.com

# From anywhere: open https://dev.yourdomain.com
# Full terminal, file browser, everything in your browser
```

### Share an App

```bash
dabbi create demo
dabbi shell demo
# Start your app on port 8080
npm run dev -- --port 8080
```

Share `http://demo-8080.localhost` (or `https://demo-8080.yourdomain.com` on a VPS).

### Workshop Setup

```bash
# Create a template with everything installed
dabbi create template
dabbi shell template
# Install tools, clone repos, configure...

# Clone for each participant
for i in {1..10}; do dabbi clone template student-$i; done
```

## CLI Reference

```bash
# Daemon
dabbi serve [--port 80] [--domain example.com]

# VM Lifecycle
dabbi list
dabbi create <name> [--cpu 2] [--mem 4G] [--disk 20G]
dabbi start|stop|restart|delete <name>
dabbi shell <name>
dabbi clone <source> <new-name>

# AI Agent
dabbi agent <name>                    # Open interactive opencode session in VM

# Snapshots
dabbi snapshot list <vm>
dabbi snapshot create <vm> [name]
dabbi snapshot restore <vm> <name>
dabbi snapshot delete <vm> <name>

# Files
dabbi cp ./local.txt vm:/path/remote.txt
dabbi cp vm:/path/remote.txt ./local.txt

# Mounts
dabbi mount add <vm> /host/path /vm/path
dabbi mount remove <vm> /vm/path

# Tunnels
dabbi tunnel <vm> <port>

# Network Restrictions
dabbi network get <vm>
dabbi network set <vm> --mode <none|allowlist|blocklist|isolated> [--allow host] [--block host]
dabbi network remove <vm>
dabbi network apply <vm>
```

## Configuration

Config lives at `~/.dabbi/config.json`:

```json
{
  "auth_token": "auto-generated-uuid",
  "defaults": {
    "cpu": 2,
    "mem": "4G",
    "disk": "20G",
    "network": {
      "mode": "none",
      "rules": []
    }
  },
  "shutdown_timeout_mins": 30
}
```

Network modes:

- `none` - No restrictions (default)
- `allowlist` - Only allow specified hosts (requires rules)
- `blocklist` - Block specified hosts (requires rules)
- `isolated` - No network access except host communication

Rules can be domains (`github.com`), IPs (`192.168.1.1`), or CIDRs (`10.0.0.0/8`).

Customize new VMs with `~/.dabbi/cloud-init.yaml` - install your tools, set up dotfiles, etc.

## Deployment

### Local (Laptop/Desktop)

```bash
dabbi serve
# http://localhost
# http://vm-port.localhost
```

### VPS with HTTPS

```bash
# Point your domain to your VPS
dabbi serve --domain yourdomain.com
# Automatic Let's Encrypt certificates
# https://yourdomain.com
```

### Behind Tailscale

```bash
dabbi serve
# Access via Tailscale IP from anywhere
# Already encrypted, no domain needed
```

## How It Works

```
┌───────────────────────────────────────────────────────┐
│                    dabbi daemon                        │
│                                                        │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐       │
│  │   Web UI   │  │  REST API  │  │ WebSocket  │       │
│  │ (embedded) │  │   /api/*   │  │  Terminal  │       │
│  └────────────┘  └────────────┘  └────────────┘       │
│                                                        │
│  ┌────────────────────────────────────────────┐       │
│  │            HTTP Proxy Router                │       │
│  │      vm-port.localhost → VM:port           │       │
│  └────────────────────────────────────────────┘       │
│                                                        │
│  ┌────────────────────────────────────────────┐       │
│  │         Watchdog (Auto-pause)               │       │
│  │      Stop idle VMs after timeout           │       │
│  └────────────────────────────────────────────┘       │
│                                                        │
└───────────────────────────────────────────────────────┘
                         │
                         ▼
                 ┌──────────────┐
                 │   multipass  │
                 └──────────────┘
                         │
           ┌─────────────┼─────────────┐
           ▼             ▼             ▼
       ┌──────┐     ┌──────┐     ┌──────┐
       │ VM 1 │     │ VM 2 │     │ VM 3 │
       └──────┘     └──────┘     └──────┘
```

dabbi is a thin layer on top of [multipass](https://multipass.run). Multipass handles the VMs. dabbi adds the web UI, remote access, auto-routing, snapshots, and idle management.

## Security

- **Auth token** required for all API/UI access
- **HttpOnly cookies** for browser sessions
- **Origin validation** prevents cross-site attacks
- **VMs are isolated** from your host by default
- **Network restrictions** - allowlist, blocklist, or fully isolate VM network access
- **HTTPS** with automatic certificates on public deployments
- **Works with Tailscale** for zero-trust access

## Building from Source

```bash
git clone https://github.com/mjshashank/dabbi
cd dabbi
make          # Build everything
make test     # Run tests
```

## Why "dabbi"?

"dabbi" (ಡಬ್ಬಿ, pronounced _dub-bee_) means "small box" in Kannada. Each VM is a little box you can do whatever you want with.

## License

MIT

---
