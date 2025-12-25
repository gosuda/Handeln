package gemini

import (
	"context"
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
	genaiContents := p.toGenAIContents(messages)
	resp, err := p.client.Models.GenerateContent(ctx, model, genaiContents, nil)
	if err != nil {
		return nil, err
	}
	return &geminiResponse{resp: resp}, nil
}

func (p *GeminiProvider) GenerateContentStream(ctx context.Context, model string, messages []provider.Message, options ...provider.Option) (provider.StreamResponse, error) {
	genaiContents := p.toGenAIContents(messages)
	stream := p.client.Models.GenerateContentStream(ctx, model, genaiContents, nil)
	return &geminiStreamResponse{stream: stream}, nil
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
			}
		}
		genaiContents[i] = &genai.Content{
			Role:  msg.Role,
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
		if !part.Thought {
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

type geminiStreamResponse struct {
	stream iter.Seq2[*genai.GenerateContentResponse, error]
	next   func() (*genai.GenerateContentResponse, error, bool)
	stop   func()
}

func (s *geminiStreamResponse) Next() (provider.Response, error) {
	if s.next == nil {
		s.next, s.stop = iter.Pull2(s.stream)
	}
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
