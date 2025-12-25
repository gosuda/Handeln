package gemini

import (
	"context"
	"fmt"
	"iter"
	"time"

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

	contents := p.toGenAIContents(messages)
	it := p.client.Models.GenerateContentStream(ctx, model, contents, config)
	next, stop := iter.Pull2(it)
	return &geminiStreamResponse{next: next, stop: stop}, nil
}

// ProviderContextCacher implementation

func (p *GeminiProvider) CreateCache(ctx context.Context, model string, messages []provider.Message, displayName string, ttl time.Duration) (*provider.ContextCache, error) {
	config := &genai.CreateCachedContentConfig{
		DisplayName: displayName,
		Contents:    p.toGenAIContents(messages),
		TTL:         ttl,
	}
	cached, err := p.client.Caches.Create(ctx, model, config)
	if err != nil {
		return nil, err
	}
	return p.fromGenAICachedContent(cached), nil
}

func (p *GeminiProvider) GetCache(ctx context.Context, name string) (*provider.ContextCache, error) {
	cached, err := p.client.Caches.Get(ctx, name, nil)
	if err != nil {
		return nil, err
	}
	return p.fromGenAICachedContent(cached), nil
}

func (p *GeminiProvider) DeleteCache(ctx context.Context, name string) error {
	_, err := p.client.Caches.Delete(ctx, name, nil)
	return err
}

func (p *GeminiProvider) ListCaches(ctx context.Context) ([]*provider.ContextCache, error) {
	var caches []*provider.ContextCache
	all := p.client.Caches.All(ctx)
	for cached, err := range all {
		if err != nil {
			return nil, err
		}
		caches = append(caches, p.fromGenAICachedContent(cached))
	}
	return caches, nil
}

func (p *GeminiProvider) fromGenAICachedContent(cached *genai.CachedContent) *provider.ContextCache {
	if cached == nil {
		return nil
	}
	return &provider.ContextCache{
		Name:        cached.Name,
		DisplayName: cached.DisplayName,
		Model:       cached.Model,
		ExpireTime:  cached.ExpireTime,
	}
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
