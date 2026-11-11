package singleinstance

import (
	"context"
	"testing"
	"time"
)

func TestActivationQueuedBeforeWindowReady(t *testing.T) {
	dir := t.TempDir()
	primary, err := acquire(dir)
	if err != nil || primary == nil {
		t.Fatalf("first launch: primary=%v, err=%v", primary, err)
	}
	defer primary.Close()
	for range 2 {
		if second, err := acquire(dir); err != nil || second != nil {
			if second != nil {
				second.Close()
			}
			t.Fatalf("duplicate launch: primary=%v, err=%v", second, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	activated := false
	primary.Watch(ctx, func() { activated = true; cancel() })
	if !activated {
		t.Fatal("activation sent before window readiness was lost")
	}
}
