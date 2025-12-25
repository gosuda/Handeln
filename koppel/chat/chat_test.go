package chat

import (
	"context"
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
	return nil, nil
}

type mockResponse struct {
	text string
}

func (r *mockResponse) Text() string {
	return r.text
}

func TestSession_Send(t *testing.T) {
	mp := &mockProvider{}
	s := NewSession(mp, "test-model")

	resp, err := s.Send(context.Background(), provider.TextPart("hello"))
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	if resp.Text() != "mock response" {
		t.Errorf("expected 'mock response', got %s", resp.Text())
	}

	if len(s.history) != 2 {
		t.Errorf("expected history length 2, got %d", len(s.history))
	}

	if s.history[0].Role != "user" || s.history[1].Role != "model" {
		t.Errorf("incorrect roles in history")
	}

	// Second turn
	_, _ = s.Send(context.Background(), provider.TextPart("how are you?"))
	if len(mp.lastMessages) != 3 {
		t.Errorf("expected 3 messages in the second turn call, got %d", len(mp.lastMessages))
	}
}
