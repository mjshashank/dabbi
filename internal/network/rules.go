package network

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/mjshashank/dabbi/internal/multipass"
)

// scriptTemplate is the template for the iptables setup script
const scriptTemplate = `#!/bin/bash
# Dabbi Network Rules
# Mode: {{.Mode}}
# Generated automatically - do not edit manually

set -e

# Flush existing rules
iptables -F OUTPUT 2>/dev/null || true
iptables -F INPUT 2>/dev/null || true
iptables -F DABBI_OUT 2>/dev/null || true

# Delete and recreate custom chain
iptables -X DABBI_OUT 2>/dev/null || true
iptables -N DABBI_OUT 2>/dev/null || true

{{if eq .Mode "isolated"}}
# ISOLATED MODE - No network access
iptables -P OUTPUT DROP
iptables -P INPUT DROP

# Allow loopback
iptables -A OUTPUT -o lo -j ACCEPT
iptables -A INPUT -i lo -j ACCEPT

# Allow established connections (for multipass communication)
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow multipass bridge network (host-VM communication)
# Auto-detect the default gateway network
GATEWAY_IP=$(ip route | grep default | awk '{print $3}')
if [ -n "$GATEWAY_IP" ]; then
    GATEWAY_NET=$(echo "$GATEWAY_IP" | sed 's/\.[0-9]*$/.0\/24/')
    iptables -A OUTPUT -d "$GATEWAY_NET" -j ACCEPT
    iptables -A INPUT -s "$GATEWAY_NET" -j ACCEPT
fi

{{else if eq .Mode "allowlist"}}
# ALLOWLIST MODE - Default deny, allow specific hosts
iptables -P OUTPUT DROP
iptables -P INPUT DROP

# Allow loopback
iptables -A OUTPUT -o lo -j ACCEPT
iptables -A INPUT -i lo -j ACCEPT

# Allow established connections
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow multipass bridge network (required for host-VM communication)
# Auto-detect the default gateway network
GATEWAY_IP=$(ip route | grep default | awk '{print $3}')
if [ -n "$GATEWAY_IP" ]; then
    GATEWAY_NET=$(echo "$GATEWAY_IP" | sed 's/\.[0-9]*$/.0\/24/')
    iptables -A OUTPUT -d "$GATEWAY_NET" -j ACCEPT
    iptables -A INPUT -s "$GATEWAY_NET" -j ACCEPT
fi

# Allow DNS for domain resolution (to local resolver and common DNS servers)
iptables -A OUTPUT -p udp --dport 53 -j ACCEPT
iptables -A OUTPUT -p tcp --dport 53 -j ACCEPT

# Jump to custom chain for user rules
iptables -A OUTPUT -j DABBI_OUT

# User-defined allow rules
{{range .Rules}}
{{if eq .Type "ip"}}
# Allow IP: {{.Value}}{{if .Comment}} - {{.Comment}}{{end}}
iptables -A DABBI_OUT -d {{.Value}} -j ACCEPT
{{else if eq .Type "cidr"}}
# Allow CIDR: {{.Value}}{{if .Comment}} - {{.Comment}}{{end}}
iptables -A DABBI_OUT -d {{.Value}} -j ACCEPT
{{else if eq .Type "domain"}}
# Allow domain: {{.Value}}{{if .Comment}} - {{.Comment}}{{end}}
# Resolve and allow all IPs for this domain
for ip in $(dig +short {{.Value}} A 2>/dev/null | grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'); do
    iptables -A DABBI_OUT -d "$ip" -j ACCEPT 2>/dev/null || true
done
for ip in $(dig +short {{.Value}} AAAA 2>/dev/null | grep -v '\.$'); do
    ip6tables -A DABBI_OUT -d "$ip" -j ACCEPT 2>/dev/null || true
done
{{end}}
{{end}}

{{else if eq .Mode "blocklist"}}
# BLOCKLIST MODE - Default allow, block specific hosts
iptables -P OUTPUT ACCEPT
iptables -P INPUT ACCEPT

# Jump to custom chain for user rules
iptables -A OUTPUT -j DABBI_OUT

# User-defined block rules
{{range .Rules}}
{{if eq .Type "ip"}}
# Block IP: {{.Value}}{{if .Comment}} - {{.Comment}}{{end}}
iptables -A DABBI_OUT -d {{.Value}} -j DROP
{{else if eq .Type "cidr"}}
# Block CIDR: {{.Value}}{{if .Comment}} - {{.Comment}}{{end}}
iptables -A DABBI_OUT -d {{.Value}} -j DROP
{{else if eq .Type "domain"}}
# Block domain: {{.Value}}{{if .Comment}} - {{.Comment}}{{end}}
# Resolve and block all IPs for this domain
for ip in $(dig +short {{.Value}} A 2>/dev/null | grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'); do
    iptables -A DABBI_OUT -d "$ip" -j DROP 2>/dev/null || true
done
for ip in $(dig +short {{.Value}} AAAA 2>/dev/null | grep -v '\.$'); do
    ip6tables -A DABBI_OUT -d "$ip" -j DROP 2>/dev/null || true
done
{{end}}
{{end}}

{{else}}
# NONE MODE - No restrictions (permissive)
iptables -P OUTPUT ACCEPT
iptables -P INPUT ACCEPT
{{end}}

echo "Network rules applied successfully (mode: {{.Mode}})"
`

