package openai

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/shared"
	"github.com/openai/openai-go/v3/shared/constant"
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
	opts, err := provider.NewOptions(options...)
	if err != nil {
		return nil, err
	}

	params := p.toChatParams(model, messages)
	if len(opts.Tools) > 0 {
		tools := make([]openai.ChatCompletionToolUnionParam, len(opts.Tools))
		for i, t := range opts.Tools {
			tools[i] = openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
				Name:        t.Name,
				Description: param.NewOpt(t.Description),
				Parameters:  shared.FunctionParameters(t.InputSchema.(map[string]interface{})),
			})
		}
		params.Tools = tools
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return &openaiResponse{resp: resp}, nil
}

func (p *OpenAIProvider) GenerateContentStream(ctx context.Context, model string, messages []provider.Message, options ...provider.Option) (provider.StreamResponse, error) {
	opts, err := provider.NewOptions(options...)
	if err != nil {
		return nil, err
	}

	params := p.toChatParams(model, messages)
	if len(opts.Tools) > 0 {
		tools := make([]openai.ChatCompletionToolUnionParam, len(opts.Tools))
		for i, t := range opts.Tools {
			tools[i] = openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
				Name:        t.Name,
				Description: param.NewOpt(t.Description),
				Parameters:  shared.FunctionParameters(t.InputSchema.(map[string]interface{})),
			})
		}
		params.Tools = tools
	}

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)
	return &openaiStreamResponse{stream: stream}, nil
}

func (p *OpenAIProvider) toChatParams(model string, messages []provider.Message) openai.ChatCompletionNewParams {
	var openaiMessages []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		role := msg.Role
		if role == "model" {
			role = "assistant"
		}

		switch role {
		case "system":

			for _, part := range msg.Parts {
				if t, ok := part.(provider.TextPart); ok {
					openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{
						OfSystem: &openai.ChatCompletionSystemMessageParam{
							Content: openai.ChatCompletionSystemMessageParamContentUnion{
								OfString: param.NewOpt(string(t)),
							},
							Role: constant.System("system"),
						},
					})
				}
			}

		case "assistant":

			var parts []openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion
			var toolCalls []openai.ChatCompletionMessageToolCallUnionParam
			for _, part := range msg.Parts {
				switch v := part.(type) {
				case provider.TextPart:
					parts = append(parts, openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
						OfText: &openai.ChatCompletionContentPartTextParam{
							Text: string(v),
							Type: constant.Text("text"),
						},
					})
				case provider.ToolCallPart:
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnionParam{
						OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
							ID: v.ID,
							Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
								Arguments: v.Arguments,
								Name:      v.Name,
							},
							Type: constant.Function("function"),
						},
					})
				}
			}
			paramMsg := openai.ChatCompletionAssistantMessageParam{}
			if len(parts) > 0 {
				paramMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfArrayOfContentParts: parts,
				}
			}
			if len(toolCalls) > 0 {
				paramMsg.ToolCalls = toolCalls
			}
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &paramMsg,
			})

		case "user":
			var parts []openai.ChatCompletionContentPartUnionParam
			for _, part := range msg.Parts {
				switch v := part.(type) {
				case provider.TextPart:
					parts = append(parts, openai.ChatCompletionContentPartUnionParam{
						OfText: &openai.ChatCompletionContentPartTextParam{
							Text: string(v),
							Type: constant.Text("text"),
						},
					})
				case provider.BlobPart:
					parts = append(parts, openai.ChatCompletionContentPartUnionParam{
						OfImageURL: &openai.ChatCompletionContentPartImageParam{
							ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
								URL: fmt.Sprintf("data:%s;base64,%s", v.MIMEType, base64.StdEncoding.EncodeToString(v.Data)),
							},
							Type: constant.ImageURL("image_url"),
						},
					})
				}
			}
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfArrayOfContentParts: parts,
					},
					Role: constant.User("user"),
				},
			})

		case "tool":
			for _, part := range msg.Parts {
				if v, ok := part.(provider.ToolResultPart); ok {
					openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{
						OfTool: &openai.ChatCompletionToolMessageParam{
							Content:    openai.ChatCompletionToolMessageParamContentUnion{OfString: param.NewOpt(v.Content)},
							ToolCallID: v.ID,
							Role:       constant.Tool("tool"),
						},
					})
				}
			}
		}
	}

	return openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(model),
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

func (r *openaiResponse) ToolCalls() []provider.ToolCallPart {
	if r.resp == nil || len(r.resp.Choices) == 0 {
		return nil
	}
	var calls []provider.ToolCallPart
	for _, call := range r.resp.Choices[0].Message.ToolCalls {
		calls = append(calls, provider.ToolCallPart{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		})
	}
	return calls
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

func (r *openaiChunkResponse) ToolCalls() []provider.ToolCallPart {
	if len(r.chunk.Choices) == 0 {
		return nil
	}
	var calls []provider.ToolCallPart
	for _, call := range r.chunk.Choices[0].Delta.ToolCalls {
		calls = append(calls, provider.ToolCallPart{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		})
	}
	return calls
}
