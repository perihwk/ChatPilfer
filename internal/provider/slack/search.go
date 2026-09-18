package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

var cloudQueries = [...]string{
	// AWS
	"s3.amazonaws.com",
	"elb.amazonaws.com",
	"rds.amazonaws.com",
	"amazon.com/s3/*",
	"*://s3*",
	"cloudfront.net",

	// Azure
	"core.windows.net",
	"azurewebsites.net",
	"vault.azure.net",
	"database.windows.net",
	// GCP
	"storage.googleapis.com",
	"storage.cloud.google.com",
	"gs://*",
	//Cloud storage
	"drive.google.com",
	"dropbox.com",
	"my.sharepoint.com",
	// misc
	"digitaloceanspaces.com",
	"cloudflarestorage.com",
	"backblazeb2.com",
}

type PinnedItem struct {
	CreatedBy string        `json:"created_by"`
	Channel   string        `json:"channel"`
	Message   PinnedMessage `json:"message"`
	Error     string        `json:"error"`
}

type PinnedMessage struct {
	Text      string `json:"text"`
	Files     []File `json:"files"`
	Permalink string `json:"permalink"`
}

type File struct {
	ID          string `json:"id"`
	Filetype    string `json:"filetype"`
	DownloadUrl string `json:"url_private_download"`
}

type searchMessagesParams struct {
	Query string // required
	Page  int    // optional; defaults to 0
	Count int    // optional; defaults to 100
}

type searchResult struct {
	TeamID    string
	ChannelID string
	UserID    string
	Timestamp string
	Text      string
	Permalink string
	Files     []SearchResultFile
}

type SearchResultFile struct {
	FileID          string `json:"id"`
	FileType        string `json:"mimetype"`
	FileSize        int    `json:"size"`
	FileName        string `json:"name"`
	FileDownloadUrl string `json:"url_private_download"`
}

// individual search request does not handle pagination
func (sp *SlackSearchProvider) searchMessages(params *searchMessagesParams) (int, []searchResult, error) {
	searchUrl, err := url.JoinPath(slackBaseUrl, "api", "search.messages")
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest("GET", searchUrl, nil)
	if err != nil {
		return 0, nil, err
	}

	q := req.URL.Query()

	if params.Query == "" {
		return 0, nil, fmt.Errorf("query param is required")
	} else {
		q.Add("query", params.Query)
	}

	q.Add("page", strconv.Itoa(params.Page))

	if params.Count > 0 {
		q.Add("count", strconv.Itoa(params.Count))
	}

	req.URL.RawQuery = q.Encode()

	resp, err := sp.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Ok       bool `json:"ok"`
		Messages struct {
			Total      int `json:"total"`
			Pagination struct {
				PageCount int `json:"page_count"`
			} `json:"pagination"`
			Matches []struct {
				TeamID  string `json:"team"`
				Channel struct {
					ID string `json:"id"`
				} `json:"channel"`
				UserID    string             `json:"user"`
				TS        string             `json:"ts"`
				Permalink string             `json:"permalink"`
				Text      string             `json:"text"`
				Files     []SearchResultFile `json:"files"`
			} `json:"matches"`
		} `json:"messages"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, nil, err
	}

	var result []searchResult
	for _, match := range apiResp.Messages.Matches {
		result = append(result, searchResult{
			TeamID:    match.TeamID,
			ChannelID: match.Channel.ID,
			UserID:    match.UserID,
			Timestamp: match.TS,
			Permalink: match.Permalink,
			Text:      match.Text,
			Files:     match.Files,
		})
	}

	return apiResp.Messages.Pagination.PageCount, result, nil
}

// searchAllMessages wraps searchMessages and exhausts all pages
func (sp *SlackSearchProvider) searchAllMessages(query string) ([]searchResult, error) {
	var allResults []searchResult
	page := 1

	for {
		pageCount, resp, err := sp.searchMessages(&searchMessagesParams{
			Query: query,
			Page:  page,
			Count: 100, // max per page, fewest round trips
		})
		if err != nil {
			return nil, fmt.Errorf("fetching page %d: %w", page, err)
		}

		allResults = append(allResults, resp...)

		if page >= pageCount {
			break
		}
		page++
	}

	return allResults, nil
}

func (sp *SlackSearchProvider) SearchCloudEndpoints() (map[string]searchResult, error) {
	// specifically searches for references to cloud providers
	cloudProviderPattern := regexp.MustCompile(`(?i)(` +
		// AWS
		`[a-zA-Z0-9\-\.\_]+\.s3[.-][a-z0-9\-]*\.amazonaws\.com` +
		`|s3://[a-zA-Z0-9\-\.\_]+` +
		`|s3\.[a-zA-Z0-9\-]+\.amazonaws\.com` +
		`|s3\.amazonaws\.com/[a-zA-Z0-9\-\.\_]+` +
		`|s3\.console\.aws\.amazon\.com/s3/buckets/[a-zA-Z0-9\-\.\_]+` +
		`|[a-zA-Z0-9\-]+\.execute-api\.[a-z0-9\-]+\.amazonaws\.com` +
		`|[a-zA-Z0-9\-]+\.cloudfront\.net` +
		`|[a-zA-Z0-9\-]+\.elb\.amazonaws\.com` +
		`|[a-zA-Z0-9\-]+\.rds\.amazonaws\.com` +
		// Azure
		`|[a-zA-Z0-9\-\.\_]+\.blob\.core\.windows\.net` +
		`|[a-zA-Z0-9\-\.\_]+\.file\.core\.windows\.net` +
		`|[a-zA-Z0-9\-\.\_]+\.dfs\.core\.windows\.net` +
		`|[a-zA-Z0-9\-\.\_]+\.queue\.core\.windows\.net` +
		`|[a-zA-Z0-9\-\.\_]+\.table\.core\.windows\.net` +
		`|[a-zA-Z0-9\-]+\.azurewebsites\.net` +
		`|[a-zA-Z0-9\-]+\.vault\.azure\.net` +
		`|[a-zA-Z0-9\-]+\.database\.windows\.net` +
		// GCP
		`|[a-zA-Z0-9\-_.]+\.storage\.googleapis\.com` +
		`|storage\.googleapis\.com/[a-zA-Z0-9\-_.]+` +
		`|storage\.cloud\.google\.com/[a-zA-Z0-9\-_./]+` +
		`|gs://[a-zA-Z0-9\-_.]+` +
		// Personal cloud storage
		`|drive\.google\.com/(?:file/d/|drive/folders/|open\?id=)[a-zA-Z0-9\-_]+` +
		`|[a-zA-Z0-9\-]+\.dropbox\.com/[a-zA-Z0-9\-_./?=&]+` +
		`|[a-zA-Z0-9\-]+-my\.sharepoint\.com/[a-zA-Z0-9\-_./:?=&]+` +
		// Other providers
		`|[a-zA-Z0-9\-]+\.digitaloceanspaces\.com` +
		`|[a-zA-Z0-9\-]+\.r2\.cloudflarestorage\.com` +
		`|[a-zA-Z0-9\-]+\.backblazeb2\.com` +
		`)`)

	cloudResults := make(map[string]searchResult)
	for _, query := range cloudQueries {
		cloudEndpoints, err := sp.searchAllMessages(query)
		if err != nil {
			continue
		}

		for _, result := range cloudEndpoints {
			fmt.Println(result.Text)
			matches := cloudProviderPattern.FindAllString(result.Text, -1)
			for _, match := range matches {
				cloudResults[match] = result
			}
		}
	}

	return cloudResults, nil
}

