package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/perihwk/chatpilfer/internal/provider"
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
	// shallow copy so each Slack provider still has the cookie jar, proxy server, etc
	cloned := *httpClient

	// extend the base httpClient's transport to include the token, user agent, and rate limit settings
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
	if err := sp.loadChannels(); err != nil {
		return err
	}
	return nil
}

func (sp *SlackSearchProvider) Pilfer(outputFolder string, queries map[string]*provider.QueryGroups) error {
	// get pinned chats
	items, err := sp.GetPinnedChats()
	if err != nil {
		return err
	}
	pinnedChatFile, err := os.OpenFile(path.Join(outputFolder, "pinned_messages.json"), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	if err := provider.WriteOutput(items, pinnedChatFile); err != nil {
		return err
	}
	pinnedChatFile.Close()

	// sign in domains
	domains, err := sp.EnumerateSigninDomains()
	if err != nil {
		return err
	}
	signinDomainsFile, err := os.OpenFile(path.Join(outputFolder, "auth_domains.json"), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	if err := provider.WriteOutput(domains, signinDomainsFile); err != nil {
		return err
	}
	signinDomainsFile.Close()

	// search for credentials
	for _, queryGroup := range queries {
		fileCache, err := os.OpenFile(path.Join(outputFolder, queryGroup.OutputFile), os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer fileCache.Close()

		var queryResults map[string][]any
		if queryGroup.SearchType == "messages" {
			queryResults, err = sp.SearchMessages(queryGroup.Queries)
			if err != nil {
				continue
			}
		} else if queryGroup.SearchType == "files" {
			queryResults, err = sp.SearchFiles(queryGroup.Queries)
			if err != nil {
				continue
			}
		}
		if err := provider.WriteOutput(queryResults, fileCache); err != nil {
			return err
		}
		fileCache.Close()
	}
	return nil
}

func (sp *SlackSearchProvider) GetChannels() ([]Channel, error) {
	var cursor string

	getChannelUrl, err := url.JoinPath(slackBaseUrl, "api", "conversations.list")
	if err != nil {
		return nil, err
	}

	var channels []Channel = make([]Channel, 0)
	for {
		req, err := http.NewRequest("GET", getChannelUrl, nil)
		if err != nil {
			return nil, err
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
			return nil, err
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
			return nil, fmt.Errorf("decoding conversations.info response: %w", err)
		}
		resp.Body.Close()

		if !apiResp.OK {
			fmt.Println(apiResp.Error)
			return nil, fmt.Errorf("slack conversations.info not OK")
		}

		channels = append(channels, apiResp.Channels...)

		if apiResp.ResponseMetadata.NextCursor != "" {
			cursor = apiResp.ResponseMetadata.NextCursor
		} else {
			break
		}
	}
	return channels, nil
}

func (sp *SlackSearchProvider) GetUserInfo() (*User, error) {
	getUserAPI, err := url.JoinPath(slackBaseUrl, "api", "users.info")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", getUserAPI, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("user", sp.cache.user.UserID)

	req.URL.RawQuery = q.Encode()

	resp, err := sp.httpClient.Do(req)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("decoding users.info response: %w", err)
	}
	if !apiResp.OK {
		return nil, fmt.Errorf("slack users.info not OK")
	}

	return &User{
		UserID:   apiResp.User.ID,
		TeamID:   apiResp.User.TeamID,
		FullName: apiResp.User.RealName,
		Username: apiResp.User.Name,
		Email:    apiResp.User.Profile.Email,
		IsAdmin:  apiResp.User.IsAdmin,
		IsOwner:  apiResp.User.IsOwner,
	}, nil

}

// enumerates all the domains you are allowed to automatically join a workspace from
// could get lucky and find an expired domain in the list you can buy and join with
func (sp *SlackSearchProvider) EnumerateSigninDomains() ([]string, error) {
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
		OK   bool `json:"ok"`
		Team struct {
			EmailDomains string `json:"email_domain"`
		} `json:"team"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return strings.Split(apiResp.Team.EmailDomains, ","), nil
}

// Load a cache with the names and ids of all public and private Slack channels that the token can access.
// This cache is required for any future call requiring a channel id.
func (sp *SlackSearchProvider) loadChannels() error {
	channels, err := sp.GetChannels()
	if err != nil {
		return err
	}
	sp.cache.mu.Lock()
	sp.cache.channels = channels
	sp.cache.mu.Unlock()

	return nil
}

func (sp *SlackSearchProvider) loadUserInfo() error {
	user, err := sp.GetUserInfo()
	if err != nil {
		return err
	}

	sp.cache.mu.Lock()
	sp.cache.user = user
	sp.cache.mu.Unlock()

	return nil
}
