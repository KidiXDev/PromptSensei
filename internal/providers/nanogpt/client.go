package nanogpt

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
		base = "https://nano-gpt.com/api/v1"
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
	return "nanogpt"
}

func (c *Client) Generate(ctx context.Context, req domain.GenerateRequest) (*domain.GenerateResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("nanogpt api_key is empty")
	}
	logging.Debug("nanogpt request", "url", c.baseURL+"/chat/completions", "model", req.Model, "temperature", req.Temperature, "max_tokens", req.MaxTokens)

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
		return nil, fmt.Errorf("nanogpt user prompts are empty")
	}

	payload := map[string]any{
		"model":       req.Model,
		"messages":    messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      false,
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
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	logging.Debug("nanogpt response status", "status_code", resp.StatusCode, "status", resp.Status)
	if resp.StatusCode >= 400 {
		msg := strings.TrimSpace(parsed.Error.Message)
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("nanogpt error: %s", msg)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("nanogpt returned no choices")
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
	if c.apiKey == "" {
		return nil, fmt.Errorf("nanogpt api_key is empty")
	}
	logging.Debug("nanogpt stream request", "url", c.baseURL+"/chat/completions", "model", req.Model, "temperature", req.Temperature, "max_tokens", req.MaxTokens)

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
		return nil, fmt.Errorf("nanogpt user prompts are empty")
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
			msg := strings.TrimSpace(parsed.Error.Message)
			if msg == "" {
				msg = resp.Status
			}
			return nil, fmt.Errorf("nanogpt error: %s", msg)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return nil, fmt.Errorf("nanogpt error: %s", strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	var outputBuilder strings.Builder
	var reasoningBuilder strings.Builder
	usage := domain.Usage{}
	sawDone := false

	for {
		frame, ok, scanErr := scanSSEFrame(scanner)
		if scanErr != nil {
			return nil, scanErr
		}
		if !ok {
			break
		}

		frame = strings.TrimSpace(frame)
		if frame == "" {
			continue
		}
		if frame == "[DONE]" {
			sawDone = true
			break
		}

		var chunk streamChatCompletionResponse
		if err := json.Unmarshal([]byte(frame), &chunk); err != nil {
			continue
		}

		if chunk.Error.Message != "" {
			return nil, fmt.Errorf("nanogpt stream error: %s", chunk.Error.Message)
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

		// NanoGPT can backfill usage-equivalent values in pricing metadata.
		if chunk.XNanoGPTPricing != nil {
			if usage.PromptTokens == 0 && chunk.XNanoGPTPricing.InputTokens > 0 {
				usage.PromptTokens = chunk.XNanoGPTPricing.InputTokens
			}
			if usage.CompletionTokens == 0 && chunk.XNanoGPTPricing.OutputTokens > 0 {
				usage.CompletionTokens = chunk.XNanoGPTPricing.OutputTokens
			}
			if usage.TotalTokens == 0 && (usage.PromptTokens > 0 || usage.CompletionTokens > 0) {
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !sawDone {
		return nil, fmt.Errorf("nanogpt stream ended before [DONE]")
	}

	output := strings.TrimSpace(outputBuilder.String())
	if output == "" {
		return nil, fmt.Errorf("nanogpt returned empty stream output")
	}

	return &domain.GenerateResponse{
		Text:      output,
		Reasoning: strings.TrimSpace(reasoningBuilder.String()),
		Provider:  c.Name(),
		Usage:     usage,
	}, nil
}

func scanSSEFrame(scanner *bufio.Scanner) (string, bool, error) {
	dataLines := make([]string, 0, 4)

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")

		if strings.TrimSpace(line) == "" {
			if len(dataLines) > 0 {
				return strings.Join(dataLines, "\n"), true, nil
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}

	if err := scanner.Err(); err != nil {
		return "", false, err
	}
	if len(dataLines) > 0 {
		return strings.Join(dataLines, "\n"), true, nil
	}
	return "", false, nil
}

type streamChatCompletionResponse struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			Reasoning        string `json:"reasoning"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason any `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	XNanoGPTPricing *struct {
		InputTokens  int `json:"inputTokens"`
		OutputTokens int `json:"outputTokens"`
	} `json:"x_nanogpt_pricing"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
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
