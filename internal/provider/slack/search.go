package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type PinnedChatResp struct {
	OK    bool         `json:"ok"`
	Items []PinnedItem `json:"items"`
	Error string       `json:"error"`
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

		var apiResp PinnedChatResp
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return nil, err
		}

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
