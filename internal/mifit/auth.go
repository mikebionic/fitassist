package mifit

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// AuthResult contains the result of a successful authentication.
type AuthResult struct {
	AppToken  string
	UserIDMi  string
	ExpiresAt time.Time
}

// Login authenticates with Mi Fitness using email and password.
// This is a 2-step process:
//  1. Get access token from Xiaomi account
//  2. Exchange it for an app token
func (c *Client) Login(email, password string) (*AuthResult, error) {
	// Step 1: Get access token via Xiaomi account registration endpoint.
	// This returns a redirect URL containing the access token.
	accessToken, countryCode, err := c.getAccessToken(email, password)
	if err != nil {
		return nil, fmt.Errorf("step 1 (access token): %w", err)
	}

	// Step 2: Exchange access token for app token.
	result, err := c.exchangeToken(accessToken, countryCode)
	if err != nil {
		return nil, fmt.Errorf("step 2 (app token): %w", err)
	}

	c.appToken = result.AppToken
	c.userIDMi = result.UserIDMi

	return result, nil
}

// getAccessToken performs Step 1: login via Huami user API.
func (c *Client) getAccessToken(email, password string) (accessToken, countryCode string, err error) {
	reqURL := fmt.Sprintf("%s/registrations/+%s/tokens", authBaseURL, url.PathEscape(email))

	form := url.Values{
		"state":        {"REDIRECTION"},
		"client_id":    {clientID},
		"redirect_uri": {"https://s3-us-west-2.amazonaws.com/hm-registration/successs498702.html"},
		"token":        {"access"},
		"password":     {password},
	}

	// Use a client that doesn't follow redirects — we need the redirect URL.
	noRedirectClient := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	body, err := doFormPost(noRedirectClient, reqURL, form, nil)
	if err != nil {
		return "", "", fmt.Errorf("login request: %w", err)
	}

	// The response body contains a redirect URL with the access token.
	// Parse the response to extract token info.
	var resp struct {
		TokenInfo struct {
			AppToken string `json:"app_token"`
			UserID   string `json:"user_id"`
			LoginToken string `json:"login_token"`
		} `json:"token_info"`
		RedirectURI string `json:"redirect"`
	}

	// Try direct JSON response first
	if err := parseJSON(body, &resp); err == nil && resp.TokenInfo.LoginToken != "" {
		// Some API versions return tokens directly
		return resp.TokenInfo.LoginToken, "", nil
	}

	// Otherwise, parse redirect URL from response body
	bodyStr := string(body)

	// Extract access token from redirect URL
	re := regexp.MustCompile(`access=([^&]+)`)
	matches := re.FindStringSubmatch(bodyStr)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("access token not found in response: %s", truncate(bodyStr, 300))
	}
	accessToken = matches[1]

	// Extract country code if present
	reCountry := regexp.MustCompile(`country_code=([^&"]+)`)
	countryMatches := reCountry.FindStringSubmatch(bodyStr)
	if len(countryMatches) >= 2 {
		countryCode = countryMatches[1]
	}

	return accessToken, countryCode, nil
}

// exchangeToken performs Step 2: exchange access token for app token.
func (c *Client) exchangeToken(accessToken, countryCode string) (*AuthResult, error) {
	reqURL := accountBaseURL + "/v2/client/login"

	form := url.Values{
		"app_name":     {appName},
		"app_version":  {appVersion},
		"code":         {accessToken},
		"country_code": {countryCode},
		"device_id":    {"fitassist_client"},
		"device_model": {"fitassist"},
		"grant_type":   {"access_token"},
		"third_name":   {"huami_phone"},
	}

	body, err := doFormPost(c.httpClient, reqURL, form, nil)
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}

	var resp struct {
		TokenInfo struct {
			AppToken   string `json:"app_token"`
			LoginToken string `json:"login_token"`
			UserID     string `json:"user_id"`
		} `json:"token_info"`
		ErrorCode interface{} `json:"error_code"`
	}

	if err := parseJSON(body, &resp); err != nil {
		return nil, err
	}

	if resp.TokenInfo.AppToken == "" && resp.TokenInfo.LoginToken == "" {
		return nil, fmt.Errorf("empty token in response: %s", truncate(string(body), 300))
	}

	token := resp.TokenInfo.AppToken
	if token == "" {
		token = resp.TokenInfo.LoginToken
	}

	return &AuthResult{
		AppToken:  token,
		UserIDMi:  resp.TokenInfo.UserID,
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour), // tokens typically last ~90 days
	}, nil
}
