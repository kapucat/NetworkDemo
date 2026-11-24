package main

import (
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDiagnosticMarkDoesNotChangeWireJSON(t *testing.T) {
	m := WebSocketMessage{Type: "player.move", AccountID: 7, X: 1}
	before, _ := json.Marshal(m)
	m.diagQueuedAt = time.Now()
	after, _ := json.Marshal(m)
	if string(before) != string(after) || strings.Contains(string(after), "diag") {
		t.Fatalf("wire message changed: %s", after)
	}
}
func TestCountersKeepQueueAndWritePeaksSeparate(t *testing.T) {
	var c wsDiagnosticCounters
	at := time.Now()
	c.observeQueue(75*time.Millisecond, 200, at)
	c.observeWrite(2*time.Millisecond, false, at)
	c.observeQueue(time.Millisecond, 4, at)
	c.observeWrite(40*time.Millisecond, true, at)
	s := c.snapshot()
	if s.Samples != 2 || s.Errors != 1 || s.QueueMaxMS != 75 || s.WriteMaxMS != 40 || s.DepthMax != 200 || s.QueueSumMS != 76 || s.WriteSumMS != 42 {
		t.Fatalf("unexpected snapshot: %+v", s)
	}
}
func TestConcurrentMaximum(t *testing.T) {
	var peak atomic.Int64
	var wg sync.WaitGroup
	for i := int64(1); i <= 100; i++ {
		wg.Add(1)
		go func(n int64) { defer wg.Done(); wsAtomicMax(&peak, n) }(i)
	}
	wg.Wait()
	if peak.Load() != 100 {
		t.Fatalf("max=%d", peak.Load())
	}
}
