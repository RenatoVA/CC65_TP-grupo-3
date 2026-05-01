package enricher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
}

func NewClient(apiKey, baseURL, model string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

type chatRequest struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:     c.model,
		Messages:  []message{{Role: "user", Content: prompt}},
		MaxTokens: 150,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	backoff := time.Second
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HTTP-Referer", "https://github.com/RenatoVA/tp-programacion-concurente")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http do: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read body: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
		}

		var out chatResponse
		if err := json.Unmarshal(respBody, &out); err != nil {
			return "", fmt.Errorf("unmarshal response: %w", err)
		}
		if out.Error != nil {
			return "", fmt.Errorf("API returned error: %s", out.Error.Message)
		}
		if len(out.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}
		return out.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("all retries failed: %w", lastErr)
}
