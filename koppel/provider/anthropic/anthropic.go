package anthropic

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"gosuda.org/koppel/provider"
)

type AnthropicProvider struct {
	client anthropic.Client
}

func NewProvider(ctx context.Context, options ...option.RequestOption) (*AnthropicProvider, error) {
	client := anthropic.NewClient(options...)
	return &AnthropicProvider{client: client}, nil
}

func (p *AnthropicProvider) GenerateContent(ctx context.Context, model string, messages []provider.Message, options ...provider.Option) (provider.Response, error) {
	params := p.toMessageParams(model, messages)
	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return &anthropicResponse{resp: resp}, nil
}

func (p *AnthropicProvider) GenerateContentStream(ctx context.Context, model string, messages []provider.Message, options ...provider.Option) (provider.StreamResponse, error) {
	params := p.toMessageParams(model, messages)
	stream := p.client.Messages.NewStreaming(ctx, params)
	return &anthropicStreamResponse{stream: stream}, nil
}

func (p *AnthropicProvider) toMessageParams(model string, messages []provider.Message) anthropic.MessageNewParams {
	var system []anthropic.TextBlockParam
	var anthropicMessages []anthropic.MessageParam

	for _, msg := range messages {
		if msg.Role == "system" {
			for _, part := range msg.Parts {
				if t, ok := part.(provider.TextPart); ok {
					system = append(system, anthropic.TextBlockParam{
						Text: string(t),
					})
				}
			}
			continue
		}

		blocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.Parts))
		for _, part := range msg.Parts {
			switch v := part.(type) {
			case provider.TextPart:
				blocks = append(blocks, anthropic.NewTextBlock(string(v)))
			case provider.BlobPart:
				encoded := base64.StdEncoding.EncodeToString(v.Data)
				blocks = append(blocks, anthropic.NewImageBlockBase64(v.MIMEType, encoded))
			case provider.ThoughtPart:
				// For now, signature is empty for input.
				blocks = append(blocks, anthropic.NewThinkingBlock("", string(v)))
			}
		}

		role := anthropic.MessageParamRoleUser
		if msg.Role == "assistant" || msg.Role == "model" {
			role = anthropic.MessageParamRoleAssistant
		}

		anthropicMessages = append(anthropicMessages, anthropic.MessageParam{
			Content: blocks,
			Role:    role,
		})
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		Messages:  anthropicMessages,
		MaxTokens: int64(4096), // Default reasonable max tokens
	}
	if len(system) > 0 {
		params.System = system
	}

	return params
}

type anthropicResponse struct {
	resp *anthropic.Message
}

func (r *anthropicResponse) Text() string {
	var text string
	for _, block := range r.resp.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return text
}

func (r *anthropicResponse) Thought() string {
	var thought string
	for _, block := range r.resp.Content {
		if block.Type == "thinking" {
			thought += block.Thinking
		}
	}
	return thought
}

type anthropicStreamResponse struct {
	stream *ssestream.Stream[anthropic.MessageStreamEventUnion]
}

func (s *anthropicStreamResponse) Next() (provider.Response, error) {
	if !s.stream.Next() {
		if err := s.stream.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no more stream items")
	}
	event := s.stream.Current()
	return &anthropicEventResponse{event: event}, nil
}

func (s *anthropicStreamResponse) Close() error {
	return s.stream.Close()
}

type anthropicEventResponse struct {
	event anthropic.MessageStreamEventUnion
}

func (r *anthropicEventResponse) Text() string {
	if r.event.Type == "content_block_delta" {
		if r.event.Delta.Type == "text_delta" {
			return r.event.Delta.Text
		}
	}
	return ""
}

func (r *anthropicEventResponse) Thought() string {
	if r.event.Type == "content_block_delta" {
		if r.event.Delta.Type == "thinking_delta" {
			return r.event.Delta.Thinking
		}
	}
	return ""
}
