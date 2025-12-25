package anthropic

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"gosuda.org/koppel/provider"
)

func TestAnthropicProvider_Interface(t *testing.T) {
	var _ provider.Provider = (*AnthropicProvider)(nil)
}

func TestToMessageParams(t *testing.T) {
	p := &AnthropicProvider{}
	messages := []provider.Message{
		{
			Role: "system",
			Parts: []provider.Part{
				provider.TextPart("you are a helpful assistant"),
			},
		},
		{
			Role: "user",
			Parts: []provider.Part{
				provider.TextPart("hello"),
				provider.BlobPart{MIMEType: "image/png", Data: []byte("fake-image")},
			},
		},
	}

	params := p.toMessageParams("claude-3-5-sonnet-20240620", messages)
	if params.Model != "claude-3-5-sonnet-20240620" {
		t.Errorf("expected model claude-3-5-sonnet-20240620, got %s", params.Model)
	}

	if len(params.System) != 1 {
		t.Fatalf("expected 1 system block, got %d", len(params.System))
	}

	if len(params.Messages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(params.Messages))
	}

	userMsg := params.Messages[0]
	if userMsg.Role != anthropic.MessageParamRoleUser {
		t.Errorf("expected role user, got %s", userMsg.Role)
	}

	blocks := userMsg.Content
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
}
