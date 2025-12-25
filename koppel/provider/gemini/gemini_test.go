package gemini

import (
	"testing"

	"gosuda.org/koppel/provider"
)

func TestGeminiProvider_Interface(t *testing.T) {
	var _ provider.Provider = (*GeminiProvider)(nil)
}

func TestToGenAIContents(t *testing.T) {
	p := &GeminiProvider{}
	messages := []provider.Message{
		{
			Role: "user",
			Parts: []provider.Part{
				provider.TextPart("hello"),
				provider.BlobPart{MIMEType: "image/png", Data: []byte("fake-data")},
			},
		},
	}

	contents := p.toGenAIContents(messages)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}
	if contents[0].Role != "user" {
		t.Errorf("expected role user, got %s", contents[0].Role)
	}
	if len(contents[0].Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(contents[0].Parts))
	}
	if contents[0].Parts[0].Text != "hello" {
		t.Errorf("expected text 'hello', got %s", contents[0].Parts[0].Text)
	}
	if contents[0].Parts[1].InlineData.MIMEType != "image/png" {
		t.Errorf("expected mimetype image/png, got %s", contents[0].Parts[1].InlineData.MIMEType)
	}
}
