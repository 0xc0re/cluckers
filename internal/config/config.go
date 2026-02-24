package config

import (
	"github.com/spf13/viper"
)

// Build-time defaults set via SetBuildDefaults(). Initialized to hardcoded
// fallback values so that `go build` without ldflags still works.
var (
	defaultGateway = "https://gateway-dev.project-crown.com"
	defaultHostX   = "157.90.131.105"
)

// DefaultGateway returns the build-time gateway URL (used by CLI flag defaults).
func DefaultGateway() string {
	return defaultGateway
}

// DefaultHostX returns the build-time hostx IP (used by CLI flag defaults).
func DefaultHostX() string {
	return defaultHostX
}

// SetBuildDefaults is called from main() to inject build-time ldflags values
// into the config package. Only non-empty values are applied.
func SetBuildDefaults(gateway, hostx string) {
	if gateway != "" {
		defaultGateway = gateway
	}
	if hostx != "" {
		defaultHostX = hostx
	}
}

// Config holds all launcher configuration values.
type Config struct {
	Gateway    string `mapstructure:"gateway"`
	WinePath   string `mapstructure:"wine_path"`
	WinePrefix string `mapstructure:"wine_prefix"`
	GameDir    string `mapstructure:"game_dir"`
	HostX      string `mapstructure:"hostx"`
	Verbose    bool   `mapstructure:"verbose"`
}

// Load reads configuration from the TOML file (if it exists), applies defaults,
// and returns a populated Config. CLI flags bound to Viper override config values.
//
// Precedence: CLI flag > config file > default (handled natively by Viper).
func Load() (*Config, error) {
	// Defaults (sourced from build-time ldflags via SetBuildDefaults)
	viper.SetDefault("gateway", defaultGateway)
	viper.SetDefault("hostx", defaultHostX)
	viper.SetDefault("verbose", false)
	viper.SetDefault("wine_path", "")
	viper.SetDefault("wine_prefix", "")
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
