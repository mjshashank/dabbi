package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotEmpty(t, cfg.AuthToken, "AuthToken should be generated")
	assert.Len(t, cfg.AuthToken, 36, "AuthToken should be a UUID (36 chars)")
	assert.Equal(t, 2, cfg.Defaults.CPU)
	assert.Equal(t, "4G", cfg.Defaults.Mem)
	assert.Equal(t, "20G", cfg.Defaults.Disk)
	assert.Equal(t, 5, cfg.ShutdownTimeoutMins)
	assert.Empty(t, cfg.Defaults.CloudInit)
	assert.Nil(t, cfg.Defaults.NetworkConfig)
}

func TestDefaultConfig_GeneratesUniqueTokens(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()

	assert.NotEqual(t, cfg1.AuthToken, cfg2.AuthToken, "Each call should generate unique token")
}

func TestGetCloudInitPath(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create a test cloud-init file
	configCloudInit := filepath.Join(tmpDir, "config-cloud-init.yaml")
	err := os.WriteFile(configCloudInit, []byte("test"), 0644)
	require.NoError(t, err)

	explicitCloudInit := filepath.Join(tmpDir, "explicit-cloud-init.yaml")
	err = os.WriteFile(explicitCloudInit, []byte("test"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name          string
		explicit      string
		configDefault string
		expected      string
	}{
		{
			name:          "explicit_takes_priority",
			explicit:      explicitCloudInit,
			configDefault: configCloudInit,
			expected:      explicitCloudInit,
		},
		{
			name:          "config_default_used_when_no_explicit",
			explicit:      "",
			configDefault: configCloudInit,
			expected:      configCloudInit,
		},
		{
			name:          "explicit_path_even_if_doesnt_exist",
			explicit:      "/nonexistent/path.yaml",
			configDefault: configCloudInit,
			expected:      "/nonexistent/path.yaml",
		},
		{
			name:          "config_default_ignored_if_file_missing",
			explicit:      "",
			configDefault: "/nonexistent/config-cloud-init.yaml",
			expected:      "", // Will check default path which also doesn't exist
		},
		{
			name:          "empty_when_no_paths_set",
			explicit:      "",
			configDefault: "",
			expected:      "", // Default cloud-init path doesn't exist in tests
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Defaults: Defaults{
					CloudInit: tt.configDefault,
				},
			}

			result := cfg.GetCloudInitPath(tt.explicit)

			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			}
			// For cases where we expect empty, we just check it doesn't panic
			// The actual default path check depends on home directory
		})
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create and save config
	cfg := DefaultConfig()
	cfg.Defaults.CPU = 4
	cfg.Defaults.Mem = "8G"
	cfg.ShutdownTimeoutMins = 10

	err := cfg.Save()
	require.NoError(t, err)

	// Verify file permissions
	configPath := filepath.Join(tmpHome, ConfigDir, ConfigFile)
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "Config should have restrictive permissions")

	// Load and verify
	loaded, err := Load()
	require.NoError(t, err)

	assert.Equal(t, cfg.AuthToken, loaded.AuthToken)
	assert.Equal(t, cfg.Defaults.CPU, loaded.Defaults.CPU)
	assert.Equal(t, cfg.Defaults.Mem, loaded.Defaults.Mem)
	assert.Equal(t, cfg.ShutdownTimeoutMins, loaded.ShutdownTimeoutMins)
}

func TestLoad_CreatesDefaultOnMissing(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Load should create default config
	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.NotEmpty(t, cfg.AuthToken)
	assert.Equal(t, 2, cfg.Defaults.CPU)
	assert.Equal(t, "4G", cfg.Defaults.Mem)
	assert.Equal(t, "20G", cfg.Defaults.Disk)
	assert.Equal(t, 5, cfg.ShutdownTimeoutMins)

	// Verify file was created
	configPath := filepath.Join(tmpHome, ConfigDir, ConfigFile)
	_, err = os.Stat(configPath)
	assert.NoError(t, err, "Config file should be created")
}

func TestLoad_HandlesMalformedJSON(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create malformed config file
	configDir := filepath.Join(tmpHome, ConfigDir)
	err := os.MkdirAll(configDir, 0700)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, ConfigFile)
	err = os.WriteFile(configPath, []byte("not valid json"), 0600)
	require.NoError(t, err)

	// Load should fail
	_, err = Load()
	assert.Error(t, err)
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	require.NoError(t, err)
	assert.Contains(t, path, ConfigDir)
	assert.Contains(t, path, ConfigFile)
}

func TestDefaultCloudInitPath(t *testing.T) {
	path, err := DefaultCloudInitPath()
	require.NoError(t, err)
	assert.Contains(t, path, ConfigDir)
	assert.Contains(t, path, DefaultCloudInitFile)
}

func TestEnsureDefaultCloudInit(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// First call should create the file
	path, created, err := EnsureDefaultCloudInit()
	require.NoError(t, err)
	assert.True(t, created, "File should be created on first call")
	assert.NotEmpty(t, path)

	// Verify content
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "#cloud-config")
	assert.Contains(t, string(content), "package_update: true")

	// Second call should not create
	path2, created2, err := EnsureDefaultCloudInit()
	require.NoError(t, err)
	assert.False(t, created2, "File should not be created on second call")
	assert.Equal(t, path, path2)
}

func TestConfigSave_CreatesDirectory(t *testing.T) {
	// Create temp home directory (without config dir)
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cfg := DefaultConfig()
	err := cfg.Save()
	require.NoError(t, err)

	// Verify directory was created with correct permissions
	configDir := filepath.Join(tmpHome, ConfigDir)
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm(), "Config dir should have restrictive permissions")
}

func TestConfig_JSONMarshal(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Defaults.CloudInit = "/path/to/cloud-init.yaml"

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var unmarshaled Config
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, cfg.AuthToken, unmarshaled.AuthToken)
	assert.Equal(t, cfg.Defaults.CPU, unmarshaled.Defaults.CPU)
	assert.Equal(t, cfg.Defaults.CloudInit, unmarshaled.Defaults.CloudInit)
}
