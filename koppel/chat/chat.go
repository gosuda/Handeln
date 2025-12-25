package chat

import (
	"context"

	"gosuda.org/koppel/provider"
)

type Session struct {
	provider provider.Provider
	model    string
	history  []provider.Message
}

func NewSession(p provider.Provider, model string) *Session {
	return &Session{
		provider: p,
		model:    model,
	}
}

func (s *Session) Send(ctx context.Context, parts ...provider.Part) (provider.Response, error) {
	msg := provider.Message{
		Role:  "user",
		Parts: parts,
	}
	s.history = append(s.history, msg)

	resp, err := s.provider.GenerateContent(ctx, s.model, s.history)
	if err != nil {
		return nil, err
	}

	modelMsg := provider.Message{
		Role:  "model",
		Parts: []provider.Part{provider.TextPart(resp.Text())},
	}
	s.history = append(s.history, modelMsg)

	return resp, nil
}

func (s *Session) SendStream(ctx context.Context, parts ...provider.Part) (provider.StreamResponse, error) {
	msg := provider.Message{
		Role:  "user",
		Parts: parts,
	}
	s.history = append(s.history, msg)

	stream, err := s.provider.GenerateContentStream(ctx, s.model, s.history)
	if err != nil {
		return nil, err
	}

	return &chatStreamResponse{
		session: s,
		stream:  stream,
	}, nil
}

type chatStreamResponse struct {
	session *Session
	stream  provider.StreamResponse
	text    string
}

func (r *chatStreamResponse) Next() (provider.Response, error) {
	resp, err := r.stream.Next()
	if err != nil {
		if err.Error() == "no more stream items" {
			// End of stream, save to history
			modelMsg := provider.Message{
				Role:  "model",
				Parts: []provider.Part{provider.TextPart(r.text)},
			}
			r.session.history = append(r.session.history, modelMsg)
		}
		return nil, err
	}
	r.text += resp.Text()
	return resp, nil
}

func (r *chatStreamResponse) Close() error {
	return r.stream.Close()
}

func (r *chatStreamResponse) Text() string {
	return r.text
}
