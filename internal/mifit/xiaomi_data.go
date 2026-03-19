package mifit

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const xiaomiHealthBaseURL = "https://ru.hlth.io.mi.com"

// doXiaomiRequest executes an authenticated, RC4-encrypted request to the Xiaomi health API.
func (c *Client) doXiaomiRequest(method, path string, params map[string]string) ([]byte, error) {
	if c.xiaomiAuth == nil {
		return nil, fmt.Errorf("not authenticated with Xiaomi")
	}

	nonceB64 := generateNonce()
	rc4KeyB64, err := deriveRC4Key(c.xiaomiAuth.Ssecurity, nonceB64)
	if err != nil {
		return nil, fmt.Errorf("deriving key: %w", err)
	}

	// Encrypt each param value with a fresh RC4 cipher
	rc4Enc, err := makeRC4Drop1024(rc4KeyB64)
	if err != nil {
		return nil, err
	}
	encParams := make(map[string]string)
	for k, v := range params {
		encParams[k] = base64.StdEncoding.EncodeToString(rc4Enc.crypt([]byte(v)))
	}

	// Generate SHA1 signature over encrypted params
	signature := sha1Sign(method, path, encParams, rc4KeyB64)

	// Build query with encrypted params + nonce + signature
	query := url.Values{}
	for k, v := range encParams {
		query.Set(k, v)
	}
	query.Set("_nonce", nonceB64)
	query.Set("rc4_hash__", signature)

	// Make request
	var req *http.Request
	fullURL := xiaomiHealthBaseURL + path

	if strings.ToUpper(method) == "GET" {
		req, err = http.NewRequest("GET", fullURL+"?"+query.Encode(), nil)
	} else {
		req, err = http.NewRequest("POST", fullURL, strings.NewReader(query.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Android-12-9.12.5-google-Pixel 4")
	req.AddCookie(&http.Cookie{Name: "cUserId", Value: c.xiaomiAuth.CUserID})
	req.AddCookie(&http.Cookie{Name: "serviceToken", Value: c.xiaomiAuth.ServiceToken})
	req.AddCookie(&http.Cookie{Name: "locale", Value: "en_us"})

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication expired, please re-link your account")
	}
	if resp.StatusCode == 204 || (resp.StatusCode == 200 && len(body) == 0) {
		return nil, ErrNoData
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	// Decrypt response (RC4 encrypted and base64 encoded)
	decrypted, err := xiaomiDecryptResponse(string(body), nonceB64, c.xiaomiAuth.Ssecurity)
	if err != nil {
		// Response might not be encrypted
		return body, nil
	}

	return []byte(decrypted), nil
}
