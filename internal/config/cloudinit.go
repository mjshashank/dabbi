package config

import (
	"fmt"
	"strings"

	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/network"
)

// GenerateCloudInitWithAuthToken injects the auth token into cloud-init
// It replaces the __DABBI_AUTH_TOKEN__ placeholder with the actual token
func GenerateCloudInitWithAuthToken(base string, authToken string) string {
	return strings.ReplaceAll(base, "__DABBI_AUTH_TOKEN__", authToken)
}

// GenerateCloudInitWithNetwork creates a cloud-init config with network rules
// It takes the base cloud-init content and appends network configuration
func GenerateCloudInitWithNetwork(base string, netConfig *multipass.NetworkConfig) (string, error) {
	if netConfig == nil || netConfig.Mode == multipass.NetworkModeNone {
		// No network restrictions, return base as-is
		return base, nil
	}

	// Validate the network config
	if err := network.ValidateConfig(netConfig); err != nil {
		return "", fmt.Errorf("invalid network config: %w", err)
	}

	// Generate the iptables script
	script, err := network.GenerateIptablesScript(netConfig)
	if err != nil {
		return "", fmt.Errorf("failed to generate iptables script: %w", err)
	}

	// Generate the systemd service
	service := network.GenerateSystemdService()

	// Generate config JSON
	configJSON := generateConfigJSON(netConfig)

	// Build the network setup section to append
	networkSection := buildNetworkSection(script, service, configJSON)

	// Append to base cloud-init
	return appendToCloudInit(base, networkSection), nil
}

func generateConfigJSON(config *multipass.NetworkConfig) string {
	// Simple JSON generation without external dependencies
	var rules []string
	for _, r := range config.Rules {
		comment := ""
		if r.Comment != "" {
			comment = fmt.Sprintf(`,"comment":"%s"`, escapeJSON(r.Comment))
		}
		rules = append(rules, fmt.Sprintf(`{"type":"%s","value":"%s"%s}`, r.Type, escapeJSON(r.Value), comment))
	}

	rulesJSON := "[]"
	if len(rules) > 0 {
		rulesJSON = "[" + strings.Join(rules, ",") + "]"
	}

	return fmt.Sprintf(`{"mode":"%s","rules":%s}`, config.Mode, rulesJSON)
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func buildNetworkSection(script, service, configJSON string) string {
	// Escape the script and service for YAML heredoc
	escapedScript := strings.ReplaceAll(script, "$", "\\$")

	return fmt.Sprintf(`
  # Dabbi network restrictions setup
  - mkdir -p /opt/dabbi/network
  - |
    cat > /opt/dabbi/network/config.json << 'DABBICONFIG'
%s
DABBICONFIG
  - |
    cat > /opt/dabbi/network/apply-rules.sh << 'DABBISCRIPT'
%s
DABBISCRIPT
  - chmod +x /opt/dabbi/network/apply-rules.sh
  - |
    cat > /etc/systemd/system/dabbi-network.service << 'DABBISERVICE'
%s
DABBISERVICE
  - systemctl daemon-reload
  - systemctl enable dabbi-network.service
  - /opt/dabbi/network/apply-rules.sh
`, configJSON, escapedScript, service)
}

func appendToCloudInit(base, networkSection string) string {
	// Find the runcmd section and append to it
	lines := strings.Split(base, "\n")
	result := make([]string, 0, len(lines)+50)
	inRuncmd := false
	inserted := false

	for _, line := range lines {
		// Check if we're entering the runcmd section
		if strings.HasPrefix(strings.TrimSpace(line), "runcmd:") {
			inRuncmd = true
			result = append(result, line)
			continue
		}

		// If we're in runcmd, look for the last item
		if inRuncmd {
			trimmed := strings.TrimSpace(line)
			// Check if we've exited runcmd (new top-level key or end of file)
			if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				// We've exited runcmd, insert network section before this line
				if !inserted {
					result = append(result, networkSection)
					inserted = true
				}
				inRuncmd = false
			}
		}

		result = append(result, line)
	}

	// If we were still in runcmd at end of file, append the network section
	if inRuncmd && !inserted {
		result = append(result, networkSection)
	}

	// If there was no runcmd section, add one
	if !inserted {
		result = append(result, "\nruncmd:")
		result = append(result, networkSection)
	}

	return strings.Join(result, "\n")
}
