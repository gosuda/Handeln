package openai

import (
	"testing"

	"gosuda.org/koppel/provider"
)

func TestOpenAIProvider_Interface(t *testing.T) {
	var _ provider.Provider = (*OpenAIProvider)(nil)
}

func TestToChatParams(t *testing.T) {
	p := &OpenAIProvider{}
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

	params := p.toChatParams("gpt-4o", messages)
	if params.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", params.Model)
	}

	if len(params.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(params.Messages))
	}

	// Verify System message
	sysMsg := params.Messages[0]
	if sysMsg.OfSystem == nil {
		t.Fatal("expected system message")
	}

	// Verify User message with multimodal parts
	userMsg := params.Messages[1].OfUser
	if userMsg == nil {
		t.Fatal("expected user message")
	}

	content := userMsg.Content
	if content.OfArrayOfContentParts == nil {
		t.Fatalf("expected array content, got %+v", content)
	}

	parts := content.OfArrayOfContentParts
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}

	if parts[0].OfText == nil || parts[0].OfText.Text != "hello" {
		t.Errorf("expected text 'hello', got %+v", parts[0].OfText)
	}

	if parts[1].OfImageURL == nil {
		t.Fatal("expected image part")
	}
	if parts[1].OfImageURL.ImageURL.URL == "" {
		t.Error("expected non-empty image URL")
	}
}
