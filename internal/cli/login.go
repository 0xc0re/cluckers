package cli

import (
	"fmt"
	"time"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate and save credentials",
	Long:  "Logs in to the Project Crown gateway and refreshes cached tokens. Uses saved credentials if available, otherwise prompts for username and password.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := gateway.NewClient(Cfg.Gateway, Cfg.Verbose)

		// Try saved credentials first.
		creds, err := auth.LoadCredentials()
		if err != nil {
			ui.Verbose(fmt.Sprintf("Could not load saved credentials: %s", err), Cfg.Verbose)
		}

		var username, password string

		if creds != nil {
			username = creds.Username
			password = creds.Password
			ui.Info("Using saved credentials for " + username)
		} else {
			username, err = ui.PromptUsername()
			if err != nil {
				return err
			}
			password, err = ui.PromptPassword()
			if err != nil {
				return err
			}
		}

		// Authenticate with gateway.
		result, err := auth.Login(cmd.Context(), client, username, password)
		if err != nil {
			if creds != nil {
				ui.Warn("Saved credentials failed, please re-enter.")
				username, err = ui.PromptUsername()
				if err != nil {
					return err
				}
				password, err = ui.PromptPassword()
				if err != nil {
					return err
				}
				result, err = auth.Login(cmd.Context(), client, username, password)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// Save credentials.
		if err := auth.SaveCredentials(username, password); err != nil {
			ui.Warn(fmt.Sprintf("Could not save credentials: %s", err))
		}

		// Cache tokens.
		cache := &auth.TokenCache{
			Username:       result.Username,
			AccessToken:    result.AccessToken,
			AccessCachedAt: time.Now(),
		}

		// Also fetch OIDC token so cache is warm for next launch.
		oidcToken, err := auth.GetOIDCToken(cmd.Context(), client, result.Username, result.AccessToken)
		if err != nil {
			// Non-fatal -- OIDC will be fetched at launch time.
			ui.Verbose(fmt.Sprintf("Could not pre-fetch OIDC token: %s", err), Cfg.Verbose)
		} else {
			cache.OIDCToken = oidcToken
			cache.OIDCCachedAt = time.Now()
		}

		if err := auth.SaveTokenCache(cache); err != nil {
			ui.Warn(fmt.Sprintf("Could not save token cache: %s", err))
		}

		ui.Success("Logged in as " + result.Username)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
