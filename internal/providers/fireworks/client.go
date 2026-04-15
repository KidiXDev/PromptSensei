package fireworks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/logging"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(cfg config.ProviderConfig) *Client {
	base := strings.TrimSpace(cfg.APIBaseURL)
	if base == "" {
		base = "https://api.fireworks.ai/inference"
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(base, "/"),
		apiKey:  strings.TrimSpace(cfg.APIKey),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Name() string {
	return "fireworks"
}

func (c *Client) Generate(ctx context.Context, req domain.GenerateRequest) (*domain.GenerateResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("fireworks api_key is empty")
	}
	logging.Debug("fireworks request", "url", c.baseURL+"/v1/chat/completions", "model", req.Model, "temperature", req.Temperature, "max_tokens", req.MaxTokens)

	messages := make([]map[string]string, 0, 1+len(req.UserPrompts))
	if req.SystemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": req.SystemPrompt})
	}
	for _, prompt := range req.UserPrompts {
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			continue
		}
		messages = append(messages, map[string]string{"role": "user", "content": prompt})
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("fireworks user prompts are empty")
	}

	payload := map[string]any{
		"model":       req.Model,
		"messages":    messages,
		"temperature": req.Temperature,
	}

	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	logging.Debug("fireworks response status", "status_code", resp.StatusCode, "status", resp.Status)

	if resp.StatusCode >= 400 {
		msg := strings.TrimSpace(parsed.Error.Message)
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("fireworks error: %s", msg)
	}

	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("fireworks returned no choices")
	}

	return &domain.GenerateResponse{
		Text:     strings.TrimSpace(parsed.Choices[0].Message.Content),
		Provider: c.Name(),
		Usage: domain.Usage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		},
	}, nil
}

func (c *Client) GenerateStream(ctx context.Context, req domain.GenerateRequest, onEvent domain.GenerateStreamCallback) (*domain.GenerateResponse, error) {
	resp, err := c.Generate(ctx, req)
	if err != nil {
		return nil, err
	}
	if onEvent != nil && resp != nil && strings.TrimSpace(resp.Text) != "" {
		if err := onEvent(domain.GenerateStreamEvent{TextDelta: resp.Text}); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
