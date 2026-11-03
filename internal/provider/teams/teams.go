package teams

import (
	"net/http"

	"github.com/perihwk/chatpilfer/internal/provider"
)

type TeamsSearchProvider struct {
	DisplayName string
	Enabled     bool
	token       string
	workspace   string
	userAgent   string

	httpClient *http.Client
}

func NewTeamsProvider(httpClient *http.Client, token, userAgent string) *TeamsSearchProvider {
	// shallow copy so each Slack provider still has the cookie jar, proxy server, etc
	cloned := *httpClient

	// extend the base httpClient's transport to include the token, user agent, and rate limit settings
	cloned.Transport = &authTransport{
		userAgent: userAgent,
		token:     token,
		base:      httpClient.Transport,
	}

	return &TeamsSearchProvider{
		DisplayName: "Teams",
		Enabled:     true,
		token:       token,
		workspace:   "",
		userAgent:   userAgent,
		httpClient:  &cloned,
	}
}

func (sp *TeamsSearchProvider) Initialize() error {
	return nil
}

func (sp *TeamsSearchProvider) Pilfer(outputDir string, queries map[string]*provider.QueryGroups) error {
	return nil
}
