package mifit

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// AuthResult contains the result of a successful authentication.
type AuthResult struct {
	AppToken   string
	UserIDMi   string
	ExpiresAt  time.Time
	AuthMethod string      // "zepp" or "xiaomi"
	XiaomiAuth *XiaomiAuth // set when AuthMethod == "xiaomi"
}

// Login authenticates with Mi Fitness using email and password.
// Tries Zepp (Amazfit) login first, then falls back to Xiaomi account login.
func (c *Client) Login(email, password string) (*AuthResult, error) {
	// Try Zepp login first (for Zepp/Amazfit accounts)
	result, err := c.loginZepp(email, password)
	if err == nil {
		c.authMethod = "zepp"
		result.AuthMethod = "zepp"
		slog.Info("Mi Fitness login via Zepp succeeded", "user_id", result.UserIDMi)
		return result, nil
	}
	slog.Debug("Zepp login failed, trying Xiaomi login", "error", err)

	// Fall back to Xiaomi account login
	result, err = c.LoginXiaomi(email, password)
	if err == nil {
		slog.Info("Mi Fitness login via Xiaomi succeeded", "user_id", result.UserIDMi)
		return result, nil
	}

	return nil, fmt.Errorf("login failed for both Zepp and Xiaomi methods: %w", err)
}

// loginZepp authenticates with Zepp (Amazfit) using email and password.
func (c *Client) loginZepp(email, password string) (*AuthResult, error) {
	accessToken, err := c.getAccessToken(email, password)
	if err != nil {
		return nil, fmt.Errorf("zepp access token: %w", err)
	}

	result, err := c.exchangeToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("zepp token exchange: %w", err)
	}

	c.appToken = result.AppToken
	c.userIDMi = result.UserIDMi

	return result, nil
}

// getAccessToken performs Step 1: submit AES-encrypted credentials to Zepp user API.
// The API returns a 303 redirect with access and refresh tokens in the Location header query params.
func (c *Client) getAccessToken(email, password string) (string, error) {
	reqURL := authBaseURL + "/v2/registrations/tokens"

	// Build form payload
	form := url.Values{
		"emailOrPhone": {email},
		"state":        {"REDIRECTION"},
		"client_id":    {clientID},
		"password":     {password},
		"redirect_uri": {"https://s3-us-west-2.amazonaws.com/hm-registration/successsignin.html"},
		"region":       {"us-west-2"},
		"token":        {"access", "refresh"},
		"country_code": {"US"},
	}

	// AES-CBC encrypt the URL-encoded form data
	plaintext := []byte(form.Encode())
	encrypted, err := aesEncrypt(plaintext, []byte(zeppEncKey), []byte(zeppEncIV))
	if err != nil {
		return "", fmt.Errorf("encrypting payload: %w", err)
	}

	// Use a client that doesn't follow redirects — we need the Location header.
	noRedirectClient := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	headers := map[string]string{
		"app_name":        appName,
		"appname":         appName,
		"cv":              "151689_" + appVersion,
		"v":               "2.0",
		"appplatform":     "android_phone",
		"vb":              buildVer,
		"vn":              appVersion,
		"user-agent":      userAgentApp,
		"x-hm-ekv":       "1",
		"content-type":    "application/x-www-form-urlencoded; charset=UTF-8",
		"accept-encoding": "gzip",
	}

	resp, err := doPost(noRedirectClient, reqURL, encrypted, headers)
	if err != nil {
		return "", fmt.Errorf("login request: %w", err)
	}

	// The server responds with a 303 redirect. Extract tokens from Location header.
	if resp.Location != "" {
		parsed, err := url.Parse(resp.Location)
		if err != nil {
			return "", fmt.Errorf("parsing redirect URL: %w", err)
		}

		// Check for error in redirect
		if errCode := parsed.Query().Get("error"); errCode != "" {
			switch errCode {
			case "401":
				return "", fmt.Errorf("invalid email or password")
			case "429":
				return "", fmt.Errorf("too many login attempts, please try again later")
			default:
				return "", fmt.Errorf("authentication error (code %s)", errCode)
			}
		}

		accessToken := parsed.Query().Get("access")
		if accessToken != "" {
			return accessToken, nil
		}
	}

	// Fallback: try to find access token in response body (older API format)
	bodyStr := string(resp.Body)

	// Try JSON response
	var jsonResp struct {
		TokenInfo struct {
			LoginToken string `json:"login_token"`
		} `json:"token_info"`
	}
	if err := parseJSON(resp.Body, &jsonResp); err == nil && jsonResp.TokenInfo.LoginToken != "" {
		return jsonResp.TokenInfo.LoginToken, nil
	}

	return "", fmt.Errorf("access token not found (status %d): %s", resp.StatusCode, truncate(bodyStr, 300))
}

// exchangeToken performs Step 2: exchange access token for app token.
func (c *Client) exchangeToken(accessToken string) (*AuthResult, error) {
	reqURL := loginBaseURL + "/v2/client/login"

	form := url.Values{
		"code":               {accessToken},
		"device_id":          {uuid.New().String()},
		"device_model":       {"android_phone"},
		"app_version":        {appVersion},
		"dn":                 {"api-mifit.zepp.com,api-user.zepp.com,api-mifit.zepp.com,api-watch.zepp.com,app-analytics.zepp.com,auth.zepp.com,api-analytics.zepp.com"},
		"third_name":         {"huami"},
		"source":             {appName + ":" + appVersion + ":151689"},
		"app_name":           {appName},
		"country_code":       {"US"},
		"grant_type":         {"access_token"},
		"allow_registration": {"false"},
		"lang":               {"en"},
	}

	headers := map[string]string{
		"app_name":     "com.huami.webapp",
		"appname":      "com.huami.webapp",
		"origin":       "https://user.zepp.com",
		"referer":      "https://user.zepp.com/",
		"user-agent":   userAgentWeb,
		"content-type": "application/x-www-form-urlencoded; charset=UTF-8",
		"accept":       "application/json, text/plain, */*",
	}

	body, err := doFormPost(c.httpClient, reqURL, form, headers)
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
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}, nil
}

// aesEncrypt encrypts data using AES-CBC with PKCS7 padding.
func aesEncrypt(plaintext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// PKCS7 padding
	padLen := aes.BlockSize - len(plaintext)%aes.BlockSize
	padding := make([]byte, padLen)
	for i := range padding {
		padding[i] = byte(padLen)
	}
	plaintext = append(plaintext, padding...)

	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	return ciphertext, nil
}
