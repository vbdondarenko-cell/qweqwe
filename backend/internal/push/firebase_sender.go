package push

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const firebaseMessagingScope = "https://www.googleapis.com/auth/firebase.messaging"

type serviceAccountFile struct {
	ProjectID  string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey string `json:"private_key"`
	TokenURI   string `json:"token_uri"`
}

type FirebaseSender struct {
	projectID string
	clientEmail string
	privateKey *rsa.PrivateKey
	tokenURI string
	client *http.Client

	mu sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewFirebaseSender(projectID, credentialsFile string) (*FirebaseSender, error) {
	credentialsFile = strings.TrimSpace(credentialsFile)
	projectID = strings.TrimSpace(projectID)
	if projectID == "" || credentialsFile == "" {
		return nil, errors.New("firebase project id and service-account file are required")
	}
	raw, err := os.ReadFile(credentialsFile)
	if err != nil { return nil, fmt.Errorf("read firebase service account: %w", err) }
	var account serviceAccountFile
	if err := json.Unmarshal(raw, &account); err != nil { return nil, errors.New("invalid firebase service-account JSON") }
	if account.ProjectID != "" && account.ProjectID != projectID {
		return nil, errors.New("firebase service-account project mismatch")
	}
	account.ClientEmail = strings.TrimSpace(account.ClientEmail)
	account.TokenURI = strings.TrimSpace(account.TokenURI)
	if account.TokenURI == "" { account.TokenURI = "https://oauth2.googleapis.com/token" }
	if account.ClientEmail == "" || account.PrivateKey == "" {
		return nil, errors.New("firebase service-account credentials are incomplete")
	}
	if !strings.HasPrefix(account.TokenURI, "https://") {
		return nil, errors.New("firebase token URI must use https")
	}
	key, err := parseRSAPrivateKey([]byte(account.PrivateKey))
	if err != nil { return nil, err }
	return &FirebaseSender{
		projectID: projectID,
		clientEmail: account.ClientEmail,
		privateKey: key,
		tokenURI: account.TokenURI,
		client: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (s *FirebaseSender) Send(ctx context.Context, registrationToken string, message Message) error {
	registrationToken = strings.TrimSpace(registrationToken)
	if registrationToken == "" { return errors.New("empty FCM registration token") }
	accessToken, err := s.token(ctx)
	if err != nil { return err }
	payload := struct {
		Message struct {
			Token string `json:"token"`
			Notification *struct { Title string `json:"title"`; Body string `json:"body"` } `json:"notification,omitempty"`
			Data map[string]string `json:"data,omitempty"`
			Android struct { Priority string `json:"priority,omitempty"`; TTL string `json:"ttl,omitempty"` } `json:"android"`
		} `json:"message"`
	}{}
	payload.Message.Token = registrationToken
	if message.Title != "" || message.Body != "" {
		payload.Message.Notification = &struct { Title string `json:"title"`; Body string `json:"body"` }{Title: message.Title, Body: message.Body}
	}
	payload.Message.Data = message.Data
	payload.Message.Android.Priority = "HIGH"
	payload.Message.Android.TTL = "3600s"
	body, err := json.Marshal(payload)
	if err != nil { return err }
	endpoint := "https://fcm.googleapis.com/v1/projects/" + url.PathEscape(s.projectID) + "/messages:send"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil { return err }
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := s.client.Do(req)
	if err != nil { return fmt.Errorf("FCM send: %w", err) }
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("FCM send returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *FirebaseSender) token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if s.accessToken != "" && now.Add(time.Minute).Before(s.tokenExpiry) {
		return s.accessToken, nil
	}
	assertion, err := s.jwtAssertion(now)
	if err != nil { return "", err }
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURI, strings.NewReader(form.Encode()))
	if err != nil { return "", err }
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil { return "", fmt.Errorf("firebase OAuth token: %w", err) }
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil { return "", err }
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("firebase OAuth token returned HTTP %d", resp.StatusCode)
	}
	var tokenResponse struct { AccessToken string `json:"access_token"`; ExpiresIn int64 `json:"expires_in"`; TokenType string `json:"token_type"` }
	if err := json.Unmarshal(raw, &tokenResponse); err != nil { return "", errors.New("invalid firebase OAuth response") }
	if tokenResponse.AccessToken == "" || tokenResponse.ExpiresIn <= 0 { return "", errors.New("firebase OAuth response missing token") }
	s.accessToken = tokenResponse.AccessToken
	s.tokenExpiry = now.Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	return s.accessToken, nil
}

func (s *FirebaseSender) jwtAssertion(now time.Time) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg":"RS256","typ":"JWT"})
	claims, _ := json.Marshal(map[string]any{
		"iss": s.clientEmail,
		"scope": firebaseMessagingScope,
		"aud": s.tokenURI,
		"iat": now.Unix(),
		"exp": now.Add(time.Hour).Unix(),
	})
	encoded := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	hash := sha256.Sum256([]byte(encoded))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, hash[:])
	if err != nil { return "", err }
	return encoded + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func parseRSAPrivateKey(raw []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil { return nil, errors.New("firebase private key is not PEM") }
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok { return nil, errors.New("firebase private key is not RSA") }
		return rsaKey, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil { return key, nil }
	return nil, errors.New("unsupported firebase RSA private key")
}