func (sp *SlackSearchProvider) GetWorkspaceUsers(limit int) ([]User, error) {
	getWorkspaceUsersUrl, err := url.JoinPath(slackBaseUrl, "api", "users.list")
	if err != nil {
		return nil, err
	}

	var result []User
	var cursor string

	for {
		req, err := http.NewRequest("GET", getWorkspaceUsersUrl, nil)
		if err != nil {
			return nil, err
		}

		q := req.URL.Query()
		if limit > 0 && limit < 900 {
			q.Add("limit", strconv.Itoa(limit))
		} else {
			q.Add("limit", "10")
		}

		if cursor != "" {
			q.Add("cursor", cursor)
		}

		req.URL.RawQuery = q.Encode()

		resp, err := sp.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		var apiResp struct {
			OK      bool `json:"ok"`
			Members []struct {
				ID       string `json:"id"`
				TeamID   string `json:"team_id"`
				RealName string `json:"real_name"`
				Name     string `json:"name"`
				IsAdmin  bool   `json:"is_admin"`
				IsOwner  bool   `json:"is_owner"`
				Has2FA   bool   `json:"has_2fa"`
				Profile  struct {
					Email string `json:"email"`
				} `json:"profile"`
			} `json:"members"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
			Error string `json:"error"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return nil, err
		}

		resp.Body.Close()

		for _, member := range apiResp.Members {
			result = append(result, User{
				userID:   member.ID,
				fullName: member.RealName,
				username: member.Name,
				teamID:   member.TeamID,
				email:    member.Profile.Email,
				isAdmin:  member.IsAdmin,
				isOwner:  member.IsOwner,
				has2FA:   member.Has2FA,
			})
		}

		cursor = apiResp.ResponseMetadata.NextCursor

		if cursor == "" {
			break
		}
	}
	return result, nil
}

func (sp *SlackSearchProvider) GetPinnedChats() ([]PinnedItem, error) {
	if sp.cache.channels == nil {
		return nil, fmt.Errorf("no channels in cache make sure initialize was ran")
	}

	sp.cache.mu.Lock()
	channelsCopy := sp.cache.channels
	sp.cache.mu.Unlock()

	getPinnedChatsUrl, err := url.JoinPath(slackBaseUrl, "api", "pins.list")
	if err != nil {
		return nil, err
	}

	var result []PinnedItem
	for _, channel := range channelsCopy {
		req, err := http.NewRequest("GET", getPinnedChatsUrl, nil)
		if err != nil {
			return nil, err
		}

		q := req.URL.Query()
		q.Add("channel", channel.ID)

		req.URL.RawQuery = q.Encode()

		resp, err := sp.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		var apiResp struct {
			OK    bool         `json:"ok"`
			Items []PinnedItem `json:"items"`
			Error string       `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return nil, err
		}

		resp.Body.Close()

		if !apiResp.OK {
			result = append(result, PinnedItem{
				Channel: channel.ID,
				Error:   apiResp.Error,
			})
			continue
		}

		if result == nil {
			result = apiResp.Items
		} else {
			result = append(result, apiResp.Items...)
		}
	}
	return result, nil
}
