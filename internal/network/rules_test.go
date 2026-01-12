package network

import (
	"strings"
	"testing"

	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateIptablesScript(t *testing.T) {
	tests := []struct {
		name     string
		config   *multipass.NetworkConfig
		contains []string
		excludes []string
	}{
		{
			name:   "nil_config_defaults_to_none",
			config: nil,
			contains: []string{
				"Mode: none",
				"iptables -P OUTPUT ACCEPT",
				"iptables -P INPUT ACCEPT",
				"NONE MODE - No restrictions",
			},
			excludes: []string{
				"DROP",
				"DABBI_OUT -d",
			},
		},
		{
			name:   "none_mode",
			config: &multipass.NetworkConfig{Mode: multipass.NetworkModeNone},
			contains: []string{
				"Mode: none",
				"iptables -P OUTPUT ACCEPT",
				"iptables -P INPUT ACCEPT",
				"NONE MODE - No restrictions",
			},
			excludes: []string{
				"DROP",
			},
		},
		{
			name:   "isolated_mode",
			config: &multipass.NetworkConfig{Mode: multipass.NetworkModeIsolated},
			contains: []string{
				"Mode: isolated",
				"ISOLATED MODE - No network access",
				"iptables -P OUTPUT DROP",
				"iptables -P INPUT DROP",
				"iptables -A OUTPUT -o lo -j ACCEPT",
				"iptables -A INPUT -i lo -j ACCEPT",
				"state ESTABLISHED,RELATED -j ACCEPT",
				"GATEWAY_IP=$(ip route | grep default",
			},
			excludes: []string{
				"DABBI_OUT -d",
			},
		},
		{
			name: "allowlist_with_ip",
			config: &multipass.NetworkConfig{
				Mode: multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{
					{Type: "ip", Value: "8.8.8.8", Comment: "Google DNS"},
				},
			},
			contains: []string{
				"Mode: allowlist",
				"ALLOWLIST MODE - Default deny",
				"iptables -P OUTPUT DROP",
				"Allow DNS for domain resolution",
				"iptables -A OUTPUT -p udp --dport 53 -j ACCEPT",
				"iptables -A OUTPUT -j DABBI_OUT",
				"# Allow IP: 8.8.8.8 - Google DNS",
				"iptables -A DABBI_OUT -d 8.8.8.8 -j ACCEPT",
			},
		},
		{
			name: "allowlist_with_cidr",
			config: &multipass.NetworkConfig{
				Mode: multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{
					{Type: "cidr", Value: "10.0.0.0/8"},
				},
			},
			contains: []string{
				"# Allow CIDR: 10.0.0.0/8",
				"iptables -A DABBI_OUT -d 10.0.0.0/8 -j ACCEPT",
			},
		},
		{
			name: "allowlist_with_domain",
			config: &multipass.NetworkConfig{
				Mode: multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{
					{Type: "domain", Value: "github.com", Comment: "GitHub"},
				},
			},
			contains: []string{
				"# Allow domain: github.com - GitHub",
				"dig +short github.com A",
				"iptables -A DABBI_OUT -d \"$ip\" -j ACCEPT",
				"ip6tables -A DABBI_OUT -d \"$ip\" -j ACCEPT",
			},
		},
		{
			name: "allowlist_multiple_rules",
			config: &multipass.NetworkConfig{
				Mode: multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{
					{Type: "ip", Value: "1.1.1.1"},
					{Type: "domain", Value: "api.github.com"},
					{Type: "cidr", Value: "192.168.0.0/16"},
				},
			},
			contains: []string{
				"iptables -A DABBI_OUT -d 1.1.1.1 -j ACCEPT",
				"dig +short api.github.com A",
				"iptables -A DABBI_OUT -d 192.168.0.0/16 -j ACCEPT",
			},
		},
		{
			name: "blocklist_with_ip",
			config: &multipass.NetworkConfig{
				Mode: multipass.NetworkModeBlocklist,
				Rules: []multipass.NetworkRule{
					{Type: "ip", Value: "1.2.3.4", Comment: "Bad IP"},
				},
			},
			contains: []string{
				"Mode: blocklist",
				"BLOCKLIST MODE - Default allow",
				"iptables -P OUTPUT ACCEPT",
				"iptables -A OUTPUT -j DABBI_OUT",
				"# Block IP: 1.2.3.4 - Bad IP",
				"iptables -A DABBI_OUT -d 1.2.3.4 -j DROP",
			},
			excludes: []string{
				"-j ACCEPT",
			},
		},
		{
			name: "blocklist_with_cidr",
			config: &multipass.NetworkConfig{
				Mode: multipass.NetworkModeBlocklist,
				Rules: []multipass.NetworkRule{
					{Type: "cidr", Value: "10.0.0.0/8"},
				},
			},
			contains: []string{
				"# Block CIDR: 10.0.0.0/8",
				"iptables -A DABBI_OUT -d 10.0.0.0/8 -j DROP",
			},
		},
		{
			name: "blocklist_with_domain",
			config: &multipass.NetworkConfig{
				Mode: multipass.NetworkModeBlocklist,
				Rules: []multipass.NetworkRule{
					{Type: "domain", Value: "malware.com"},
				},
			},
			contains: []string{
				"# Block domain: malware.com",
				"dig +short malware.com A",
				"iptables -A DABBI_OUT -d \"$ip\" -j DROP",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := GenerateIptablesScript(tt.config)
			require.NoError(t, err)
			assert.NotEmpty(t, script)

			// Check required strings are present
			for _, s := range tt.contains {
				assert.Contains(t, script, s, "script should contain: %s", s)
			}

			// Check excluded strings are NOT present
			for _, s := range tt.excludes {
				assert.NotContains(t, script, s, "script should NOT contain: %s", s)
			}

			// All scripts should start with shebang
			assert.True(t, strings.HasPrefix(script, "#!/bin/bash"))

			// All scripts should end with success message
			assert.Contains(t, script, "Network rules applied successfully")
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *multipass.NetworkConfig
		expectErr bool
		errMsg    string
	}{
		{
			name:      "nil_config_valid",
			config:    nil,
			expectErr: false,
		},
		{
			name:      "none_mode_valid",
			config:    &multipass.NetworkConfig{Mode: multipass.NetworkModeNone},
			expectErr: false,
		},
		{
			name:      "none_mode_with_rules_valid",
			config:    &multipass.NetworkConfig{Mode: multipass.NetworkModeNone, Rules: []multipass.NetworkRule{{Type: "ip", Value: "1.1.1.1"}}},
			expectErr: false,
		},
		{
			name:      "isolated_mode_valid",
			config:    &multipass.NetworkConfig{Mode: multipass.NetworkModeIsolated},
			expectErr: false,
		},
		{
			name:      "allowlist_requires_rules",
			config:    &multipass.NetworkConfig{Mode: multipass.NetworkModeAllowlist},
			expectErr: true,
			errMsg:    "requires at least one rule",
		},
		{
			name:      "blocklist_requires_rules",
			config:    &multipass.NetworkConfig{Mode: multipass.NetworkModeBlocklist},
			expectErr: true,
			errMsg:    "requires at least one rule",
		},
		{
			name: "allowlist_with_rules_valid",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{{Type: "ip", Value: "8.8.8.8"}},
			},
			expectErr: false,
		},
		{
			name: "blocklist_with_rules_valid",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeBlocklist,
				Rules: []multipass.NetworkRule{{Type: "ip", Value: "1.2.3.4"}},
			},
			expectErr: false,
		},
		{
			name:      "invalid_mode",
			config:    &multipass.NetworkConfig{Mode: "invalid"},
			expectErr: true,
			errMsg:    "invalid network mode",
		},
		{
			name:      "empty_mode_invalid",
			config:    &multipass.NetworkConfig{Mode: ""},
			expectErr: true,
			errMsg:    "invalid network mode",
		},
		{
			name: "invalid_ip",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{{Type: "ip", Value: "not.an.ip"}},
			},
			expectErr: true,
			errMsg:    "invalid IP address",
		},
		{
			name: "ip_with_letters",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{{Type: "ip", Value: "192.168.1.abc"}},
			},
			expectErr: true,
			errMsg:    "invalid IP address",
		},
		{
			name: "ip_too_few_parts",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{{Type: "ip", Value: "192.168.1"}},
			},
			expectErr: true,
			errMsg:    "invalid IP address",
		},
		{
			name: "ip_too_many_parts",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{{Type: "ip", Value: "192.168.1.1.1"}},
			},
			expectErr: true,
			errMsg:    "invalid IP address",
		},
		{
			name: "cidr_requires_slash",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeBlocklist,
				Rules: []multipass.NetworkRule{{Type: "cidr", Value: "10.0.0.0"}},
			},
			expectErr: true,
			errMsg:    "CIDR must contain /",
		},
		{
			name: "cidr_valid",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeBlocklist,
				Rules: []multipass.NetworkRule{{Type: "cidr", Value: "10.0.0.0/8"}},
			},
			expectErr: false,
		},
		{
			name: "domain_no_spaces",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{{Type: "domain", Value: "has space.com"}},
			},
			expectErr: true,
			errMsg:    "cannot contain spaces",
		},
		{
			name: "domain_valid",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{{Type: "domain", Value: "github.com"}},
			},
			expectErr: false,
		},
		{
			name: "invalid_rule_type",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{{Type: "invalid", Value: "test"}},
			},
			expectErr: true,
			errMsg:    "invalid rule type",
		},
		{
			name: "empty_rule_value",
			config: &multipass.NetworkConfig{
				Mode:  multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{{Type: "ip", Value: ""}},
			},
			expectErr: true,
			errMsg:    "rule value cannot be empty",
		},
		{
			name: "multiple_rules_one_invalid",
			config: &multipass.NetworkConfig{
				Mode: multipass.NetworkModeAllowlist,
				Rules: []multipass.NetworkRule{
					{Type: "ip", Value: "8.8.8.8"},
					{Type: "ip", Value: "invalid"},
				},
			},
			expectErr: true,
			errMsg:    "rule 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		ip    string
		valid bool
	}{
		{"192.168.1.1", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"1.2.3.4", true},
		{"10.0.0.1", true},
		{"127.0.0.1", true},

		// Invalid cases
		{"192.168.1", false},         // Too few parts
		{"192.168.1.1.1", false},     // Too many parts
		{"not.an.ip", false},         // Letters
		{"192.168.1.abc", false},     // Letters in last octet
		{"", false},                  // Empty
		{"192.168.1.", false},        // Trailing dot
		{".192.168.1.1", false},      // Leading dot
		{"192..168.1.1", false},      // Double dot
		{"192.168.1.1234", false},    // Part too long (>3 digits)
		{"-1.2.3.4", false},          // Negative
		{"1.2.3.4/24", false},        // CIDR notation
		{"192.168.1.1:80", false},    // Port notation
		{"192 .168.1.1", false},      // Space in IP
		{"a.b.c.d", false},           // All letters

		// Range validation (0-255)
		{"256.1.1.1", false},         // First octet > 255
		{"1.256.1.1", false},         // Second octet > 255
		{"1.1.256.1", false},         // Third octet > 255
		{"1.1.1.256", false},         // Fourth octet > 255
		{"999.999.999.999", false},   // All octets > 255

		// Leading zeros
		{"192.168.001.1", false},     // Leading zero in third octet
		{"01.02.03.04", false},       // Leading zeros throughout
		{"192.168.1.01", false},      // Leading zero in last octet
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := isValidIP(tt.ip)
			assert.Equal(t, tt.valid, result, "isValidIP(%q) should be %v", tt.ip, tt.valid)
		})
	}
}

