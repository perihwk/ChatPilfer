package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"uuid"
)

const teamsSearchBaseUrl = "https://substrate.office.com"

type searchMessagesParams struct {
	Query string
	Size  int // number of messages to return per API request
	From  int // pagination starting point in search results
}

type searchResult struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	ThreadID       string    `json:"thread_id"`
	SenderName     string    `json:"sender_name"`
	SenderEmail    string    `json:"sender_email"`
	Timestamp      time.Time `json:"timestamp"`
	Text           string    `json:"text"`
	HasAttachments bool      `json:"has_attachments"`
}

type TeamsSearchReqBody struct {
	EntityRequests []*EntityRequest `json:"entityRequests"`
	Cvid           string           `json:"cvid"`
	Scenario       searchScenario   `json:"scenario"`
}

type searchScenario struct {
	Name string `json:"Name"`
}

type EntityRequest struct {
	EntityType     string      `json:"entityType"`
	ContentSources []string    `json:"contentSources"`
	Query          searchQuery `json:"query"`
	From           int         `json:"from"`
	Size           int         `json:"size"`
}

type searchQuery struct {
	QueryString string `json:"queryString"`
}

type teamsSearchAPIResponse struct {
	ApiVersion           string          `json:"ApiVersion"`
	SearchTerms          []string        `json:"SearchTerms"`
	EntitySets           []entitySet     `json:"EntitySets"`
	RestrictedSearchMode string          `json:"RestrictedSearchMode"`
	Instrumentation      instrumentation `json:"Instrumentation"`
}

type instrumentation struct {
	TraceId string `json:"TraceId"`
}

type entitySet struct {
	ResultSets  []resultSet    `json:"ResultSets"`
	EntityType  string         `json:"EntityType"`
	SearchTerms []string       `json:"SearchTerms"`
	Properties  entitySetProps `json:"Properties"`
	QueryId     string         `json:"QueryId"`
	IsPartial   bool           `json:"IsPartial"`
}

type entitySetProps struct {
	WordBreakerLanguage string `json:"WordBreakerLanguage"`
	HasParseException   bool   `json:"HasParseException"`
}

type resultSet struct {
	Provenance             string             `json:"Provenance"`
	ContentSources         []string           `json:"ContentSources"`
	Results                []searchResultItem `json:"Results"`
	Total                  int                `json:"Total"`
	TotalWithoutCollapsing int                `json:"TotalWithoutCollapsing"`
	MoreResultsAvailable   bool               `json:"MoreResultsAvailable"`
	QueryId                string             `json:"QueryId"`
	Rank                   int                `json:"Rank"`
	Properties             resultSetProps     `json:"Properties"`
}

type resultSetProps struct {
	RenderTemplateId string `json:"RenderTemplateId"`
}

type searchResultItem struct {
	Id                    string        `json:"Id"`
	Provenance            string        `json:"Provenance"`
	ContentSource         string        `json:"ContentSource"`
	ReferenceId           string        `json:"ReferenceId"`
	HitHighlightedSummary string        `json:"HitHighlightedSummary"`
	Rank                  int           `json:"Rank"`
	SearchKey             string        `json:"SearchKey"`
	SortOrderSource       string        `json:"SortOrderSource"`
	Source                messageSource `json:"Source"`
	Type                  string        `json:"Type"`
	ResultSearchType      string        `json:"ResultSearchType"`
	ProviderType          string        `json:"ProviderType"`
}

