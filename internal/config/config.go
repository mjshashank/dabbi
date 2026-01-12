package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/mjshashank/dabbi/internal/multipass"
)

const (
	ConfigDir            = ".dabbi"
	ConfigFile           = "config.json"
	DefaultCloudInitFile = "cloud-init.yaml"
)

// Config holds the application configuration
type Config struct {
	AuthToken           string   `json:"auth_token"`
	Defaults            Defaults `json:"defaults"`
	ShutdownTimeoutMins int      `json:"shutdown_timeout_mins"`
}

// Defaults holds default VM configuration
type Defaults struct {
	CPU           int                      `json:"cpu"`
	Mem           string                   `json:"mem"`
	Disk          string                   `json:"disk"`
	CloudInit     string                   `json:"cloud_init,omitempty"`  // path to default cloud-init file
	NetworkConfig *multipass.NetworkConfig `json:"network,omitempty"`     // default network restrictions
}

// DefaultConfig returns a new config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		AuthToken: uuid.New().String(),
		Defaults: Defaults{
			CPU:  2,
			Mem:  "4G",
			Disk: "20G",
		},
		ShutdownTimeoutMins: 5,
	}
}

// ConfigPath returns the path to the config file
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ConfigDir, ConfigFile), nil
}

// DefaultCloudInitPath returns the path to the default cloud-init file
func DefaultCloudInitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ConfigDir, DefaultCloudInitFile), nil
}

// GetCloudInitPath returns the cloud-init path to use
// Priority: explicit path > config default > ~/.dabbi/cloud-init.yaml (if exists)
func (c *Config) GetCloudInitPath(explicit string) string {
	// Explicit path takes priority
	if explicit != "" {
		return explicit
	}

	// Config default takes second priority
	if c.Defaults.CloudInit != "" {
		if _, err := os.Stat(c.Defaults.CloudInit); err == nil {
			return c.Defaults.CloudInit
		}
	}

	// Check for default cloud-init file in config dir
	defaultPath, err := DefaultCloudInitPath()
	if err != nil {
		return ""
	}
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}

	return ""
}

