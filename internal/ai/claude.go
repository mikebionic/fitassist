package ai

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/mike/fitassist/internal/config"
	"github.com/mike/fitassist/internal/model"
)

// Client wraps the Anthropic API for health-related conversations.
type Client struct {
	client *anthropic.Client
	model  string
	maxTok int
}

func NewClient(cfg config.ClaudeConfig) *Client {
	client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))
	maxTok := cfg.MaxTokens
	if maxTok <= 0 {
		maxTok = 4096
	}
	return &Client{
		client: &client,
		model:  cfg.Model,
		maxTok: maxTok,
	}
}

// ChatRequest holds what we need to call Claude.
type ChatRequest struct {
	SystemPrompt string
	Messages     []model.AIMessage
	UserMessage  string
}

// StreamCallback is called for each text chunk during streaming.
type StreamCallback func(text string) error

// Chat sends a non-streaming request and returns the full response.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (string, int, error) {
	msgs := buildMessages(req.Messages, req.UserMessage)

	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: int64(c.maxTok),
		System: []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		},
		Messages: msgs,
	})
	if err != nil {
		return "", 0, fmt.Errorf("claude api: %w", err)
	}

	text := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}

	tokens := int(resp.Usage.InputTokens + resp.Usage.OutputTokens)
	return text, tokens, nil
}

// ChatStream sends a streaming request and calls cb for each text delta.
// Returns the full assembled text and token count when done.
func (c *Client) ChatStream(ctx context.Context, req ChatRequest, cb StreamCallback) (string, int, error) {
	msgs := buildMessages(req.Messages, req.UserMessage)

	stream := c.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: int64(c.maxTok),
		System: []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		},
		Messages: msgs,
	})

	// Use Accumulate to track the full message including usage
	message := &anthropic.Message{}
	full := ""

	for stream.Next() {
		event := stream.Current()
		_ = message.Accumulate(event)

		switch variant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			if textDelta, ok := variant.Delta.AsAny().(anthropic.TextDelta); ok {
				full += textDelta.Text
				if err := cb(textDelta.Text); err != nil {
					return full, 0, err
				}
			}
		}
	}

	if err := stream.Err(); err != nil {
		return full, 0, fmt.Errorf("claude stream: %w", err)
	}

	tokens := int(message.Usage.InputTokens + message.Usage.OutputTokens)
	return full, tokens, nil
}

func buildMessages(history []model.AIMessage, userMsg string) []anthropic.MessageParam {
	var msgs []anthropic.MessageParam

	for _, m := range history {
		switch m.Role {
		case "user":
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case "assistant":
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		}
	}

	if userMsg != "" {
		msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)))
	}

	return msgs
}
