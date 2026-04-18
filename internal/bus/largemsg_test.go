package bus

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFileBus_LargeMessage(t *testing.T) {
	dir := t.TempDir()
	b, err := NewFileBus(dir, 50*time.Millisecond, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	ch, err := b.Subscribe("large")
	if err != nil {
		t.Fatal(err)
	}

	// 80KB payload -- well over bufio.Scanner default (64KB)
	big := strings.Repeat("x", 80*1024)
	payload, _ := json.Marshal(big)
	if err := b.Publish("large", Message{Type: "blob", Payload: payload}); err != nil {
		t.Fatal(err)
	}

	select {
	case env := <-ch:
		var got string
		if err := json.Unmarshal(env.Payload, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got != big {
			t.Errorf("content mismatch: got %d bytes, want %d", len(got), len(big))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for large message")
	}
}
