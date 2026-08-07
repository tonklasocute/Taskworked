// Package ai wraps the Claude Messages API to power the AI Assistant module
// (task generation, summarization, and suggestion features). It talks to
// the API directly over net/http rather than pulling in the full Anthropic
// SDK — every call here is a single structured-output request with no
// streaming, tool loops, or batching, which raw HTTP covers in a few lines.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

const (
	messagesURL     = "https://api.anthropic.com/v1/messages"
	anthropicVer    = "2023-06-01"
	defaultModel    = "claude-opus-5"
	defaultMaxToken = 2048
)

// Client is the local abstraction over the Claude API that Service depends
// on, so tests can supply a fake instead of making real network calls.
type Client interface {
	// CompleteJSON sends systemPrompt + userPrompt with output constrained
	// to schema (a JSON Schema object per Anthropic's structured-outputs
	// format), then unmarshals the model's response into target.
	CompleteJSON(ctx context.Context, systemPrompt, userPrompt string, schema map[string]any, target any) error
}

type anthropicClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewAnthropicClient(apiKey string) Client {
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = defaultModel
	}
	return &anthropicClient{apiKey: apiKey, model: model, httpClient: &http.Client{Timeout: 60 * time.Second}}
}

type messageRequest struct {
	Model        string        `json:"model"`
	MaxTokens    int           `json:"max_tokens"`
	System       string        `json:"system,omitempty"`
	Messages     []message     `json:"messages"`
	OutputConfig *outputConfig `json:"output_config,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type outputConfig struct {
	Format outputFormat `json:"format"`
}

type outputFormat struct {
	Type   string         `json:"type"`
	Schema map[string]any `json:"schema"`
}

type messageResponse struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type errorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *anthropicClient) CompleteJSON(ctx context.Context, systemPrompt, userPrompt string, schema map[string]any, target any) error {
	if c.apiKey == "" {
		return apperrors.Internal("AI assistant is not configured (missing ANTHROPIC_API_KEY)")
	}

	reqBody, err := json.Marshal(messageRequest{
		Model:     c.model,
		MaxTokens: defaultMaxToken,
		System:    systemPrompt,
		Messages:  []message{{Role: "user", Content: userPrompt}},
		OutputConfig: &outputConfig{Format: outputFormat{
			Type:   "json_schema",
			Schema: schema,
		}},
	})
	if err != nil {
		return apperrors.Internal("failed to build AI request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, messagesURL, bytes.NewReader(reqBody))
	if err != nil {
		return apperrors.Internal("failed to build AI request")
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVer)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return apperrors.Internal("AI request failed: " + err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return apperrors.Internal("failed to read AI response")
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr errorResponse
		_ = json.Unmarshal(respBody, &apiErr)
		return apperrors.Internal(fmt.Sprintf("AI request rejected: %s", apiErr.Error.Message))
	}

	var parsed messageResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return apperrors.Internal("failed to parse AI response")
	}
	if parsed.StopReason == "refusal" || len(parsed.Content) == 0 {
		return apperrors.Internal("AI declined to respond")
	}

	if err := json.Unmarshal([]byte(parsed.Content[0].Text), target); err != nil {
		return apperrors.Internal("failed to parse AI response payload")
	}
	return nil
}
