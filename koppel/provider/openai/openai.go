package openai

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"gosuda.org/koppel/provider"
)

type OpenAIProvider struct {
	client *openai.Client
}

func NewProvider(ctx context.Context, options ...option.RequestOption) (*OpenAIProvider, error) {
	client := openai.NewClient(options...)
	return &OpenAIProvider{client: &client}, nil
}

func (p *OpenAIProvider) GenerateContent(ctx context.Context, model string, messages []provider.Message, options ...provider.Option) (provider.Response, error) {
	params := p.toChatParams(model, messages)
	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return &openaiResponse{resp: resp}, nil
}

func (p *OpenAIProvider) GenerateContentStream(ctx context.Context, model string, messages []provider.Message, options ...provider.Option) (provider.StreamResponse, error) {
	params := p.toChatParams(model, messages)
	stream := p.client.Chat.Completions.NewStreaming(ctx, params)
	return &openaiStreamResponse{stream: stream}, nil
}

func (p *OpenAIProvider) toChatParams(model string, messages []provider.Message) openai.ChatCompletionNewParams {
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, msg := range messages {
		parts := make([]openai.ChatCompletionContentPartUnionParam, len(msg.Parts))
		for j, part := range msg.Parts {
			switch v := part.(type) {
			case provider.TextPart:
				parts[j] = openai.TextContentPart(string(v))
			case provider.BlobPart:
				data := base64.StdEncoding.EncodeToString(v.Data)
				url := fmt.Sprintf("data:%s;base64,%s", v.MIMEType, data)
				parts[j] = openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL: url,
				})
			}
		}

		switch msg.Role {
		case "system":
			// System message only supports text in basic usage, or specific parts.
			// For simplicity, we'll join text parts if there are multiple.
			var text string
			for _, part := range msg.Parts {
				if t, ok := part.(provider.TextPart); ok {
					text += string(t)
				}
			}
			openaiMessages[i] = openai.SystemMessage(text)
		case "assistant", "model":
			var text string
			for _, part := range msg.Parts {
				if t, ok := part.(provider.TextPart); ok {
					text += string(t)
				}
			}
			openaiMessages[i] = openai.AssistantMessage(text)
		default:
			openaiMessages[i] = openai.UserMessage(parts)
		}
	}

	return openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(model),
		Messages: openaiMessages,
	}
}

type openaiResponse struct {
	resp *openai.ChatCompletion
}

func (r *openaiResponse) Text() string {
	if r.resp == nil || len(r.resp.Choices) == 0 {
		return ""
	}
	return r.resp.Choices[0].Message.Content
}

func (r *openaiResponse) Thought() string {
	return ""
}

type openaiStreamResponse struct {
	stream *ssestream.Stream[openai.ChatCompletionChunk]
}

func (s *openaiStreamResponse) Next() (provider.Response, error) {
	if !s.stream.Next() {
		if err := s.stream.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no more stream items")
	}
	chunk := s.stream.Current()
	return &openaiChunkResponse{chunk: chunk}, nil
}

func (s *openaiStreamResponse) Close() error {
	return s.stream.Close()
}

type openaiChunkResponse struct {
	chunk openai.ChatCompletionChunk
}

func (r *openaiChunkResponse) Text() string {
	if len(r.chunk.Choices) == 0 {
		return ""
	}
	return r.chunk.Choices[0].Delta.Content
}

func (r *openaiChunkResponse) Thought() string {
	return ""
}
