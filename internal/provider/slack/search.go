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

func (sp *SlackSearchProvider) GetPinnedChats() error {
	if sp.cache.channels == nil {
		return fmt.Errorf("no channels in cache make sure initialize was ran")
	}

	sp.cache.mu.Lock()
	channelsCopy := sp.cache.channels
	sp.cache.mu.Unlock()

	getPinnedChatsUrl, err := url.JoinPath(slackBaseUrl, "api", "pins.list")
	if err != nil {
		return err
	}

	for _, channel := range channelsCopy {
		req, err := http.NewRequest("GET", getPinnedChatsUrl, nil)
		if err != nil {
			return nil
		}

		q := req.URL.Query()
		q.Add("channel", channel.ID)

		req.URL.RawQuery = q.Encode()

		resp, err := sp.httpClient.Do(req)
		if err != nil {
			return err
		}

		var apiResp PinnedChatResp
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return err
		}

		pretty, err := json.MarshalIndent(apiResp, "", "    ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", pretty)
	}
	return nil
}
