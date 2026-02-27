package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the CLI configuration map
type Config struct {
	Cli            CliConfig                 `mapstructure:"cli" json:"cli"`
	ActiveInstance string                    `mapstructure:"active_instance" json:"active_instance"`
	Instances      map[string]InstanceConfig `mapstructure:"instances" json:"instances"`
	Settings       SettingsConfig            `mapstructure:"settings" json:"settings"`
}

type CliConfig struct {
	Version              string `mapstructure:"version" json:"version"`
	Checksum             string `mapstructure:"checksum" json:"checksum"`
	LastUpdateCheck      string `mapstructure:"last_update_check" json:"last_update_check"`
	CompletionsInstalled bool   `mapstructure:"completions_installed" json:"completions_installed"`
}

type InstanceConfig struct {
	URL     string `mapstructure:"url" json:"url"`
	APIKey  string `mapstructure:"api_key" json:"api_key"`
	AddedAt string `mapstructure:"added_at" json:"added_at"`
}

// RedactedAPIKey returns a safe-to-print version of the API Key.
// It shows up to the first 6 characters and masks the rest.
func (c InstanceConfig) RedactedAPIKey() string {
	if len(c.APIKey) <= 6 {
		return "******"
	}
	return c.APIKey[:6] + "******"
}

type SettingsConfig struct {
	OutputFormat   string `mapstructure:"output_format" json:"output_format"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds" json:"timeout_seconds"`
}

// UserConfigDirFunc holds the pathing function for configuration mapping
var UserConfigDirFunc = os.UserConfigDir

// ConfigPath resolves the configuration file path
func ConfigPath() (string, error) {
	base, err := UserConfigDirFunc()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "owui")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the configuration using viper
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// Guard against empty/truncated files left by interrupted writes
	if info, statErr := os.Stat(path); statErr == nil && info.Size() == 0 {
		cfg := defaultConfig()
		_ = cfg.Save() // best-effort: reset file to defaults
		return cfg, nil
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")
	v.SetDefault("settings.output_format", "pretty")
	v.SetDefault("settings.timeout_seconds", 30)
	v.SetDefault("instances", map[string]InstanceConfig{})

	if err := v.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			cfg := defaultConfig()
			_ = cfg.Save()
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Settings:  SettingsConfig{OutputFormat: "pretty", TimeoutSeconds: 30},
		Instances: map[string]InstanceConfig{},
	}
}

// Save writes the Config to the configuration file using viper
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigType("json")
	v.Set("cli", c.Cli)
	v.Set("active_instance", c.ActiveInstance)
	v.Set("instances", c.Instances)
	v.Set("settings", c.Settings)

	// Atomic write: write to a .tmp.json file then rename (POSIX atomic).
	// Must share the same directory as path so rename is within one filesystem.
	// Must end in .json so Viper uses the JSON encoder.
	tmpPath := filepath.Join(filepath.Dir(path), "config.tmp.json")
	if err := v.WriteConfigAs(tmpPath); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return os.Rename(tmpPath, path)
}
