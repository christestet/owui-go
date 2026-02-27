package config

import (
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

	viper.SetConfigFile(path)
	viper.SetConfigType("json")

	// Set defaults
	viper.SetDefault("settings.output_format", "pretty")
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
	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the Config to the configuration file using viper
func (c *Config) Save() error {
	viper.Set("cli", c.Cli)
	viper.Set("active_instance", c.ActiveInstance)
	viper.Set("instances", c.Instances)
	viper.Set("settings", c.Settings)
	return viper.WriteConfig()
}
