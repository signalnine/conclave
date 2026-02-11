package bus

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
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

func TestChannelBusPublishSubscribe(t *testing.T) {
	bus := NewChannelBus()
	defer bus.Close()

	ch, err := bus.Subscribe("test.topic")
	if err != nil {
		t.Fatal(err)
	}

	msg := Message{Type: "greeting", Sender: "alice", Payload: json.RawMessage(`{"hi":true}`)}
	if err := bus.Publish("test.topic", msg); err != nil {
		t.Fatal(err)
	}

	select {
	case env := <-ch:
		if env.Type != "greeting" {
			t.Errorf("type = %q, want greeting", env.Type)
		}
		if env.Sender != "alice" {
			t.Errorf("sender = %q, want alice", env.Sender)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestChannelBusFanOut(t *testing.T) {
	bus := NewChannelBus()
	defer bus.Close()

	ch1, _ := bus.Subscribe("topic")
	ch2, _ := bus.Subscribe("topic")

	msg := Message{Type: "broadcast", Sender: "bob", Payload: json.RawMessage(`{}`)}
	bus.Publish("topic", msg)

	for i, ch := range []<-chan Envelope{ch1, ch2} {
		select {
		case env := <-ch:
			if env.Type != "broadcast" {
				t.Errorf("subscriber %d: type = %q", i, env.Type)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timeout", i)
		}
	}
}

func TestChannelBusPrefixSubscription(t *testing.T) {
	bus := NewChannelBus()
	defer bus.Close()

	ch, _ := bus.Subscribe("consensus")

	bus.Publish("consensus.s1.debate", Message{Type: "finding", Sender: "a", Payload: json.RawMessage(`{}`)})
	bus.Publish("parallel.wave-0", Message{Type: "other", Sender: "b", Payload: json.RawMessage(`{}`)})

	select {
	case env := <-ch:
		if env.Type != "finding" {
			t.Errorf("type = %q, want finding", env.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	// Second message should NOT arrive (different prefix)
	select {
	case env := <-ch:
		t.Errorf("unexpected message: %+v", env)
	case <-time.After(50 * time.Millisecond):
		// Good — no message
	}
}

func TestChannelBusBackpressureDrop(t *testing.T) {
	bus := NewChannelBus()
	defer bus.Close()

	ch, _ := bus.Subscribe("flood")

	// Fill the buffer (capacity 64) plus overflow
	for i := 0; i < 70; i++ {
		bus.Publish("flood", Message{Type: "msg", Sender: "s", Payload: json.RawMessage(`{}`)})
	}

	// Should get exactly 64 messages (buffer capacity)
	count := 0
	for {
		select {
		case <-ch:
			count++
		case <-time.After(50 * time.Millisecond):
			goto done
		}
	}
done:
	if count != 64 {
		t.Errorf("received %d messages, want 64 (buffer capacity)", count)
	}
}

func TestChannelBusClose(t *testing.T) {
	bus := NewChannelBus()
	ch, _ := bus.Subscribe("topic")
	bus.Close()

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after bus.Close()")
	}
}

func TestChannelBusUnsubscribe(t *testing.T) {
	bus := NewChannelBus()
	defer bus.Close()

	ch, _ := bus.Subscribe("topic")
	bus.Unsubscribe("topic")

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after Unsubscribe()")
	}

	// Publish should not panic
	bus.Publish("topic", Message{Type: "msg", Sender: "s", Payload: json.RawMessage(`{}`)})
}

func TestChannelBusConcurrentPublish(t *testing.T) {
	bus := NewChannelBus()
	defer bus.Close()

	ch, _ := bus.Subscribe("race")

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func(n int) {
			bus.Publish("race", Message{
				Type:    "test",
				Sender:  fmt.Sprintf("g-%d", n),
				Payload: json.RawMessage(`{}`),
			})
			done <- struct{}{}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Drain and count
	count := 0
	for {
		select {
		case <-ch:
			count++
		case <-time.After(100 * time.Millisecond):
			goto finished
		}
	}
finished:
	if count == 0 {
		t.Error("expected some messages from concurrent publishers")
	}
	t.Logf("received %d of 50 messages (some may be dropped due to backpressure)", count)
}
