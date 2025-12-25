package chat

import (
	"context"
	"fmt"
	"testing"

	"gosuda.org/koppel/provider"
)

type mockProvider struct {
	lastMessages []provider.Message
}

func (m *mockProvider) GenerateContent(ctx context.Context, model string, messages []provider.Message, options ...provider.Option) (provider.Response, error) {
	m.lastMessages = messages
	return &mockResponse{text: "mock response"}, nil
}

func (m *mockProvider) GenerateContentStream(ctx context.Context, model string, messages []provider.Message, options ...provider.Option) (provider.StreamResponse, error) {
	m.lastMessages = messages
	return &mockStreamResponse{text: "mock response"}, nil
}

type mockResponse struct {
	text string
}

func (r *mockResponse) Text() string {
	return r.text
}

func (r *mockResponse) Thought() string {
	return ""
}

// Added mockStreamResponse for streaming tests
type mockStreamResponse struct {
	text string
	sent bool
}

func (s *mockStreamResponse) Next() (provider.Response, error) {
	if s.sent {
		return nil, fmt.Errorf("no more stream items")
	}
	s.sent = true
	return &mockResponse{text: s.text}, nil
}

func (s *mockStreamResponse) Close() error {
	return nil
}

func TestSession_Send(t *testing.T) {
	mock := &mockProvider{}
	s := NewSession("test-model")
	s.SetProvider(mock)

	ctx := context.Background()
	_, err := s.Send(ctx, provider.TextPart("hello"))
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if len(s.History) != 2 {
		t.Errorf("expected 2 messages in history, got %d", len(s.History))
	}

	if s.History[0].Role != "user" || s.History[1].Role != "model" {
		t.Errorf("unexpected roles in history: %s, %s", s.History[0].Role, s.History[1].Role)
	}

	if mock.lastMessages[0].Role != "user" {
		t.Errorf("mock received wrong history")
	}
}

func TestSession_SendStream(t *testing.T) {
	mock := &mockProvider{}
	s := NewSession("test-model")
	s.SetProvider(mock)

	ctx := context.Background()
	stream, err := s.SendStream(ctx, provider.TextPart("hello stream"))
	if err != nil {
		t.Fatalf("SendStream failed: %v", err)
	}

	var text string
	for {
		resp, err := stream.Next()
		if err != nil {
			if err.Error() == "no more stream items" {
				break
			}
			t.Fatalf("stream.Next() failed: %v", err)
		}
		text += resp.Text()
	}

	if text != "mock response" {
		t.Errorf("expected 'mock response', got %s", text)
	}

	if len(s.History) != 2 {
		t.Errorf("expected 2 messages in history, got %d", len(s.History))
	}
}
