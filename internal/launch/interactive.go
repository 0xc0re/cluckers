package launch

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// PrintLinkCode prints a Discord link code with instructions (CLI only).
func PrintLinkCode(code string) {
	fmt.Println()
	ui.Info("Your account must be linked to Discord before you can play.")
	ui.Info("DM the following code to the Project Crown bot on Discord:")
	fmt.Println()
	fmt.Printf("  Your link code: %s\n", code)
	fmt.Println()
	ui.Info("Bot DM: " + auth.DiscordBotDMURL)
	ui.Info("Server: " + auth.DiscordInviteURL)
}

// LoginInteractive performs a login and, if the account is not linked to
// Discord yet, walks the user through the link flow on the terminal: it prints
// the link code, then polls the gateway until the link completes and a real
// session is issued. Any other error (including the developer-only PIN gate)
// is returned as is. CLI only; the GUI has its own linking view.
func LoginInteractive(ctx context.Context, client *gateway.Client, username, password string) (*auth.LoginResult, error) {
	result, err := auth.Login(ctx, client, username, password)
	if err == nil {
		return result, nil
	}
	var nl *auth.NotLinkedError
	if !errors.As(err, &nl) {
		return nil, err
	}
	return WaitForLinkInteractive(ctx, client, username, password, nl.LinkCode)
}

// WaitForLinkInteractive prints firstCode and polls until the account is
// linked, re-printing the code if the server rotates it. CLI only.
func WaitForLinkInteractive(ctx context.Context, client *gateway.Client, username, password, firstCode string) (*auth.LoginResult, error) {
	const waiting = "Waiting for Discord linking (checking every 3 seconds)..."
	PrintLinkCode(firstCode)
	sp := ui.StartStep(waiting)
	result, err := auth.WaitForLink(ctx, client, username, password, func(code string) {
		if code == firstCode {
			return // Already printed.
		}
		sp.Stop()
		ui.Warn("The server issued a new link code.")
		PrintLinkCode(code)
		sp = ui.StartStep(waiting)
	})
	if err != nil {
		sp.Fail()
		return nil, err
	}
	sp.Success()
	ui.Success("Discord account linked!")
	return result, nil
}
