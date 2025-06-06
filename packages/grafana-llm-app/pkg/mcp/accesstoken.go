package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/authlib/authn"
)

// tokenTimeout is the expiration time for the Grafana Live session token.
// Note: We tried setting this to 30 minutes, but it caused the following error:
// "failed to perform token exchange with auth api: invalid exchange response: invalid expiresIn: requested token expiration 30m0s exceeds maximum allowed expiration: 10m0s"
const tokenTimeout = time.Minute * 10

// accessTokenClient handles token exchange operations for Grafana Cloud authentication.
// It manages the exchange of access policy tokens for temporary access tokens that can
// be used to authenticate with Grafana services on behalf of users.
// It will only exchange tokens if the access policy token is not empty and if we're
// running in Grafana Cloud.
type accessTokenClient struct {
	// tokenExchangeClient is the client used to perform token exchanges with the auth API.
	tokenExchangeClient *authn.TokenExchangeClient
	// tenant is the Grafana Cloud tenant identifier.
	tenant string
	// isGrafanaCloud indicates whether this client is running in Grafana Cloud environment.
	isGrafanaCloud bool
}

// newAccessTokenClient creates a new access token client for handling token exchange operations.
// If accessPolicyToken is empty, the client will be created but token exchange will not be available.
// Returns an error if the token exchange client cannot be initialized.
func newAccessTokenClient(accessPolicyToken, tenant string, isGrafanaCloud bool) (*accessTokenClient, error) {
	acc := &accessTokenClient{isGrafanaCloud: isGrafanaCloud, tenant: tenant}
	if accessPolicyToken == "" {
		return acc, nil
	}
	var err error
	if acc.tokenExchangeClient, err = authn.NewTokenExchangeClient(authn.TokenExchangeConfig{
		Token:            accessPolicyToken,
		TokenExchangeURL: "http://api-lb.auth.svc.cluster.local./v1/sign-access-token", // TODO: make this configurable.
	}); err != nil {
		return nil, fmt.Errorf("create token exchange client: %w", err)
	}
	return acc, nil
}

// getAccessToken exchanges the access policy token for a temporary access token
// that can be used to authenticate with Grafana services. Returns an empty string
// if not running in Grafana Cloud or if no token exchange client is configured.
func (a *accessTokenClient) getAccessToken(ctx context.Context) (string, error) {
	if !a.isGrafanaCloud {
		return "", nil
	}
	tokenTimeoutSeconds := int(tokenTimeout.Seconds())
	t, err := a.tokenExchangeClient.Exchange(ctx, authn.TokenExchangeRequest{
		Namespace: fmt.Sprintf("stack-%s", a.tenant),
		Audiences: []string{"grafana"},
		ExpiresIn: &tokenTimeoutSeconds,
	})
	if err != nil {
		return "", fmt.Errorf("perform token exchange with auth api: %w", err)
	}
	return t.Token, nil
}