// systemdServiceTemplate is the template for the systemd service
const systemdServiceTemplate = `[Unit]
Description=Dabbi Network Rules
After=network.target

[Service]
Type=oneshot
ExecStart=/opt/dabbi/network/apply-rules.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
`

// GenerateIptablesScript generates a shell script to apply iptables rules
func GenerateIptablesScript(config *multipass.NetworkConfig) (string, error) {
	if config == nil {
		config = &multipass.NetworkConfig{Mode: multipass.NetworkModeNone}
	}

	tmpl, err := template.New("iptables").Parse(scriptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// GenerateSystemdService returns the systemd service unit file content
func GenerateSystemdService() string {
	return systemdServiceTemplate
}

// ValidateConfig validates a network configuration
func ValidateConfig(config *multipass.NetworkConfig) error {
	if config == nil {
		return nil
	}

	switch config.Mode {
	case multipass.NetworkModeNone, multipass.NetworkModeIsolated:
		// These modes don't need rules
		return nil
	case multipass.NetworkModeAllowlist, multipass.NetworkModeBlocklist:
		// These modes require at least one rule (or it would be confusing)
		if len(config.Rules) == 0 {
			return fmt.Errorf("mode %q requires at least one rule", config.Mode)
		}
	default:
		return fmt.Errorf("invalid network mode: %q", config.Mode)
	}

	// Validate each rule
	for i, rule := range config.Rules {
		if err := validateRule(&rule); err != nil {
			return fmt.Errorf("rule %d: %w", i+1, err)
		}
	}

	return nil
}

func validateRule(rule *multipass.NetworkRule) error {
	if rule.Value == "" {
		return fmt.Errorf("rule value cannot be empty")
	}

	switch rule.Type {
	case "ip":
		// Basic IP validation
		if !isValidIP(rule.Value) {
			return fmt.Errorf("invalid IP address: %q", rule.Value)
		}
	case "cidr":
		// Basic CIDR validation
		if !strings.Contains(rule.Value, "/") {
			return fmt.Errorf("CIDR must contain /: %q", rule.Value)
		}
	case "domain":
		// Basic domain validation
		if strings.Contains(rule.Value, " ") {
			return fmt.Errorf("domain cannot contain spaces: %q", rule.Value)
		}
	default:
		return fmt.Errorf("invalid rule type: %q (must be ip, cidr, or domain)", rule.Type)
	}

	return nil
}

func isValidIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		// Parse the octet value
		val := 0
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
			val = val*10 + int(c-'0')
		}
		// Validate range 0-255
		if val > 255 {
			return false
		}
		// Reject leading zeros (except "0" itself)
		if len(part) > 1 && part[0] == '0' {
			return false
		}
	}
	return true
}
