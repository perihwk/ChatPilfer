package slack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"
)

const (
	firefoxUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:155.0) Gecko/20100101 Firefox/155.0"
	workspaceRegex   = `([a-zA-Z0-9\-]+\.slack\.com)/ssb/redirect`
	apiTokenRegex    = `"api_token"\s*:\s*"([^"]+)"`
	userIDRegex      = `"user_id"\s*:\s*"([^"]+)"`
	teamIDRegex      = `"team_id"\s*:\s*"([^"]+)"`
	slackBaseUrl     = "https://slack.com"
)

// cache for data the authed user has access to
type cache struct {
	user     *User
	channels []Channel
	mu       sync.Mutex
}

type User struct {
	UserID   string `json:"user_id"`
	TeamID   string `json:"team_id"`
	FullName string `json:"full_name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"is_admin"`
	IsOwner  bool   `json:"is_owner"`
	Has2FA   bool   `json:"has_2fa"`
}

type Channel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsPrivate bool   `json:"is_private"`
}

type Config struct {
	userAgent string
	dCookie   string
}

type authTransport struct {
	token     string
	userAgent string
	base      http.RoundTripper
}

func (at *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header.Set("User-Agent", at.userAgent)
	cloned.Header.Set("Authorization", fmt.Sprintf("Bearer %s", at.token))

	return at.base.RoundTrip(cloned)
}

type rateLimitTransport struct {
	sleepFor      int
	retryAttempts int
	base          http.RoundTripper
}

func (rt *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.base == nil {
		return nil, fmt.Errorf("rateLimitTransport must wrap an authTransport")
	}

	// body of request is consumed by the http.Do so have to make a copy for retries
	var getBody func() (io.ReadCloser, error)
	if req.GetBody != nil {
		req = req.Clone(req.Context())
		getBody = req.GetBody
	}

	var lastResp *http.Response
	for attempt := range rt.retryAttempts {
		if getBody != nil {
			body, err := getBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}

		resp, err := rt.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		// only care if it is a 429
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		lastResp = resp

		if attempt == rt.retryAttempts-1 {
			break
		}

		requestedRetry, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		if err != nil || requestedRetry <= 0 {
			requestedRetry = rt.sleepFor
		}

		resp.Body.Close()

		select {
		case <-time.After(time.Duration(requestedRetry) * time.Second):
			continue
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}

	return lastResp, nil
}

func GetProvidersFromCookie(proxyServer, userAgent, dCookie string) ([]*SlackSearchProvider, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(slackBaseUrl)
	if err != nil {
		return nil, err
	}

	jar.SetCookies(u, []*http.Cookie{
		{
			Name:   "d",
			Value:  dCookie,
			Domain: "slack.com", // Allows all subdomains
		},
	})

	proxyUrl, err := url.Parse(proxyServer)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{Jar: jar, Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}

	if userAgent == "" {
		userAgent = firefoxUserAgent
	}

	workspaces, err := getWorkspaces(httpClient, userAgent)
	if err != nil {
		return nil, err
	}

	tokens, err := getSlackAPITokens(httpClient, workspaces, userAgent)
	if err != nil {
		return nil, err
	}

	var providers []*SlackSearchProvider
	for workspaceUrl, token := range tokens {
		name, authUser, err := validateAuth(httpClient, token, userAgent)
		if err != nil {
			continue
		}

		provider := NewSlackProvider(httpClient, name, authUser.UserID, authUser.TeamID, token, workspaceUrl, userAgent)
		providers = append(providers, provider)
	}

	return providers, nil
}

func validateAuth(httpClient *http.Client, token, userAgent string) (string, *User, error) {
	testAuthUrl, err := url.JoinPath(slackBaseUrl, "api", "auth.test")
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("GET", testAuthUrl, nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	if userAgent != "" {
		req.Header.Add("User-Agent", userAgent)
	} else {
		req.Header.Add("User-Agent", firefoxUserAgent)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}

	var apiResp struct {
		OK     bool   `json:"ok"`
		Team   string `json:"team"`
		User   string `json:"user"`
		UserID string `json:"user_id"`
		TeamID string `json:"team_id"`
		Error  string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", nil, err
	}
	if !apiResp.OK {
		fmt.Println(apiResp.Error)
		return "", nil, fmt.Errorf("slack auth.test not OK")
	}

	apiUser := &User{
		UserID:   apiResp.UserID,
		TeamID:   apiResp.TeamID,
		Username: apiResp.User,
	}

	return apiResp.Team, apiUser, nil
}

func getWorkspaces(httpClient *http.Client, userAgent string) ([]string, error) {
	req, err := http.NewRequest("GET", slackBaseUrl, nil)
	if err != nil {
		return nil, err
	}

	if userAgent != "" {
		req.Header.Add("User-Agent", userAgent)
	} else {
		req.Header.Add("User-Agent", firefoxUserAgent)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	Wbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []string
	workspaceRe := regexp.MustCompile(workspaceRegex)
	matches := workspaceRe.FindAllStringSubmatch(string(Wbody), -1)
	for _, match := range matches {
		if len(match) >= 2 {
			result = append(result, match[1])
		}
	}
	return result, nil
}

func getSlackAPITokens(httpClient *http.Client, workspaces []string, userAgent string) (map[string]string, error) {
	result := make(map[string]string)
	for _, wUrl := range workspaces {
		if result[wUrl] == "" {
			url, err := url.JoinPath("https://", wUrl, "ssb", "redirect")
			if err != nil {
				return nil, err
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}

			if userAgent != "" {
				req.Header.Add("User-Agent", userAgent)
			} else {
				req.Header.Add("User-Agent", firefoxUserAgent)
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				return nil, err
			}

			Wbody, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			var apiTokenRe = regexp.MustCompile(apiTokenRegex)
			matches := apiTokenRe.FindStringSubmatch(string(Wbody))
			if len(matches) >= 2 {
				result[wUrl] = matches[1]
			}
		}
	}
	return result, nil
}
