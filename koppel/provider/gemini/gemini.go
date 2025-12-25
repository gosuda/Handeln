package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"google.golang.org/genai"
	"gosuda.org/koppel/provider"
)

type GeminiProvider struct {
	client *genai.Client
}

func NewProvider(ctx context.Context, config *genai.ClientConfig) (*GeminiProvider, error) {
	client, err := genai.NewClient(ctx, config)
	if err != nil {
		return nil, err
	}
	return &GeminiProvider{client: client}, nil
}

func (p *GeminiProvider) GenerateContent(ctx context.Context, model string, messages []provider.Message, options ...provider.Option) (provider.Response, error) {
	opts := &provider.Options{}
	for _, o := range options {
		if err := o(opts); err != nil {
			return nil, err
		}
	}

	config := &genai.GenerateContentConfig{}
	if opts.CacheName != "" {
		config.CachedContent = opts.CacheName
	}
	if len(opts.Tools) > 0 {
		genaiTools := make([]*genai.Tool, len(opts.Tools))
		for i, t := range opts.Tools {
			genaiTools[i] = &genai.Tool{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{
						Name:        t.Name,
						Description: t.Description,
						Parameters:  p.toGenAISchema(t.InputSchema),
					},
				},
			}
		}
		config.Tools = genaiTools
	}

	contents := p.toGenAIContents(messages)
	resp, err := p.client.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return nil, err
	}
	return &geminiResponse{resp: resp}, nil
}

func (p *GeminiProvider) GenerateContentStream(ctx context.Context, model string, messages []provider.Message, options ...provider.Option) (provider.StreamResponse, error) {
	opts := &provider.Options{}
	for _, o := range options {
		if err := o(opts); err != nil {
			return nil, err
		}
	}

	config := &genai.GenerateContentConfig{}
	if opts.CacheName != "" {
		config.CachedContent = opts.CacheName
	}
	if len(opts.Tools) > 0 {
		genaiTools := make([]*genai.Tool, len(opts.Tools))
		for i, t := range opts.Tools {
			genaiTools[i] = &genai.Tool{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{
						Name:        t.Name,
						Description: t.Description,
						Parameters:  p.toGenAISchema(t.InputSchema),
					},
				},
			}
		}
		config.Tools = genaiTools
	}

	contents := p.toGenAIContents(messages)
	it := p.client.Models.GenerateContentStream(ctx, model, contents, config)
	next, stop := iter.Pull2(it)
	return &geminiStreamResponse{next: next, stop: stop}, nil
}

// ... (ContextCacher methods)

func (p *GeminiProvider) toGenAISchema(schema interface{}) *genai.Schema {
	if schema == nil {
		return nil
	}
	b, _ := json.Marshal(schema)
	var s genai.Schema
	json.Unmarshal(b, &s)
	return &s
}

func (p *GeminiProvider) toGenAIContents(messages []provider.Message) []*genai.Content {
	genaiContents := make([]*genai.Content, len(messages))
	for i, msg := range messages {
		genaiParts := make([]*genai.Part, len(msg.Parts))
		for j, part := range msg.Parts {
			switch v := part.(type) {
			case provider.TextPart:
				genaiParts[j] = &genai.Part{Text: string(v)}
			case provider.BlobPart:
				genaiParts[j] = &genai.Part{InlineData: &genai.Blob{
					MIMEType: v.MIMEType,
					Data:     v.Data,
				}}
			case provider.ThoughtPart:
				genaiParts[j] = &genai.Part{Thought: true, Text: string(v)}
			case provider.ToolCallPart:
				var args map[string]interface{}
				json.Unmarshal([]byte(v.Arguments), &args)
				genaiParts[j] = &genai.Part{FunctionCall: &genai.FunctionCall{
					Name: v.Name,
					Args: args,
				}}
			case provider.ToolResultPart:
				var response map[string]interface{}
				if err := json.Unmarshal([]byte(v.Content), &response); err != nil {
					response = map[string]interface{}{"result": v.Content}
				}
				genaiParts[j] = &genai.Part{FunctionResponse: &genai.FunctionResponse{
					Name:     v.Name,
					Response: response,
				}}
			}
		}
		// In Gemini, ToolResultPart must have role "user" or "function"
		role := msg.Role
		if role == "tool" {
			role = "function"
		}
		genaiContents[i] = &genai.Content{
			Role:  role,
			Parts: genaiParts,
		}
	}
	return genaiContents
}

type geminiResponse struct {
	resp *genai.GenerateContentResponse
}

func (r *geminiResponse) Text() string {
	if r.resp == nil || len(r.resp.Candidates) == 0 || r.resp.Candidates[0].Content == nil {
		return ""
	}
	var text string
	for _, part := range r.resp.Candidates[0].Content.Parts {
		if !part.Thought && part.Text != "" {
			text += part.Text
		}
	}
	return text
}

func (r *geminiResponse) Thought() string {
	if r.resp == nil || len(r.resp.Candidates) == 0 || r.resp.Candidates[0].Content == nil {
		return ""
	}
	var thought string
	for _, part := range r.resp.Candidates[0].Content.Parts {
		if part.Thought {
			thought += part.Text
		}
	}
	return thought
}

func (r *geminiResponse) ToolCalls() []provider.ToolCallPart {
	if r.resp == nil || len(r.resp.Candidates) == 0 || r.resp.Candidates[0].Content == nil {
		return nil
	}
	var calls []provider.ToolCallPart
	for _, part := range r.resp.Candidates[0].Content.Parts {
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			calls = append(calls, provider.ToolCallPart{
				Name:      part.FunctionCall.Name,
				Arguments: string(args),
			})
		}
	}
	return calls
}

type geminiStreamResponse struct {
	next func() (*genai.GenerateContentResponse, error, bool)
	stop func()
}

func (s *geminiStreamResponse) Next() (provider.Response, error) {
	resp, err, ok := s.next()
	if !ok {
		return nil, fmt.Errorf("no more stream items")
	}
	if err != nil {
		return nil, err
	}
	return &geminiResponse{resp: resp}, nil
}

func (s *geminiStreamResponse) Close() error {
	if s.stop != nil {
		s.stop()
	}
	return nil
}
