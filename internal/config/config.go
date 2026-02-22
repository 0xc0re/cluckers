package config

import (
	"github.com/spf13/viper"
)

// Config holds all launcher configuration values.
type Config struct {
	Gateway  string `mapstructure:"gateway"`
	WinePath string `mapstructure:"wine_path"`
	GameDir  string `mapstructure:"game_dir"`
	HostX    string `mapstructure:"hostx"`
	Verbose  bool   `mapstructure:"verbose"`
}

// Load reads configuration from the TOML file (if it exists), applies defaults,
// and returns a populated Config. CLI flags bound to Viper override config values.
//
// Precedence: CLI flag > config file > default (handled natively by Viper).
func Load() (*Config, error) {
	// Defaults
	viper.SetDefault("gateway", "https://gateway-dev.project-crown.com")
	viper.SetDefault("hostx", "157.90.131.105")
	viper.SetDefault("verbose", false)
	viper.SetDefault("wine_path", "")
	viper.SetDefault("game_dir", "")

	// TOML config file
	viper.SetConfigName("settings")
	viper.SetConfigType("toml")
	viper.AddConfigPath(ConfigDir())

	// Read config file; ignore "not found" — config file is optional.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
