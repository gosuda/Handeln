package chat

import (
	"context"

	"gosuda.org/koppel/provider"
)

type Session struct {
	provider provider.Provider  `json:"-"`
	Model    string             `json:"model"`
	History  []provider.Message `json:"history"`
}

func NewSession(model string) *Session {
	return &Session{
		Model: model,
	}
}

func (s *Session) SetProvider(p provider.Provider) {
	s.provider = p
}

func (s *Session) Send(ctx context.Context, parts ...provider.Part) (provider.Response, error) {
	msg := provider.Message{
		Role:  "user",
		Parts: parts,
	}
	s.History = append(s.History, msg)

	resp, err := s.provider.GenerateContent(ctx, s.Model, s.History)
	if err != nil {
		return nil, err
	}

	modelMsg := provider.Message{
		Role: "model",
	}
	if thought := resp.Thought(); thought != "" {
		modelMsg.Parts = append(modelMsg.Parts, provider.ThoughtPart(thought))
	}
	modelMsg.Parts = append(modelMsg.Parts, provider.TextPart(resp.Text()))
	s.History = append(s.History, modelMsg)

	return resp, nil
}

func (s *Session) SendStream(ctx context.Context, parts ...provider.Part) (provider.StreamResponse, error) {
	msg := provider.Message{
		Role:  "user",
		Parts: parts,
	}
	s.History = append(s.History, msg)

	stream, err := s.provider.GenerateContentStream(ctx, s.Model, s.History)
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
	thought string
}

func (r *chatStreamResponse) Next() (provider.Response, error) {
	resp, err := r.stream.Next()
	if err != nil {
		if err.Error() == "no more stream items" {
			// End of stream, save to history
			modelMsg := provider.Message{
				Role: "model",
			}
			if r.thought != "" {
				modelMsg.Parts = append(modelMsg.Parts, provider.ThoughtPart(r.thought))
			}
			modelMsg.Parts = append(modelMsg.Parts, provider.TextPart(r.text))
			r.session.History = append(r.session.History, modelMsg)
		}
		return nil, err
	}
	r.text += resp.Text()
	r.thought += resp.Thought()
	return resp, nil
}

func (r *chatStreamResponse) Close() error {
	return r.stream.Close()
}

func (r *chatStreamResponse) Thought() string {
	return r.thought
}
