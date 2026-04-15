package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		base = "https://api.openai.com/v1"
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
	return "openai"
}

func (c *Client) Generate(ctx context.Context, req domain.GenerateRequest) (*domain.GenerateResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("openai api_key is empty")
	}
	logging.Debug("openai request", "url", c.baseURL+"/chat/completions", "model", req.Model, "temperature", req.Temperature, "max_tokens", req.MaxTokens)

	messages := make([]map[string]string, 0, 1+len(req.UserPrompts))
	messages = append(messages, map[string]string{"role": "system", "content": req.SystemPrompt})
	for _, prompt := range req.UserPrompts {
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			continue
		}
		messages = append(messages, map[string]string{"role": "user", "content": prompt})
	}
	if len(messages) == 1 {
		return nil, fmt.Errorf("openai user prompts are empty")
	}

	payload := map[string]any{
		"model":       req.Model,
		"messages":    messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(data))
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
	logging.Debug("openai response status", "status_code", resp.StatusCode, "status", resp.Status)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openai error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	reasoning := strings.TrimSpace(parsed.Choices[0].Message.ReasoningContent)
	if reasoning == "" {
		reasoning = strings.TrimSpace(parsed.Choices[0].Message.Reasoning)
	}

	return &domain.GenerateResponse{
		Text:      strings.TrimSpace(parsed.Choices[0].Message.Content),
		Reasoning: reasoning,
		Provider:  c.Name(),
		Usage: domain.Usage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		},
	}, nil
}

func (c *Client) GenerateStream(ctx context.Context, req domain.GenerateRequest, onEvent domain.GenerateStreamCallback) (*domain.GenerateResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("openai api_key is empty")
	}
	logging.Debug("openai stream request", "url", c.baseURL+"/chat/completions", "model", req.Model, "temperature", req.Temperature, "max_tokens", req.MaxTokens)

	messages := make([]map[string]string, 0, 1+len(req.UserPrompts))
	messages = append(messages, map[string]string{"role": "system", "content": req.SystemPrompt})
	for _, prompt := range req.UserPrompts {
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			continue
		}
		messages = append(messages, map[string]string{"role": "user", "content": prompt})
	}
	if len(messages) == 1 {
		return nil, fmt.Errorf("openai user prompts are empty")
	}

	payload := map[string]any{
		"model":       req.Model,
		"messages":    messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var parsed chatCompletionResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err == nil {
			return nil, fmt.Errorf("openai error: %s", parsed.Error.Message)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return nil, fmt.Errorf("openai error: %s", strings.TrimSpace(string(body)))
	}

	var outputBuilder strings.Builder
	var reasoningBuilder strings.Builder
	usage := domain.Usage{}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if raw == "" {
			continue
		}
		if raw == "[DONE]" {
			break
		}

		var chunk streamChatCompletionResponse
		if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
			continue
		}
		if chunk.Error.Message != "" {
			return nil, fmt.Errorf("openai stream error: %s", chunk.Error.Message)
		}
		if chunk.Usage != nil {
			usage = domain.Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}

		for _, choice := range chunk.Choices {
			textDelta := choice.Delta.Content
			reasoningDelta := choice.Delta.ReasoningContent
			if reasoningDelta == "" {
				reasoningDelta = choice.Delta.Reasoning
			}

			if textDelta != "" {
				outputBuilder.WriteString(textDelta)
			}
			if reasoningDelta != "" {
				reasoningBuilder.WriteString(reasoningDelta)
			}
			if onEvent != nil && (textDelta != "" || reasoningDelta != "") {
				if err := onEvent(domain.GenerateStreamEvent{
					TextDelta:      textDelta,
					ReasoningDelta: reasoningDelta,
				}); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	output := strings.TrimSpace(outputBuilder.String())
	if output == "" {
		return nil, fmt.Errorf("openai returned empty stream output")
	}

	return &domain.GenerateResponse{
		Text:      output,
		Reasoning: strings.TrimSpace(reasoningBuilder.String()),
		Provider:  c.Name(),
		Usage:     usage,
	}, nil
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			Reasoning        string `json:"reasoning"`
			ReasoningContent string `json:"reasoning_content"`
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

type streamChatCompletionResponse struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			Reasoning        string `json:"reasoning"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