type messageSource struct {
	Id                      string           `json:"Id"`
	SortOrderSource         string           `json:"SortOrderSource"`
	ItemId                  idRef            `json:"ItemId"`
	ImmutableId             string           `json:"ImmutableId"`
	AnnouncementTitle       string           `json:"AnnouncementTitle"`
	ClientConversationId    string           `json:"ClientConversationId"`
	ClientThreadId          string           `json:"ClientThreadId"`
	DateTimeCreated         time.Time        `json:"DateTimeCreated"`
	DateTimeReceived        time.Time        `json:"DateTimeReceived"`
	DateTimeSent            time.Time        `json:"DateTimeSent"`
	DisplayTo               string           `json:"DisplayTo"`
	HasAttachments          bool             `json:"HasAttachments"`
	Importance              string           `json:"Importance"`
	InternetMessageId       string           `json:"InternetMessageId"`
	ParentFolderDisplayName string           `json:"ParentFolderDisplayName"`
	Preview                 string           `json:"Preview"`
	Sender                  emailAddressWrap `json:"Sender"`
	Subject                 string           `json:"Subject"`
	ConversationId          idRef            `json:"ConversationId"`
	From                    emailAddressWrap `json:"From"`
	MailboxGuids            []string         `json:"MailboxGuids"`
	MentionsPreview         mentionsPreview  `json:"MentionsPreview"`
}

type idRef struct {
	Id string `json:"Id"`
}

type emailAddressWrap struct {
	EmailAddress emailAddress `json:"EmailAddress"`
}

type emailAddress struct {
	Name    string `json:"Name"`
	Address string `json:"Address"`
}

type mentionsPreview struct {
	IsMentioned bool `json:"IsMentioned"`
}

type fileSearchResult struct {
	ID               string    `json:"id"`
	FileName         string    `json:"file_name"`
	FileType         string    `json:"file_type"`
	Size             int       `json:"size"`
	Author           string    `json:"author"`
	ModifiedBy       string    `json:"modified_by"`
	LastModifiedTime time.Time `json:"last_modified_time"`
	URL              string    `json:"url"`
	SiteTitle        string    `json:"site_title"`
}

type filesSearchAPIResponse struct {
	ApiVersion           string          `json:"ApiVersion"`
	SearchTerms          []string        `json:"SearchTerms"`
	EntitySets           []fileEntitySet `json:"EntitySets"`
	WasGroupsRestricted  bool            `json:"WasGroupsRestricted"`
	RestrictedSearchMode string          `json:"RestrictedSearchMode"`
	Instrumentation      instrumentation `json:"Instrumentation"`
}

type fileEntitySet struct {
	ResultSets []fileResultSet    `json:"ResultSets"`
	EntityType string             `json:"EntityType"`
	Properties fileEntitySetProps `json:"Properties"`
	QueryId    string             `json:"QueryId"`
	IsPartial  bool               `json:"IsPartial"`
}

type fileEntitySetProps struct {
	WordBreakerLanguage string `json:"WordBreakerLanguage"`
	SourceId            string `json:"SourceId"`
	HasParseException   bool   `json:"HasParseException"`
}

type fileResultSet struct {
	Provenance             string           `json:"Provenance"`
	ContentSources         []string         `json:"ContentSources"`
	Results                []fileResultItem `json:"Results"`
	Total                  int              `json:"Total"`
	TotalWithoutCollapsing int              `json:"TotalWithoutCollapsing"`
	MoreResultsAvailable   bool             `json:"MoreResultsAvailable"`
	QueryId                string           `json:"QueryId"`
	Rank                   int              `json:"Rank"`
	QueryModification      string           `json:"QueryModification"`
	Properties             resultSetProps   `json:"Properties"`
}

type fileResultItem struct {
	Id            string     `json:"Id"`
	Provenance    string     `json:"Provenance"`
	ContentSource string     `json:"ContentSource"`
	ReferenceId   string     `json:"ReferenceId"`
	Rank          int        `json:"Rank"`
	Source        fileSource `json:"Source"`
	Type          string     `json:"Type"`
	ProviderType  string     `json:"ProviderType"`
}

