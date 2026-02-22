package cli

import (
	"github.com/cstory/cluckers/internal/config"
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
	Short: "Project Crown Linux Launcher",
	Long:  "Cluckers Central — a native Linux launcher for Realm Royale on the Project Crown private server.",
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
	rootCmd.PersistentFlags().String("gateway", "https://gateway-dev.project-crown.com", "Gateway API base URL")

	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("gateway", rootCmd.PersistentFlags().Lookup("gateway"))
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
