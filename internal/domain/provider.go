package domain

import "context"

type GenerateRequest struct {
	SystemPrompt string
	UserPrompts  []string
	Model        string
	Temperature  float64
	MaxTokens    int
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type GenerateResponse struct {
	Text     string
	Usage    Usage
	Provider string
}

type Provider interface {
	Name() string
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
}