type fileSource struct {
	DocId                                   string    `json:"DocId"`
	FileType                                string    `json:"FileType"`
	Filename                                string    `json:"Filename"`
	WorkingSetId                            string    `json:"WorkingSetId"`
	Title                                   string    `json:"Title"`
	ContentClass                            string    `json:"ContentClass"`
	ContentTypeId                           string    `json:"ContentTypeId"`
	DefaultEncodingUrl                      string    `json:"DefaultEncodingUrl"`
	LinkingUrl                              string    `json:"LinkingUrl"`
	Path                                    string    `json:"Path"`
	SPWebUrl                                string    `json:"SPWebUrl"`
	SiteTemplateId                          int       `json:"SiteTemplateId"`
	SiteTitle                               string    `json:"SiteTitle"`
	SiteId                                  string    `json:"SiteId"`
	ListID                                  string    `json:"ListID"`
	WebId                                   string    `json:"WebId"`
	UniqueID                                string    `json:"UniqueID"`
	LastModifiedTime                        time.Time `json:"LastModifiedTime"`
	ModifiedBy                              string    `json:"ModifiedBy"`
	IsDocument                              bool      `json:"isDocument"`
	IsContainer                             bool      `json:"IsContainer"`
	IsExternalContent                       bool      `json:"isexternalcontent"`
	ViewsLifeTimeUniqueUsers                int       `json:"ViewsLifeTimeUniqueUsers"`
	ViewCount                               int       `json:"ViewCount"`
	ViewerCount                             int       `json:"ViewerCount"`
	IntentScore                             int       `json:"IntentScore"`
	Author                                  string    `json:"Author"`
	Created                                 time.Time `json:"Created"`
	Size                                    int       `json:"Size"`
	Description                             string    `json:"Description"`
	LastSharedWithMailboxOwnerByDisplayName string    `json:"LastSharedWithMailboxOwnerByDisplayName"`
	LastSharedWithMailboxOwnerBySmtp        string    `json:"LastSharedWithMailboxOwnerBySmtp"`
	LastSharedWithMailboxOwnerDateTime      time.Time `json:"LastSharedWithMailboxOwnerDateTime"`
	FileID                                  string    `json:"FileID"`
	FromFileCI                              bool      `json:"FromFileCI"`
}

// individual search request does not handle pagination
func (sp *TeamsSearchProvider) searchMessages(params *searchMessagesParams) (int, []searchResult, error) {
	searchUrl, err := url.JoinPath(teamsSearchBaseUrl, "searchservice", "api", "v2", "query")
	if err != nil {
		return 0, nil, err
	}

	if params.Query == "" {
		return 0, nil, fmt.Errorf("query param is required")
	}

	body, err := json.Marshal(&TeamsSearchReqBody{
		EntityRequests: []*EntityRequest{
			{
				EntityType:     "Message",
				ContentSources: []string{"Teams"},
				Query: searchQuery{
					QueryString: params.Query,
				},
				From: params.From,
				Size: params.Size,
			},
		},
		Cvid: uuid.New().String(),
		Scenario: searchScenario{
			Name: "powerbar",
		},
	})
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest("POST", searchUrl, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}

	resp, err := sp.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	var apiResp teamsSearchAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, nil, err
	}

	if len(apiResp.EntitySets) == 0 || len(apiResp.EntitySets[0].ResultSets) == 0 {
		return 0, nil, nil
	}

	rs := apiResp.EntitySets[0].ResultSets[0]

	var result []searchResult
	for _, match := range rs.Results {
		result = append(result, searchResult{
			ID:             match.Id,
			ConversationID: match.Source.ClientConversationId,
			ThreadID:       match.Source.ClientThreadId,
			SenderName:     match.Source.Sender.EmailAddress.Name,
			SenderEmail:    match.Source.Sender.EmailAddress.Address,
			Timestamp:      match.Source.DateTimeSent,
			Text:           match.HitHighlightedSummary,
			HasAttachments: match.Source.HasAttachments,
		})
	}

	return rs.Total, result, nil
}

// searchAllMessages wraps searchMessages and exhausts all pages
func (sp *TeamsSearchProvider) searchAllMessages(query string) ([]searchResult, error) {
	var allResults []searchResult
	from := 0
	size := 25 // messages per request, fewest round trips

	for {
		total, resp, err := sp.searchMessages(&searchMessagesParams{
			Query: query,
			From:  from,
			Size:  size,
		})
		if err != nil {
			return nil, fmt.Errorf("fetching from %d: %w", from, err)
		}

		allResults = append(allResults, resp...)

		from += size
		if from >= total {
			break
		}
	}

	return allResults, nil
}

