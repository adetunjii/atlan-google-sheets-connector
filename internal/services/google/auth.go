package google

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/adetunjii/google-sheets-connector/pkg/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	ErrAuthenticationFailed   = errors.New("user authentication faield")
	ErrFailedSheetSvcCreation = errors.New("failed to create new google sheets service")
	ErrInvalidAuthGrantCode   = errors.New("invalid auth grant code to provision accessToken")
	ErrUserDeniedPermission   = errors.New("user denied permission")
)

type GoogleClient struct {
	logger logger.AppLogger
	config *oauth2.Config
}

func NewGoogleClient(client_id string, client_secret string, scopes []string, redirect_url string, logger logger.AppLogger) *GoogleClient {

	oauth2Conf := &oauth2.Config{
		ClientID:     client_id,
		ClientSecret: client_secret,
		Scopes:       scopes,
		RedirectURL:  redirect_url,
		Endpoint:     google.Endpoint,
	}

	return &GoogleClient{config: oauth2Conf, logger: logger}
}

func (g *GoogleClient) ExchangeCode(code string) (*oauth2.Token, error) {

	// validate auth code
	if code == "" {
		return nil, errors.New("code not found to provide an access token")
	}

	// exchange auth code for token details
	token, err := g.config.Exchange(context.Background(), code)
	if err != nil {
		g.logger.Error("google code exchange failed with: ", err)
		return nil, errors.New("google code exchange failed")
	}

	return token, nil
}

func (g *GoogleClient) HandleGoogleLogin() (string, error) {

	URL, err := url.Parse(g.config.Endpoint.AuthURL)
	if err != nil {
		return "", err
	}

	parameters := url.Values{}
	parameters.Add("client_id", g.config.ClientID)
	parameters.Add("scope", strings.Join(g.config.Scopes, " "))
	parameters.Add("redirect_uri", g.config.RedirectURL)
	parameters.Add("response_type", "code")
	// parameters.Add("state", oauthStateString)
	URL.RawQuery = parameters.Encode()
	url := URL.String()
	return url, nil
}

func (g *GoogleClient) HandleGoogleCallback(code string, errorReason string) (*oauth2.Token, error) {

	if code == "" {
		return nil, ErrInvalidAuthGrantCode
	}

	if errorReason != "" {
		if errorReason == "user_denied" {
			return nil, ErrUserDeniedPermission
		}
		return nil, fmt.Errorf("failed to authenticate user due to %s", errorReason)
	}

	token, err := g.ExchangeCode(code)
	if err != nil {
		return nil, err
	}

	return token, nil
}