// DefaultCloudInit is the default cloud-init configuration
const DefaultCloudInit = `#cloud-config
# Default dabbi cloud-init configuration
# Edit this file to customize all new VMs

# Update package cache (needed for installs)
package_update: true

# Essential packages only - rest installed in background
packages:
  - git
  - curl
  - wget
  - jq
  - neovim
  - tmux
  - fzf
  - bash-completion

# Configure the default ubuntu user
users:
  - default
  - name: ubuntu
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash

# Set timezone (change as needed)
timezone: UTC

# Run commands on first boot
runcmd:
  # Ensure ubuntu home directory has correct ownership
  - mkdir -p /home/ubuntu/.config /home/ubuntu/.local/bin /home/ubuntu/.bashrc.d
  - chown -R ubuntu:ubuntu /home/ubuntu
  # Git config
  - sudo -u ubuntu git config --global init.defaultBranch main
  - sudo -u ubuntu git config --global color.ui auto
  # Setup .bashrc.d sourcing
  - echo '' >> /home/ubuntu/.bashrc
  - echo '# Source all files in ~/.bashrc.d' >> /home/ubuntu/.bashrc
  - echo 'for f in ~/.bashrc.d/*.sh; do [ -r "$f" ] && . "$f"; done' >> /home/ubuntu/.bashrc
  # Write bashrc defaults
  - |
    cat > /home/ubuntu/.bashrc.d/dabbi-defaults.sh << 'BASHRC'
    # Fix for unknown terminal types (e.g., ghostty)
    if ! infocmp "$TERM" &>/dev/null; then
      export TERM=xterm-256color
    fi

    # Shell variables
    export EDITOR=nvim
    export VISUAL=nvim
    export HISTSIZE=10000
    export HISTFILESIZE=20000
    export HISTCONTROL=ignoreboth:erasedups
    export PATH="$HOME/.local/bin:$PATH"

    # Enable bash completion
    if [ -f /usr/share/bash-completion/bash_completion ]; then
      . /usr/share/bash-completion/bash_completion
    elif [ -f /etc/bash_completion ]; then
      . /etc/bash_completion
    fi

    # fzf keybindings and completion
    [ -f /usr/share/doc/fzf/examples/key-bindings.bash ] && . /usr/share/doc/fzf/examples/key-bindings.bash
    [ -f /usr/share/doc/fzf/examples/completion.bash ] && . /usr/share/doc/fzf/examples/completion.bash

    # mise activation
    [ -f "$HOME/.local/bin/mise" ] && eval "$($HOME/.local/bin/mise activate bash)"

    # Simple PS1 with git branch
    parse_git_branch() { git branch 2>/dev/null | sed -e '/^[^*]/d' -e 's/* \(.*\)/ (\1)/'; }
    export PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[33m\]$(parse_git_branch)\[\033[00m\]\$ '

    # OpenCode authentication (injected by dabbi)
    export OPENCODE_SERVER_PASSWORD="__DABBI_AUTH_TOKEN__"
    BASHRC
  - chown ubuntu:ubuntu /home/ubuntu/.bashrc.d/dabbi-defaults.sh
  # Write background install script
  - |
    cat > /opt/dabbi-install.sh << 'SCRIPT'
    #!/bin/bash
    set -e
    LOG=/var/log/dabbi-install.log
    exec > >(tee -a $LOG) 2>&1
    echo "[$(date)] Starting background tool installation..."

    # Install htop, unzip
    echo "[$(date)] Installing additional packages..."
    apt-get install -y htop unzip

    # Install GitHub CLI
    echo "[$(date)] Installing GitHub CLI..."
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
    chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null
    apt-get update && apt-get install -y gh

    # Install mise (for ubuntu user)
    echo "[$(date)] Installing mise..."
    sudo -u ubuntu bash -c 'curl https://mise.run | sh'

    # Install Claude Code (for ubuntu user)
    echo "[$(date)] Installing Claude Code..."
    sudo -u ubuntu bash -c 'curl -fsSL https://claude.ai/install.sh | bash'

    # Install OpenCode (for ubuntu user)
    echo "[$(date)] Installing OpenCode..."
    sudo -u ubuntu bash -c 'curl -fsSL https://opencode.ai/install | bash'

    echo "[$(date)] Background installation complete!"
    touch /home/ubuntu/.dabbi-install-complete
    chown ubuntu:ubuntu /home/ubuntu/.dabbi-install-complete
    SCRIPT
  - chmod +x /opt/dabbi-install.sh
  # Run install script synchronously (VM won't be ready until complete)
  - /opt/dabbi-install.sh
  # Create OpenCode systemd service for web UI
  - |
    cat > /etc/systemd/system/dabbi-opencode.service << 'OPENCODESVC'
    [Unit]
    Description=OpenCode Web Server
    After=network.target

    [Service]
    Type=simple
    User=ubuntu
    WorkingDirectory=/home/ubuntu
    Environment="HOME=/home/ubuntu"
    Environment="OPENCODE_SERVER_PASSWORD=__DABBI_AUTH_TOKEN__"
    ExecStart=/home/ubuntu/.opencode/bin/opencode web --port 1234 --hostname 0.0.0.0
    Restart=always
    RestartSec=10

    [Install]
    WantedBy=multi-user.target
    OPENCODESVC
  - systemctl daemon-reload
  - systemctl enable dabbi-opencode.service
  - systemctl start dabbi-opencode.service || true

# Optional: Add your SSH keys for passwordless access
# ssh_authorized_keys:
#   - ssh-rsa AAAA... your-key-here
`

// EnsureDefaultCloudInit creates the default cloud-init file if it doesn't exist
// Returns the path to the file and whether it was created
func EnsureDefaultCloudInit() (string, bool, error) {
	path, err := DefaultCloudInitPath()
	if err != nil {
		return "", false, err
	}

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return path, false, nil
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", false, err
	}

	// Write default cloud-init
	if err := os.WriteFile(path, []byte(DefaultCloudInit), 0644); err != nil {
		return "", false, err
	}

	return path, true, nil
}

// Load loads the configuration from disk, creating a default one if it doesn't exist
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Create default config
		cfg := DefaultConfig()
		if err := cfg.Save(); err != nil {
			return nil, err
		}
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save persists the configuration to disk
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists with restrictive permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// Write with restrictive permissions (contains auth token)
	return os.WriteFile(path, data, 0600)
}