func TestGenerateSystemdService(t *testing.T) {
	service := GenerateSystemdService()

	assert.Contains(t, service, "[Unit]")
	assert.Contains(t, service, "Description=Dabbi Network Rules")
	assert.Contains(t, service, "After=network.target")
	assert.Contains(t, service, "[Service]")
	assert.Contains(t, service, "Type=oneshot")
	assert.Contains(t, service, "/opt/dabbi/network/apply-rules.sh")
	assert.Contains(t, service, "[Install]")
	assert.Contains(t, service, "WantedBy=multi-user.target")
}

func TestGenerateIptablesScript_ValidBashSyntax(t *testing.T) {
	// Test that generated scripts have valid structure
	configs := []*multipass.NetworkConfig{
		nil,
		{Mode: multipass.NetworkModeNone},
		{Mode: multipass.NetworkModeIsolated},
		{Mode: multipass.NetworkModeAllowlist, Rules: []multipass.NetworkRule{{Type: "ip", Value: "8.8.8.8"}}},
		{Mode: multipass.NetworkModeBlocklist, Rules: []multipass.NetworkRule{{Type: "cidr", Value: "10.0.0.0/8"}}},
	}

	for _, config := range configs {
		script, err := GenerateIptablesScript(config)
		require.NoError(t, err)

		// Check basic bash structure
		assert.True(t, strings.HasPrefix(script, "#!/bin/bash"))
		assert.Contains(t, script, "set -e")

		// Check for balanced quotes (basic check)
		singleQuotes := strings.Count(script, "'")
		assert.Equal(t, 0, singleQuotes%2, "unbalanced single quotes")

		doubleQuotes := strings.Count(script, "\"")
		assert.Equal(t, 0, doubleQuotes%2, "unbalanced double quotes")
	}
}
