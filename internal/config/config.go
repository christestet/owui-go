package config

import (
	"os"
	"path/filepath"
	"time"

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
	Version         string    `mapstructure:"version" json:"version"`
	Checksum        string    `mapstructure:"checksum" json:"checksum"`
	LastUpdateCheck time.Time `mapstructure:"last_update_check" json:"last_update_check"`
}

type InstanceConfig struct {
	URL     string    `mapstructure:"url" json:"url"`
	APIKey  string    `mapstructure:"api_key" json:"api_key"`
	AddedAt time.Time `mapstructure:"added_at" json:"added_at"`
}

type SettingsConfig struct {
	OutputFormat   string `mapstructure:"output_format" json:"output_format"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds" json:"timeout_seconds"`
}

// ConfigPath resolves the configuration file path
func ConfigPath() (string, error) {
	base, err := os.UserConfigDir()
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

	viper.SetConfigFile(path)
	viper.SetConfigType("json")

	// Set defaults
	viper.SetDefault("settings.output_format", "console")
	viper.SetDefault("settings.timeout_seconds", 30)
	viper.SetDefault("instances", map[string]InstanceConfig{})

	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			// Write default config if not exists
			if err := viper.SafeWriteConfig(); err != nil {
				// We can ignore error here as it will use defaults anyway
			}
		} else {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the current viper configuration to file
func Save() error {
	return viper.WriteConfig()
}
