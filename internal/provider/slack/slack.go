package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SlackSearchProvider struct {
	DisplayName  string
	Enabled      bool
	token        string
	refreshToken string
	workspace    string
	dCookie      string
	userAgent    string

	cache      *cache
	httpClient *http.Client
}

func NewSlackProvider(httpClient *http.Client, name, userID, teamID, token, workspace, userAgent string) *SlackSearchProvider {
	cloned := *httpClient

	cloned.Transport = &rateLimitTransport{
		base: &authTransport{
			userAgent: userAgent,
			token:     token,
			base:      httpClient.Transport,
		},
		sleepFor:      60,
		retryAttempts: 2,
	}

	return &SlackSearchProvider{
		DisplayName: name,
		Enabled:     true,
		token:       token,
		workspace:   workspace,
		userAgent:   userAgent,
		cache: &cache{
			user: &User{
				UserID: userID,
				TeamID: teamID,
			},
		},
		httpClient: &cloned,
	}
}

func (sp *SlackSearchProvider) Initialize() error {
	if err := sp.loadUserInfo(); err != nil {
		return err
	}
	if err := sp.loadAccessibleChannels(); err != nil {
		return err
	}
	return nil
}

// Load a cache with the names and ids of all public and private Slack channels that the token can access.
// This cache is required for any future call requiring a channel id.
func (sp *SlackSearchProvider) loadAccessibleChannels() error {
	var cursor string

	getChannelUrl, err := url.JoinPath(slackBaseUrl, "api", "conversations.list")
	if err != nil {
		return err
	}

	for {
		req, err := http.NewRequest("GET", getChannelUrl, nil)
		if err != nil {
			return err
		}

		q := req.URL.Query()
		q.Add("limit", "20")
		q.Add("types", "public_channel,private_channel")

		if cursor != "" {
			q.Add("cursor", cursor)
		}

		req.URL.RawQuery = q.Encode()

		resp, err := sp.httpClient.Do(req)
		if err != nil {
			return err
		}

		var apiResp struct {
			OK               bool      `json:"ok"`
			Channels         []Channel `json:"channels"`
			Error            string    `json:"error"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return fmt.Errorf("decoding conversations.info response: %w", err)
		}
		if !apiResp.OK {
			fmt.Println(apiResp.Error)
			return fmt.Errorf("slack conversations.info not OK")
		}

		sp.cache.mu.Lock()
		if sp.cache.channels == nil {
			sp.cache.channels = apiResp.Channels
		} else {
			sp.cache.channels = append(sp.cache.channels, apiResp.Channels...)
		}
		sp.cache.mu.Unlock()

		if apiResp.ResponseMetadata.NextCursor != "" {
			cursor = apiResp.ResponseMetadata.NextCursor
		} else {
			break
		}
	}
	return nil
}

// enumerates all the domains you are allowed to automatically join a workspace from
// could get lucky and find an expired domain in the list you can buy and join with
func (sp *SlackSearchProvider) enumerateSigninDomains() ([]string, error) {
	getSigninDomainsReq, err := url.JoinPath(slackBaseUrl, "api", "team.info")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", getSigninDomainsReq, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		OK   string `json:"ok"`
		Team struct {
			EmailDomains string `json:"email_domain"`
		} `json:"team"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return strings.Split(apiResp.Team.EmailDomains, ","), nil
}

func (sp *SlackSearchProvider) loadUserInfo() error {
	getUserAPI, err := url.JoinPath(slackBaseUrl, "api", "users.info")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", getUserAPI, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("user", sp.cache.user.UserID)

	req.URL.RawQuery = q.Encode()

	resp, err := sp.httpClient.Do(req)
	if err != nil {
		return err
	}

	var apiResp struct {
		OK   bool `json:"ok"`
		User struct {
			ID       string `json:"id"`
			TeamID   string `json:"team_id"`
			RealName string `json:"real_name"`
			Name     string `json:"name"`
			IsAdmin  bool   `json:"is_admin"`
			IsOwner  bool   `json:"is_owner"`
			Profile  struct {
				Email string `json:"email"`
			} `json:"profile"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decoding users.info response: %w", err)
	}
	if !apiResp.OK {
		return fmt.Errorf("slack users.info not OK")
	}

	sp.cache.mu.Lock()
	sp.cache.user = &User{
		UserID:   apiResp.User.ID,
		TeamID:   apiResp.User.TeamID,
		FullName: apiResp.User.RealName,
		Username: apiResp.User.Name,
		Email:    apiResp.User.Profile.Email,
		IsAdmin:  apiResp.User.IsAdmin,
		IsOwner:  apiResp.User.IsOwner,
	}
	sp.cache.mu.Unlock()

	return nil
}
