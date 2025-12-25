package chat

import (
	"encoding/json"
	"reflect"
	"testing"

	"gosuda.org/koppel/provider"
)

func TestSession_Serialization(t *testing.T) {
	s := NewSession("gemini-1.5-pro")
	s.History = []provider.Message{
		{
			Role: "user",
			Parts: []provider.Part{
				provider.TextPart("hello"),
				provider.BlobPart{MIMEType: "image/png", Data: []byte("fake-data")},
			},
		},
		{
			Role: "model",
			Parts: []provider.Part{
				provider.ThoughtPart("thinking..."),
				provider.TextPart("hi there!"),
			},
		},
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("failed to marshal session: %v", err)
	}

	s2 := &Session{}
	if err := json.Unmarshal(data, s2); err != nil {
		t.Fatalf("failed to unmarshal session: %v", err)
	}

	if s.Model != s2.Model {
		t.Errorf("model mismatch: %s != %s", s.Model, s2.Model)
	}

	if len(s.History) != len(s2.History) {
		t.Fatalf("history length mismatch: %d != %d", len(s.History), len(s2.History))
	}

	for i := range s.History {
		msg1, msg2 := s.History[i], s2.History[i]
		if msg1.Role != msg2.Role {
			t.Errorf("role mismatch at index %d: %s != %s", i, msg1.Role, msg2.Role)
		}
		if len(msg1.Parts) != len(msg2.Parts) {
			t.Errorf("parts length mismatch at index %d: %d != %d", i, len(msg1.Parts), len(msg2.Parts))
			continue
		}
		for j := range msg1.Parts {
			if !reflect.DeepEqual(msg1.Parts[j], msg2.Parts[j]) {
				t.Errorf("part mismatch at message %d, part %d: %T != %T", i, j, msg1.Parts[j], msg2.Parts[j])
			}
		}
	}
}

func TestOptions_Serialization(t *testing.T) {
	opts := &provider.Options{
		CacheName: "cached-context-123",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal options: %v", err)
	}

	opts2 := &provider.Options{}
	if err := json.Unmarshal(data, opts2); err != nil {
		t.Fatalf("failed to unmarshal options: %v", err)
	}

	if opts.CacheName != opts2.CacheName {
		t.Errorf("cache name mismatch: %s != %s", opts.CacheName, opts2.CacheName)
	}
}
