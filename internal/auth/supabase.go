package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Principal struct {
	AccessToken string
	AuthUserID  string
	Email       string
	FullName    string
}

type UserInfo struct {
	ID           string         `json:"id"`
	Email        string         `json:"email"`
	UserMetadata map[string]any `json:"user_metadata"`
}

type userResponse struct {
	UserInfo
}

type SupabaseClient struct {
	baseURL        string
	anonKey        string
	serviceRoleKey string
	httpClient     *http.Client
}

func NewSupabaseClient(baseURL, anonKey, serviceRoleKey string) *SupabaseClient {
	return &SupabaseClient{
		baseURL:        strings.TrimRight(baseURL, "/"),
		anonKey:        anonKey,
		serviceRoleKey: serviceRoleKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func ParseBearerToken(header string) (string, bool) {
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return "", false
	}
	return token, true
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (c *SupabaseClient) Login(ctx context.Context, email, password string) (*TokenResponse, error) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/v1/token?grant_type=password", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build login request: %w", err)
	}
	req.Header.Set("apikey", c.anonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login rejected with status %d", resp.StatusCode)
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decode login response: %w", err)
	}
	return &token, nil
}

func (c *SupabaseClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/v1/token?grant_type=refresh_token", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set("apikey", c.anonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh rejected with status %d", resp.StatusCode)
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decode refresh response: %w", err)
	}
	return &token, nil
}

func (c *SupabaseClient) ValidateAccessToken(ctx context.Context, accessToken string) (*Principal, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/auth/v1/user", nil)
	if err != nil {
		return nil, fmt.Errorf("build auth request: %w", err)
	}

	req.Header.Set("apikey", c.anonKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth rejected token with status %d", resp.StatusCode)
	}

	var user UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode auth user: %w", err)
	}

	fullName := ""
	if v, ok := user.UserMetadata["full_name"].(string); ok {
		fullName = v
	} else if v, ok := user.UserMetadata["name"].(string); ok {
		fullName = v
	}

	return &Principal{
		AccessToken: accessToken,
		AuthUserID:  user.ID,
		Email:       user.Email,
		FullName:    fullName,
	}, nil
}
