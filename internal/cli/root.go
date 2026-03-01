package cli

import (
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Cfg holds the loaded configuration, available after PersistentPreRunE.
	Cfg *config.Config

	versionStr string
)

var rootCmd = &cobra.Command{
	Use:   "cluckers",
	Short: "Project Crown Launcher",
	Long:  "Cluckers Central — a native launcher for Realm Royale on the Project Crown private server.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		Cfg = cfg
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output (debug details, API responses, timing)")
	rootCmd.PersistentFlags().String("gateway", config.DefaultGateway(), "Gateway API base URL")

	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("gateway", rootCmd.PersistentFlags().Lookup("gateway"))
}

// InitFlags re-applies flag defaults that depend on build-time values.
// Called from main() after config.SetBuildDefaults() so that --help
// reflects the injected gateway URL rather than the compiled-in fallback.
func InitFlags() {
	rootCmd.PersistentFlags().Lookup("gateway").DefValue = config.DefaultGateway()
}

// SetVersion sets the version string displayed by --version.
func SetVersion(v string) {
	versionStr = v
	rootCmd.Version = v
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
