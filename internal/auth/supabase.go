package auth

import (
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
