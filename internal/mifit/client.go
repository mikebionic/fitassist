package mifit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// Zepp/Huami API endpoints (updated 2025)
	authBaseURL    = "https://api-user-us2.zepp.com"
	loginBaseURL   = "https://api-mifit-us2.zepp.com"

	appName    = "com.huami.midong"
	clientID   = "HuaMi"
	appVersion = "9.12.5"
	buildVer   = "202509151347"

	userAgentApp = "Zepp/9.12.5 (Pixel 4; Android 12; Density/2.75)"
	userAgentWeb = "Mozilla/5.0 (X11; Linux x86_64; rv:133.0) Gecko/20100101 Firefox/133.0"

	// AES-CBC encryption params for token request payload
	zeppEncKey = "xeNtBVqzDc6tuNTh"
	zeppEncIV  = "MAAAYAAAAAAAAABg"
)

// Client interacts with the Mi Fitness (Huami/Zepp) API.
type Client struct {
	httpClient  *http.Client
	baseURL     string // data API base URL (region-specific)
	appToken    string
	userIDMi    string
	authMethod  string       // "zepp" or "xiaomi"
	xiaomiAuth  *XiaomiAuth  // set when using Xiaomi login
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

	req.Header.Set("User-Agent", userAgentApp)
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

// postResponse holds the HTTP response data we care about.
type postResponse struct {
	Body       []byte
	StatusCode int
	Location   string // redirect Location header, if any
}

// doPost executes a POST with raw body bytes and custom headers.
func doPost(client *http.Client, reqURL string, bodyData []byte, headers map[string]string) (*postResponse, error) {
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

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

	return &postResponse{
		Body:       body,
		StatusCode: resp.StatusCode,
		Location:   resp.Header.Get("Location"),
	}, nil
}

// doFormPost executes a POST with form-encoded data.
func doFormPost(client *http.Client, reqURL string, form url.Values, headers map[string]string) ([]byte, error) {
	h := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded; charset=UTF-8",
	}
	for k, v := range headers {
		h[k] = v
	}
	resp, err := doPost(client, reqURL, []byte(form.Encode()), h)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
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
