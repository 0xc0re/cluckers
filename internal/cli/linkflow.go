package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// printLinkCode prints a Discord link code with instructions.
func printLinkCode(code string) {
	fmt.Println()
	ui.Info("Your account must be linked to Discord before you can play.")
	ui.Info("DM the following code to the Project Crown bot on Discord (" + auth.DiscordInviteURL + "):")
	fmt.Println()
	fmt.Printf("  Your link code: %s\n", code)
	fmt.Println()
}

// completeLogin performs a login and, if the account is not linked to Discord
// yet, walks the user through the link flow: it prints the link code, then
// polls the gateway until the link completes and a real session is issued.
// Any other error (including the developer-only PIN gate) is returned as is.
func completeLogin(ctx context.Context, client *gateway.Client, username, password string) (*auth.LoginResult, error) {
	result, err := auth.Login(ctx, client, username, password)
	if err == nil {
		return result, nil
	}
	var nl *auth.NotLinkedError
	if !errors.As(err, &nl) {
		return nil, err
	}

	printLinkCode(nl.LinkCode)
	sp := ui.StartStep("Waiting for Discord linking (checking every 3 seconds)...")
	result, err = auth.WaitForLink(ctx, client, username, password, func(code string) {
		if code == nl.LinkCode {
			return // Already printed.
		}
		sp.Stop()
		ui.Warn("The server issued a new link code.")
		printLinkCode(code)
		sp = ui.StartStep("Waiting for Discord linking (checking every 3 seconds)...")
	})
	if err != nil {
		sp.Fail()
		return nil, err
	}
	sp.Success()
	ui.Success("Discord account linked!")
	return result, nil
}
