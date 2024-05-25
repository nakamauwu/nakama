package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	"golang.org/x/sync/errgroup"
)

type GitHubProvider struct {
	*oauth2.Config
}

func NewGitHubProvider(clientID, clientSecret, redirectURL string) *GitHubProvider {
	return &GitHubProvider{
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     endpoints.GitHub,
			Scopes:       []string{"read:user", "user:email"},
		},
	}
}

func (p *GitHubProvider) Name() string {
	return "github"
}

func (p *GitHubProvider) Claims(ctx context.Context, token *oauth2.Token) (Claims, error) {
	var out Claims

	client := p.Config.Client(ctx, token)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		out.Email, out.EmailVerified, err = p.email(ctx, client)
		return err
	})
	g.Go(func() error {
		var err error
		out.Picture, err = p.avatarURL(ctx, client)
		return err
	})

	if err := g.Wait(); err != nil {
		return out, err
	}

	return out, nil
}

func (p *GitHubProvider) email(ctx context.Context, client *http.Client) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", false, fmt.Errorf("create github emails request: %w", err)
	}

	req.Header.Set("User-Agent", "Nakama Server")
	resp, err := client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("do github emails request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("github emails request failed: %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", false, fmt.Errorf("decode github emails response: %w", err)
	}

	for _, email := range emails {
		if email.Primary {
			return email.Email, email.Verified, nil
		}
	}

	return "", false, errors.New("no primary email found")
}

func (p *GitHubProvider) avatarURL(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", fmt.Errorf("create github user request: %w", err)
	}

	req.Header.Set("User-Agent", "Nakama Server")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("do github user request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github user request failed: %d", resp.StatusCode)
	}

	var user struct {
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("decode github user response: %w", err)
	}

	return user.AvatarURL, nil
}