// perform search and return the searchResult. Do not do any regex to extract interesting information
func (sp *TeamsSearchProvider) SearchMessages(queries []string) (map[string][]any, error) {
	// track all results so we don't add duplicate messages if there are multiple query matches
	seen := make(map[string]bool)

	searchResults := make(map[string][]any)
	for _, query := range queries {
		results, err := sp.searchAllMessages(query)
		if err != nil {
			continue
		}

		for _, result := range results {
			if seen[result.ID] {
				continue
			}

			seen[result.ID] = true
			searchResults[query] = append(searchResults[query], result)
		}
	}

	return searchResults, nil
}

// search messages then attempt to extract the useful information via regex
func (sp *TeamsSearchProvider) SearchMessagesRegex(queries []string, regex string) (map[string][]any, error) {
	regexPattern := regexp.MustCompile(regex)

	// track all results so we don't add duplicate messages if there are multiple query matches
	type resultKey struct {
		match string
		id    string
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
					match: match,
					id:    result.ID,
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

// individual search request does not handle pagination
func (sp *TeamsSearchProvider) searchFiles(params *searchMessagesParams) (int, []fileSearchResult, error) {
	searchUrl, err := url.JoinPath(teamsSearchBaseUrl, "searchservice", "api", "v2", "query")
	if err != nil {
		return 0, nil, err
	}

	if params.Query == "" {
		return 0, nil, fmt.Errorf("query param is required")
	}

	body, err := json.Marshal(&TeamsSearchReqBody{
		EntityRequests: []*EntityRequest{
			{
				EntityType:     "File",
				ContentSources: []string{"OneDriveBusiness", "SharePoint", "Exchange"},
				Query: searchQuery{
					QueryString: params.Query,
				},
				From: params.From,
				Size: params.Size,
			},
		},
		Cvid: uuid.New().String(),
		Scenario: searchScenario{
			Name: "powerbar",
		},
	})
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest("POST", searchUrl, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}

	resp, err := sp.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	var apiResp filesSearchAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, nil, err
	}

	if len(apiResp.EntitySets) == 0 || len(apiResp.EntitySets[0].ResultSets) == 0 {
		return 0, nil, nil
	}

	rs := apiResp.EntitySets[0].ResultSets[0]

	var result []fileSearchResult
	for _, match := range rs.Results {
		result = append(result, fileSearchResult{
			ID:               match.Id,
			FileName:         match.Source.Filename,
			FileType:         match.Source.FileType,
			Size:             match.Source.Size,
			Author:           match.Source.Author,
			ModifiedBy:       match.Source.ModifiedBy,
			LastModifiedTime: match.Source.LastModifiedTime,
			URL:              match.Source.LinkingUrl,
			SiteTitle:        match.Source.SiteTitle,
		})
	}

	return rs.Total, result, nil
}

// searchAllFiles wraps searchFiles and exhausts all pages
func (sp *TeamsSearchProvider) searchAllFiles(query string) ([]fileSearchResult, error) {
	var allResults []fileSearchResult
	from := 0
	size := 25 // files per request, fewest round trips

	for {
		total, resp, err := sp.searchFiles(&searchMessagesParams{
			Query: query,
			From:  from,
			Size:  size,
		})
		if err != nil {
			return nil, fmt.Errorf("fetching from %d: %w", from, err)
		}

		allResults = append(allResults, resp...)

		from += size
		if from >= total {
			break
		}
	}

	return allResults, nil
}

func (sp *TeamsSearchProvider) SearchFiles(queries []string) (map[string][]any, error) {
	// track all results so we don't add duplicate files if there are multiple query matches
	seen := make(map[string]bool)

	searchResults := make(map[string][]any)
	for _, query := range queries {
		results, err := sp.searchAllFiles(query)
		if err != nil {
			continue
		}

		for _, result := range results {
			if seen[result.ID] {
				continue
			}

			seen[result.ID] = true
			searchResults[query] = append(searchResults[query], result)
		}
	}

	return searchResults, nil
}
