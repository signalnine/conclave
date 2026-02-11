package bus

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewEnvelope(t *testing.T) {
	msg := Message{
		Type:    "board.discovery",
		Sender:  "task-3",
		Payload: json.RawMessage(`{"text":"found something"}`),
	}
	env := NewEnvelope("test-topic", msg)

	if env.Topic != "test-topic" {
		t.Errorf("topic = %q, want test-topic", env.Topic)
	}
	if env.Type != "board.discovery" {
		t.Errorf("type = %q, want board.discovery", env.Type)
	}
	if env.Sender != "task-3" {
		t.Errorf("sender = %q, want task-3", env.Sender)
	}
	if env.Seq == 0 {
		t.Error("seq should be > 0")
	}
	if env.ID == "" {
		t.Error("id should not be empty")
	}
	if env.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestIDsAreUnique(t *testing.T) {
	msg := Message{Type: "test", Sender: "a", Payload: json.RawMessage(`{}`)}
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		env := NewEnvelope("t", msg)
		if ids[env.ID] {
			t.Fatalf("duplicate ID: %s", env.ID)
		}
		ids[env.ID] = true
	}
}

func TestIDContainsPID(t *testing.T) {
	msg := Message{Type: "test", Sender: "a", Payload: json.RawMessage(`{}`)}
	env := NewEnvelope("t", msg)
	if !strings.Contains(env.ID, "-") {
		t.Errorf("ID %q should contain PID prefix and dash", env.ID)
	}
}

func TestSequenceMonotonic(t *testing.T) {
	msg := Message{Type: "test", Sender: "a", Payload: json.RawMessage(`{}`)}
	var lastSeq uint64
	for i := 0; i < 10; i++ {
		env := NewEnvelope("t", msg)
		if env.Seq <= lastSeq {
			t.Errorf("seq %d not greater than previous %d", env.Seq, lastSeq)
		}
		lastSeq = env.Seq
	}
}

func TestEnvelopeJSON(t *testing.T) {
	msg := Message{
		Type:    "debate.rebuttal",
		Sender:  "claude",
		Payload: json.RawMessage(`{"position":"disagree"}`),
	}
	env := NewEnvelope("consensus.s1.debate", msg)

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Topic != env.Topic {
		t.Errorf("topic = %q, want %q", decoded.Topic, env.Topic)
	}
	if decoded.Type != env.Type {
		t.Errorf("type = %q, want %q", decoded.Type, env.Type)
	}
	if string(decoded.Payload) != string(env.Payload) {
		t.Errorf("payload = %q, want %q", decoded.Payload, env.Payload)
	}
}

func TestTopicMatch(t *testing.T) {
	tests := []struct {
		pattern string
		topic   string
		want    bool
	}{
		{"consensus", "consensus", true},
		{"consensus", "consensus.s1.debate", true},
		{"consensus.s1", "consensus.s1.debate", true},
		{"consensus.s2", "consensus.s1.debate", false},
		{"parallel.wave-0", "parallel.wave-0.board", true},
		{"parallel.wave-1", "parallel.wave-0.board", false},
		{"", "anything", true},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.topic, func(t *testing.T) {
			got := TopicMatch(tt.pattern, tt.topic)
			if got != tt.want {
				t.Errorf("TopicMatch(%q, %q) = %v, want %v", tt.pattern, tt.topic, got, tt.want)
			}
		})
	}
}
