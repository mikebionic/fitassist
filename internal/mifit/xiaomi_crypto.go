package mifit

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"
)

// rc4 implements RC4 stream cipher.
type rc4Cipher struct {
	s    [256]byte
	i, j byte
}

func newRC4(key []byte) *rc4Cipher {
	var c rc4Cipher
	for i := 0; i < 256; i++ {
		c.s[i] = byte(i)
	}
	var j byte
	for i := 0; i < 256; i++ {
		j += c.s[i] + key[i%len(key)]
		c.s[i], c.s[j] = c.s[j], c.s[i]
	}
	return &c
}

func (c *rc4Cipher) crypt(data []byte) []byte {
	out := make([]byte, len(data))
	for idx, b := range data {
		c.i++
		c.j += c.s[c.i]
		c.s[c.i], c.s[c.j] = c.s[c.j], c.s[c.i]
		out[idx] = b ^ c.s[c.s[c.i]+c.s[c.j]]
	}
	return out
}

// makeRC4Drop1024 creates an RC4 cipher, dropping the first 1024 bytes of keystream.
func makeRC4Drop1024(keyB64 string) (*rc4Cipher, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("decoding RC4 key: %w", err)
	}
	rc4 := newRC4(keyBytes)
	rc4.crypt(make([]byte, 1024)) // drop first 1024 bytes
	return rc4, nil
}

// deriveRC4Key derives the RC4 key: base64(SHA256(ssecurity_bytes || nonce_bytes)).
func deriveRC4Key(ssecurityB64, nonceB64 string) (string, error) {
	ssecBytes, err := base64.StdEncoding.DecodeString(ssecurityB64)
	if err != nil {
		return "", err
	}
	nonceBytes, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", err
	}
	combined := append(ssecBytes, nonceBytes...)
	hash := sha256.Sum256(combined)
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

// generateNonce generates a nonce: base64(random_8_bytes || int32_be(now_minutes)).
func generateNonce() string {
	randBytes := make([]byte, 8)
	for i := range randBytes {
		randBytes[i] = byte(rand.Intn(256))
	}
	timeMinutes := int32(time.Now().UnixMilli() / 60000)
	timeBytes := []byte{
		byte(timeMinutes >> 24),
		byte(timeMinutes >> 16),
		byte(timeMinutes >> 8),
		byte(timeMinutes),
	}
	return base64.StdEncoding.EncodeToString(append(randBytes, timeBytes...))
}

// sha1Sign creates the SHA1 signature: base64(SHA1("METHOD&path&k1=v1&k2=v2&rc4_key")).
func sha1Sign(method, urlPath string, params map[string]string, rc4KeyB64 string) string {
	parts := []string{}
	if method != "" {
		parts = append(parts, strings.ToUpper(method))
	}
	if urlPath != "" {
		parts = append(parts, urlPath)
	}
	if len(params) > 0 {
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
		}
	}
	parts = append(parts, rc4KeyB64)

	signingString := strings.Join(parts, "&")
	hash := sha1.Sum([]byte(signingString))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// xiaomiEncryptParams encrypts request parameters for the Xiaomi health API.
// Returns URL-encoded query string with encrypted params + nonce + signature.
func xiaomiEncryptParams(method, path string, params map[string]string, ssecurityB64 string) (url.Values, error) {
	nonceB64 := generateNonce()
	rc4KeyB64, err := deriveRC4Key(ssecurityB64, nonceB64)
	if err != nil {
		return nil, fmt.Errorf("deriving RC4 key: %w", err)
	}

	// Encrypt each parameter value
	rc4Enc, err := makeRC4Drop1024(rc4KeyB64)
	if err != nil {
		return nil, err
	}

	encParams := make(map[string]string)
	for k, v := range params {
		encrypted := rc4Enc.crypt([]byte(v))
		encParams[k] = base64.StdEncoding.EncodeToString(encrypted)
	}

	// Generate signature
	signature := sha1Sign(method, path, encParams, rc4KeyB64)

	// Build result
	result := url.Values{}
	for k, v := range encParams {
		result.Set(k, v)
	}
	result.Set("_nonce", nonceB64)
	result.Set("rc4_hash__", signature)

	return result, nil
}

// xiaomiDecryptResponse decrypts a response from the Xiaomi health API.
func xiaomiDecryptResponse(encrypted string, nonceB64, ssecurityB64 string) (string, error) {
	rc4KeyB64, err := deriveRC4Key(ssecurityB64, nonceB64)
	if err != nil {
		return "", err
	}

	rc4Dec, err := makeRC4Drop1024(rc4KeyB64)
	if err != nil {
		return "", err
	}

	encBytes, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	decrypted := rc4Dec.crypt(encBytes)
	return string(decrypted), nil
}
