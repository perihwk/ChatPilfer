package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

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
	TeamID    string         `json:"team_id"`
	ChannelID string         `json:"channel_id"`
	UserID    string         `json:"user_id"`
	Timestamp string         `json:"ts"`
	Text      string         `json:"text"`
	Permalink string         `json:"permalink"`
	Files     []AttachedFile `json:"files"`
}

type AttachedFile struct {
	FileID          string `json:"id"`
	FileType        string `json:"mimetype"`
	FileSize        int    `json:"size"`
	FileName        string `json:"name"`
	FileDownloadUrl string `json:"url_private_download"`
}

type fileSearchResult struct {
	FileID      string   `json:"id"`
	UserID      string   `json:"user"`
	TeamID      string   `json:"user_team"`
	Mimetype    string   `json:"mimetype"`
	FileName    string   `json:"name"`
	DownloadUrl string   `json:"url_private_download"`
	Size        int      `json:"size"`
	Channels    []string `json:"channels"`
	Timestamp   string   `json:"ts"`
	Permalink   string   `json:"permalink"`
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
				UserID    string         `json:"user"`
				TS        string         `json:"ts"`
				Permalink string         `json:"permalink"`
				Text      string         `json:"text"`
				Files     []AttachedFile `json:"files"`
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

// individual search request does not handle pagination
func (sp *SlackSearchProvider) searchFiles(params *searchMessagesParams) (int, []fileSearchResult, error) {
	searchUrl, err := url.JoinPath(slackBaseUrl, "api", "search.files")
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
		Ok    bool `json:"ok"`
		Files struct {
			Total      int `json:"total"`
			Pagination struct {
				PageCount int `json:"page_count"`
			} `json:"pagination"`
			Matches []struct {
				FileID      string   `json:"id"`
				TeamID      string   `json:"user_team"`
				Mimetype    string   `json:"mimetype"`
				FileName    string   `json:"name"`
				DownloadUrl string   `json:"url_private_download"`
				Size        int      `json:"size"`
				Channels    []string `json:"channels"`
				UserID      string   `json:"user"`
				TS          string   `json:"ts"`
				Permalink   string   `json:"permalink"`
			} `json:"matches"`
		} `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, nil, err
	}

	var result []fileSearchResult
	for _, match := range apiResp.Files.Matches {
		result = append(result, fileSearchResult{
			FileID:      match.FileID,
			TeamID:      match.TeamID,
			Channels:    match.Channels,
			UserID:      match.UserID,
			Timestamp:   match.TS,
			Permalink:   match.Permalink,
			Mimetype:    match.Mimetype,
			FileName:    match.FileName,
			DownloadUrl: match.DownloadUrl,
			Size:        match.Size,
		})
	}

	return apiResp.Files.Pagination.PageCount, result, nil
}

// searchAllMessages wraps searchMessages and exhausts all pages
func (sp *SlackSearchProvider) searchAllFiles(query string) ([]fileSearchResult, error) {
	var allResults []fileSearchResult
	page := 1

	for {
		pageCount, resp, err := sp.searchFiles(&searchMessagesParams{
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

// perform search and return the searchResult. Do not do any regex to extract interesting information
func (sp *SlackSearchProvider) SearchMessages(queries []string) (map[string][]any, error) {
	// track all results so we don't add duplicate messages if there are multiple query matches
	type resultKey struct {
		timestamp string
		permalink string
	}
	seen := make(map[resultKey]bool)

	searchResults := make(map[string][]any)
	for _, query := range queries {
		results, err := sp.searchAllMessages(query)
		if err != nil {
			continue
		}

		for _, result := range results {
			key := resultKey{
				timestamp: result.Timestamp,
				permalink: result.Permalink,
			}

			if seen[key] {
				continue
			}

			seen[key] = true
			searchResults[query] = append(searchResults[query], result)
		}
	}

	return searchResults, nil
}

func (sp *SlackSearchProvider) SearchFiles(queries []string) (map[string][]any, error) {
	// track all results so we don't add duplicate messages if there are multiple query matches
	type resultKey struct {
		timestamp string
		permalink string
	}
	seen := make(map[resultKey]bool)

	searchResults := make(map[string][]any)
	for _, query := range queries {
		results, err := sp.searchAllFiles(query)
		if err != nil {
			continue
		}

		for _, result := range results {
			key := resultKey{
				timestamp: result.Timestamp,
				permalink: result.Permalink,
			}

			if seen[key] {
				continue
			}

			seen[key] = true
			searchResults[query] = append(searchResults[query], result)
		}
	}

	return searchResults, nil
}

// search messages then attempt to extract the useful information via regex
func (sp *SlackSearchProvider) SearchMessagesRegex(queries []string, regex string) (map[string][]any, error) {
	regexPattern := regexp.MustCompile(regex)

	// track all results so we don't add duplicate messages if there are multiple query matches
	type resultKey struct {
		match     string
		permalink string
	}
	seen := make(map[resultKey]bool)

	searchResults := make(map[string][]any)
	for _, query := range queries {
		results, err := sp.searchAllMessages(query)
		if err != nil {
			continue
		}

		for _, result := range results {
			matches := regexPattern.FindAllString(result.Text, -1)
			for _, match := range matches {
				key := resultKey{
					match:     match,
					permalink: result.Permalink,
				}
				if seen[key] {
					continue
				}
				seen[key] = true
				searchResults[match] = append(searchResults[match], result)
			}
		}
	}

	return searchResults, nil
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
				UserID:   member.ID,
				FullName: member.RealName,
				Username: member.Name,
				TeamID:   member.TeamID,
				Email:    member.Profile.Email,
				IsAdmin:  member.IsAdmin,
				IsOwner:  member.IsOwner,
				Has2FA:   member.Has2FA,
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
