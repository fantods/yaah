package zai

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func generateToken(apiKey string) (string, error) {
	parts := strings.SplitN(apiKey, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid zai API key format: expected 'id.secret'")
	}

	id, secret := parts[0], parts[1]

	now := time.Now().Unix()
	header := map[string]string{"alg": "HS256", "sign_type": "SIGN"}
	payload := map[string]any{"api_key": id, "exp": now + 3600, "timestamp": now}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshaling jwt header: %w", err)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling jwt payload: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sig, nil
}
