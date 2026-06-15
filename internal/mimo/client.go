package mimo

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Client is an HTTP client for the MiMo API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// NewClient creates a new MiMo API client.
func NewClient(baseURL, apiKey string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		httpClient: &http.Client{},
		baseURL:    baseURL,
		apiKey:     apiKey,
	}
}

// ChatCompletion sends a chat completion request and returns the raw HTTP response.
// The caller is responsible for closing the response body.
func (c *Client) ChatCompletion(ctx context.Context, model string, messages []Message, stream bool, extra map[string]any) (*http.Response, error) {
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   stream,
	}
	for k, v := range extra {
		body[k] = v
	}
	return c.doRequest(ctx, "/chat/completions", body)
}

// doRequest marshals body as JSON and performs a POST to the given API path.
func (c *Client) doRequest(ctx context.Context, path string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	// Check for API errors (non-2xx status)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// ---- Helper source types for building multimodal content ----

// ImageSource describes an image to include in a request.
type ImageSource struct {
	URL string // data URI or remote URL
}

// AudioSource describes audio to include in a request.
type AudioSource struct {
	Data   string // base64-encoded audio bytes
	Format string // e.g. "wav", "mp3"
}

// VideoSource describes a video to include in a request.
type VideoSource struct {
	URL string // remote URL
}

// FileToBase64DataURI reads a file, detects its MIME type, and returns a
// data URI string of the form "data:<mime>;base64,<encoded>".
func FileToBase64DataURI(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}
