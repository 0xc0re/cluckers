package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/0xc0re/cluckers/internal/gateway"
)

// ListBotNames retrieves the supporter bot names via
// GET /launcher/v1/supporter/bot-names using the access token as a Bearer
// credential. The response is a slot-indexed array of names (empty string for
// an unset slot). Non-supporters receive an error.
func ListBotNames(ctx context.Context, client *gateway.Client, accessToken string) ([]string, error) {
	var names []string
	if err := client.Do(ctx, http.MethodGet, pathBotNames, accessToken, nil, &names); err != nil {
		return nil, err
	}
	return names, nil
}

// UpsertBotName sets the supporter bot name at the given 1-indexed slot via
// PUT /launcher/v1/supporter/bot-names/{slot} with a Bearer credential.
func UpsertBotName(ctx context.Context, client *gateway.Client, accessToken string, slot int, name string) error {
	path := fmt.Sprintf("%s/%d", pathBotNames, slot)
	req := gateway.BotNameUpsertRequest{BotName: name}
	return client.Do(ctx, http.MethodPut, path, accessToken, req, nil)
}

// DeleteBotName clears the supporter bot name at the given 1-indexed slot via
// DELETE /launcher/v1/supporter/bot-names/{slot} with a Bearer credential.
func DeleteBotName(ctx context.Context, client *gateway.Client, accessToken string, slot int) error {
	path := fmt.Sprintf("%s/%d", pathBotNames, slot)
	return client.Do(ctx, http.MethodDelete, path, accessToken, nil, nil)
}
