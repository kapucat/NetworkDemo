package main

import (
	"testing"
	"time"
)

func TestAllWritesThresholdsAndControlClassification(t *testing.T) {
	var c wsAllWriteCounters
	start := time.Now()
	for _, d := range []time.Duration{9 * time.Millisecond, 10 * time.Millisecond, 49 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond} {
		c.observe(d, false, "data", start, start.Add(d), 1)
	}
	c.observe(time.Millisecond, true, "ping", start, start.Add(time.Millisecond), 1)
	s := c.snapshot()
	if s.Data != 5 || s.Control != 1 || s.Errors != 1 || s.Over10 != 4 || s.Over50 != 2 || s.Over100 != 1 || s.MaxMS != 100 {
		t.Fatalf("unexpected: %+v", s)
	}
}
func TestSlowEventBufferNeverBlocks(t *testing.T) {
	for len(wsSlowWrites) > 0 {
		<-wsSlowWrites
	}
	wsSlowWritesDropped.Store(0)
	var c wsAllWriteCounters
	at := time.Now()
	for i := 0; i < cap(wsSlowWrites)+3; i++ {
		c.observe(60*time.Millisecond, false, "data", at, at.Add(60*time.Millisecond), 7)
	}
	if wsSlowWritesDropped.Load() != 3 {
		t.Fatalf("dropped=%d", wsSlowWritesDropped.Load())
	}
	event := <-wsSlowWrites
	if event.AccountID != 7 || event.DurationMS != 60 || event.Kind != "data" {
		t.Fatalf("event=%+v", event)
	}
	for len(wsSlowWrites) > 0 {
		<-wsSlowWrites
	}
}
