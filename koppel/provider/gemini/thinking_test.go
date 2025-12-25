package gemini

import (
	"testing"

	"google.golang.org/genai"
	"gosuda.org/koppel/provider"
)

func TestGeminiProvider_Thinking(t *testing.T) {
	p := &GeminiProvider{}
	messages := []provider.Message{
		{
			Role: "user",
			Parts: []provider.Part{
				provider.ThoughtPart("I should say hello"),
				provider.TextPart("hello"),
			},
		},
	}

	contents := p.toGenAIContents(messages)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}
	if len(contents[0].Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(contents[0].Parts))
	}
	if !contents[0].Parts[0].Thought {
		t.Error("expected first part to be a thought")
	}
	if contents[0].Parts[0].Text != "I should say hello" {
		t.Errorf("expected thought text 'I should say hello', got %s", contents[0].Parts[0].Text)
	}
}

func TestGeminiResponse_Thinking(t *testing.T) {
	resp := &geminiResponse{
		resp: &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Thought: true, Text: "Thinking..."},
							{Text: "Hello!"},
						},
					},
				},
			},
		},
	}

	if resp.Text() != "Hello!" {
		t.Errorf("expected text 'Hello!', got %s", resp.Text())
	}
	if resp.Thought() != "Thinking..." {
		t.Errorf("expected thought 'Thinking...', got %s", resp.Thought())
	}
}
