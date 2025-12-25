package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Options struct {
	CacheName string `json:"cache_name,omitempty"`
}

type Option func(*Options) error

type Part interface {
	IsPart()
}

type TextPart string

func (TextPart) IsPart() {}

type BlobPart struct {
	MIMEType string `json:"mime_type"`
	Data     []byte `json:"data"`
}

func (BlobPart) IsPart() {}

type ThoughtPart string

func (ThoughtPart) IsPart() {}

type Response interface {
	Text() string
	Thought() string
}

type StreamResponse interface {
	Next() (Response, error)
	io.Closer
}

type Message struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

type partJSON struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	MIMEType string      `json:"mime_type,omitempty"`
	Data     []byte      `json:"data,omitempty"`
	Thought  string      `json:"thought,omitempty"`
	Content  interface{} `json:"content,omitempty"`
}

func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	aux := &struct {
		Parts []partJSON `json:"parts"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	m.Parts = make([]Part, len(aux.Parts))
	for i, p := range aux.Parts {
		switch p.Type {
		case "text":
			m.Parts[i] = TextPart(p.Text)
		case "blob":
			m.Parts[i] = BlobPart{MIMEType: p.MIMEType, Data: p.Data}
		case "thought":
			m.Parts[i] = ThoughtPart(p.Thought)
		default:
			return fmt.Errorf("unknown part type: %s", p.Type)
		}
	}
	return nil
}

func (m Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	parts := make([]partJSON, len(m.Parts))
	for i, p := range m.Parts {
		switch v := p.(type) {
		case TextPart:
			parts[i] = partJSON{Type: "text", Text: string(v)}
		case BlobPart:
			parts[i] = partJSON{Type: "blob", MIMEType: v.MIMEType, Data: v.Data}
		case ThoughtPart:
			parts[i] = partJSON{Type: "thought", Thought: string(v)}
		}
	}

	return json.Marshal(&struct {
		Parts []partJSON `json:"parts"`
		*Alias
	}{
		Parts: parts,
		Alias: (*Alias)(&m),
	})
}

type Provider interface {
	GenerateContent(ctx context.Context, model string, messages []Message, options ...Option) (Response, error)
	GenerateContentStream(ctx context.Context, model string, messages []Message, options ...Option) (StreamResponse, error)
}

type ContextCache struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name,omitempty"`
	Model       string    `json:"model"`
	ExpireTime  time.Time `json:"expire_time"`
}

type ProviderContextCacher interface {
	CreateCache(ctx context.Context, model string, messages []Message, displayName string, ttl time.Duration) (*ContextCache, error)
	GetCache(ctx context.Context, name string) (*ContextCache, error)
	DeleteCache(ctx context.Context, name string) error
	ListCaches(ctx context.Context) ([]*ContextCache, error)
}
