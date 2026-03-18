package mifit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authBaseURL    = "https://api-user.huami.com"
	accountBaseURL = "https://account.huami.com"

	appName    = "com.xiaomi.hm.health"
	clientID   = "HuaMi"
	appVersion = "6.3.5"

	userAgent = "MiFit/6.3.5 (Linux; Android 12)"
)

// Client interacts with the Mi Fitness (Huami/Zepp) API.
type Client struct {
	httpClient *http.Client
	baseURL    string // data API base URL (region-specific)
	appToken   string
	userIDMi   string
}

// NewClient creates a new Mi Fitness API client.
func NewClient(apiBaseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: apiBaseURL,
	}
}

// SetAuth sets pre-existing auth credentials (loaded from DB).
func (c *Client) SetAuth(appToken, userIDMi string) {
	c.appToken = appToken
	c.userIDMi = userIDMi
}

// IsAuthenticated returns true if the client has auth credentials.
func (c *Client) IsAuthenticated() bool {
	return c.appToken != "" && c.userIDMi != ""
}

// Token returns the current app token.
func (c *Client) Token() string {
	return c.appToken
}

// UserID returns the Mi user ID.
func (c *Client) UserID() string {
	return c.userIDMi
}

// doRequest executes an authenticated request to the data API.
func (c *Client) doRequest(method, path string, params url.Values) ([]byte, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	reqURL := c.baseURL + path
	if params != nil {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("apptoken", c.appToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// doFormPost executes a POST with form data.
func doFormPost(client *http.Client, reqURL string, form url.Values, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return body, nil
}

// parseJSON is a helper to unmarshal JSON response.
func parseJSON(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parsing JSON: %w (body: %s)", err, truncate(string(data), 200))
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
