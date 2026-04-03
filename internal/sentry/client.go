// Package sentry provides a thin client for the Sentry REST API.
// Only the operations required by the operator are implemented.
package sentry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a minimal Sentry API client.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Sentry API client.
// baseURL should be "https://sentry.io" for Sentry Cloud, or your self-hosted URL.
// token is a Sentry auth token with project:write and project:read scopes.
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Project represents a Sentry project.
type Project struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// DSNKey represents a Sentry client key (DSN).
type DSNKey struct {
	ID    string  `json:"id"`
	Label string  `json:"label"`
	DSN   dsnData `json:"dsn"`
}

type dsnData struct {
	Public string `json:"public"`
	Secret string `json:"secret"`
	CSP    string `json:"csp"`
}

// GetProject retrieves a project by slug. Returns nil, nil if the project does not exist.
func (c *Client) GetProject(ctx context.Context, org, slug string) (*Project, error) {
	url := fmt.Sprintf("%s/api/0/projects/%s/%s/", c.baseURL, org, slug)
	req, err := c.newRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET project: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("decode project: %w", err)
	}
	return &project, nil
}

// CreateProject creates a new project under the given team and returns it.
func (c *Client) CreateProject(ctx context.Context, org, team, name, slug, platform string) (*Project, error) {
	url := fmt.Sprintf("%s/api/0/teams/%s/%s/projects/", c.baseURL, org, team)

	body := map[string]string{
		"name":     name,
		"platform": platform,
	}
	if slug != "" {
		body["slug"] = slug
	}

	req, err := c.newRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST project: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("decode created project: %w", err)
	}
	return &project, nil
}

// DeleteProject deletes a Sentry project.
func (c *Client) DeleteProject(ctx context.Context, org, slug string) error {
	url := fmt.Sprintf("%s/api/0/projects/%s/%s/", c.baseURL, org, slug)
	req, err := c.newRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE project: %w", err)
	}
	defer resp.Body.Close()

	// 404 means already gone — that's fine.
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return checkResponse(resp)
}

// GetProjectDSN returns the public DSN for the first client key of the project.
func (c *Client) GetProjectDSN(ctx context.Context, org, slug string) (string, error) {
	url := fmt.Sprintf("%s/api/0/projects/%s/%s/keys/", c.baseURL, org, slug)
	req, err := c.newRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET project keys: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return "", err
	}

	var keys []DSNKey
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return "", fmt.Errorf("decode keys: %w", err)
	}
	if len(keys) == 0 {
		return "", fmt.Errorf("project %s/%s has no DSN keys", org, slug)
	}
	return keys[0].DSN.Public, nil
}

func (c *Client) newRequest(ctx context.Context, method, url string, body interface{}) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		r = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// APIError represents a Sentry API error response.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("sentry API error %d: %s", e.StatusCode, e.Body)
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
}
