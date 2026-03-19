package mifit

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// XiaomiAuth holds Xiaomi session credentials needed for API calls.
type XiaomiAuth struct {
	UserID       string `json:"user_id"`
	CUserID      string `json:"c_user_id"`
	ServiceToken string `json:"service_token"`
	Ssecurity    string `json:"ssecurity"`
}

// LoginXiaomi authenticates via Xiaomi account (3-step OAuth flow).
// This is for users who registered with a Xiaomi/Mi account rather than Zepp/Amazfit.
func (c *Client) LoginXiaomi(email, password string) (*AuthResult, error) {
	jar, _ := cookiejar.New(nil)
	noRedirect := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	ua := "Mozilla/5.0 (Linux; Android 12; Pixel 4 Build/SP1A.210812.016.C1; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/131.0.6778.200 Mobile Safari/537.36"
	deviceID := "an_" + fmt.Sprintf("%x", md5.Sum([]byte(email)))

	// Step 1: Get login page params
	params := url.Values{"_json": {"true"}, "sid": {"miothealth"}, "_locale": {"en_US"}}
	req, _ := http.NewRequest("GET", "https://account.xiaomi.com/pass/serviceLogin?"+params.Encode(), nil)
	req.Header.Set("User-Agent", ua)
	req.AddCookie(&http.Cookie{Name: "userId", Value: email})
	req.AddCookie(&http.Cookie{Name: "deviceId", Value: deviceID})

	resp, err := noRedirect.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xiaomi login page: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var loginPage struct {
		Sign     string `json:"_sign"`
		Callback string `json:"callback"`
		Qs       string `json:"qs"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(string(body), "&&&START&&&")), &loginPage); err != nil {
		return nil, fmt.Errorf("parsing login page: %w", err)
	}
	if loginPage.Sign == "" {
		return nil, fmt.Errorf("missing _sign in login page response")
	}

	// Step 2: Submit credentials with MD5-hashed password
	hash := md5.Sum([]byte(password))
	passwordHash := strings.ToUpper(hex.EncodeToString(hash[:]))

	form := url.Values{
		"_json":    {"true"},
		"_sign":    {loginPage.Sign},
		"callback": {loginPage.Callback},
		"qs":       {loginPage.Qs},
		"sid":      {"miothealth"},
		"_locale":  {"en_US"},
		"user":     {email},
		"hash":     {passwordHash},
	}

	req, _ = http.NewRequest("POST", "https://account.xiaomi.com/pass/serviceLoginAuth2", strings.NewReader(form.Encode()))
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "deviceId", Value: deviceID})

	resp, err = noRedirect.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xiaomi auth: %w", err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	var authResp struct {
		Location  string `json:"location"`
		Ssecurity string `json:"ssecurity"`
		Code      int    `json:"code"`
		Nonce     int64  `json:"nonce"`
		UserID    int64  `json:"userId"`
		CUserID   string `json:"cUserId"`
		Result    string `json:"result"`
		Desc      string `json:"description"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(string(body), "&&&START&&&")), &authResp); err != nil {
		return nil, fmt.Errorf("parsing auth response: %w", err)
	}

	if authResp.Code != 0 || authResp.Result != "ok" {
		if authResp.Code == 70016 {
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("xiaomi auth failed (code %d): %s", authResp.Code, authResp.Desc)
	}

	// Step 3: Follow location with clientSign to get serviceToken
	nonceStr := fmt.Sprintf("%d", authResp.Nonce)
	signInput := fmt.Sprintf("nonce=%s&%s", nonceStr, authResp.Ssecurity)
	h := sha1.Sum([]byte(signInput))
	clientSign := base64.StdEncoding.EncodeToString(h[:])

	tokenURL := authResp.Location + "&clientSign=" + url.QueryEscape(clientSign)
	req, _ = http.NewRequest("GET", tokenURL, nil)
	req.Header.Set("User-Agent", ua)

	resp, err = noRedirect.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xiaomi token: %w", err)
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()

	serviceToken := ""
	cUserID := ""
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "serviceToken" {
			serviceToken = cookie.Value
		}
		if cookie.Name == "cUserId" {
			cUserID = cookie.Value
		}
	}

	if serviceToken == "" {
		return nil, fmt.Errorf("service token not found in response cookies")
	}

	userIDStr := fmt.Sprintf("%d", authResp.UserID)
	if cUserID == "" {
		cUserID = authResp.CUserID
	}

	// Store Xiaomi auth for data API calls
	c.xiaomiAuth = &XiaomiAuth{
		UserID:       userIDStr,
		CUserID:      cUserID,
		ServiceToken: serviceToken,
		Ssecurity:    authResp.Ssecurity,
	}
	c.appToken = serviceToken // store for IsAuthenticated check
	c.userIDMi = userIDStr
	c.authMethod = "xiaomi"

	return &AuthResult{
		AppToken:   serviceToken,
		UserIDMi:   userIDStr,
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
		AuthMethod: "xiaomi",
		XiaomiAuth: c.xiaomiAuth,
	}, nil
}
