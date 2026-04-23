package bus

import (
	"testing"
	"time"
)

func TestFileBus_UnsubscribeAfterClose(t *testing.T) {
	dir := t.TempDir()
	b, err := NewFileBus(dir, 50*time.Millisecond, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.Subscribe("a")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	// Should not panic or deadlock
	if err := b.Unsubscribe("a"); err != nil {
		t.Fatal(err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFileBus_SubscribeAfterClose(t *testing.T) {
	dir := t.TempDir()
	b, err := NewFileBus(dir, 50*time.Millisecond, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	b.Close()
	_, err = b.Subscribe("a")
	if err == nil {
		t.Error("Subscribe after Close should fail")
	}
}

func TestChannelBus_DoubleClose(t *testing.T) {
	b := NewChannelBus()
	b.Subscribe("a")
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	// Second Close should not panic
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
}
