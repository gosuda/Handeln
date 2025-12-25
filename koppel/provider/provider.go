package provider

import (
	"context"
	"io"
)

type Options struct {
}

type Option func(*Options) error

type Part interface {
	IsPart()
}

type TextPart string

func (TextPart) IsPart() {}

type BlobPart struct {
	MIMEType string
	Data     []byte
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
	Role  string
	Parts []Part
}

type Provider interface {
	GenerateContent(ctx context.Context, model string, messages []Message, options ...Option) (Response, error)
	GenerateContentStream(ctx context.Context, model string, messages []Message, options ...Option) (StreamResponse, error)
}
