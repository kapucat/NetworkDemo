package main

import (
	"sync"
	"testing"
	"time"
)

func testClient() *Client {
	return &Client{send: make(chan WebSocketMessage, 256), moveWake: make(chan struct{}, 1)}
}
func TestLatestMoveBoundAndPriority(t *testing.T) {
	c := testClient()
	for i := 0; i < 10000; i++ {
		c.enqueueMove(WebSocketMessage{Type: "player.move", AccountID: 1, X: float64(i)})
	}
	if c.moveCount() != 1 {
		t.Fatal("unbounded moves")
	}
	c.sendSafely(WebSocketMessage{Type: "player.leave", AccountID: 2})
	m, k := c.nextDelivery(make(chan time.Time))
	if k != "data" || m.Type != "player.leave" {
		t.Fatal("control not prioritized")
	}
	m, _ = c.takeMove()
	if m.X != 9999 {
		t.Fatal("latest not retained")
	}
	for i := int64(1); i <= 300; i++ {
		c.enqueueMove(WebSocketMessage{AccountID: i})
	}
	if c.moveCount() != 256 {
		t.Fatal("capacity")
	}
	if c.movesEvicted.Load() != 44 {
		t.Fatal("eviction")
	}
}
func TestLeavePurgesPendingAndCloseRejects(t *testing.T) {
	c := testClient()
	c.enqueueMove(WebSocketMessage{AccountID: 1})
	c.discardMove(1)
	if _, ok := c.takeMove(); ok {
		t.Fatal("stale move")
	}
	c.closeSend()
	if c.enqueueMove(WebSocketMessage{AccountID: 1}) {
		t.Fatal("closed accepted")
	}
	_, kind := c.nextDelivery(make(chan time.Time))
	if kind != "close" {
		t.Fatal(kind)
	}
}
func TestLeaveWavePreservesControls(t *testing.T) {
	c := testClient()
	for i := int64(1); i <= 200; i++ {
		c.enqueueMove(WebSocketMessage{AccountID: i})
		c.discardMove(i)
		if !c.sendSafely(WebSocketMessage{Type: "player.leave", AccountID: i}) {
			t.Fatal("control lost")
		}
	}
	for i := int64(1); i <= 200; i++ {
		m, _ := c.nextDelivery(make(chan time.Time))
		if m.Type != "player.leave" || m.AccountID != i {
			t.Fatal("control order")
		}
	}
	if c.moveCount() != 0 {
		t.Fatal("moves resurrected")
	}
}
func TestConcurrentMoveAccess(t *testing.T) {
	c := testClient()
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 5000; i++ {
				c.enqueueMove(WebSocketMessage{AccountID: int64(i % 200)})
				c.takeMove()
				c.discardMove(int64(i % 200))
			}
		}()
	}
	wg.Wait()
	if c.moveCount() > 256 {
		t.Fatal("capacity")
	}
}
func TestMovementBudgetKeepsControlsResponsive(t *testing.T) {
	c := testClient()
	refill := make(chan time.Time, 1)
	ping := make(chan time.Time)
	refill <- time.Now()
	for i := int64(1); i <= 7; i++ {
		c.enqueueMove(WebSocketMessage{Type: "player.move", AccountID: i})
	}
	for i := int64(1); i <= 6; i++ {
		m, _ := c.nextDelivery(ping, refill)
		if m.AccountID != i {
			t.Fatal("fair order")
		}
	}
	done := make(chan string, 1)
	go func() { m, _ := c.nextDelivery(ping, refill); done <- m.Type }()
	c.sendSafely(WebSocketMessage{Type: "player.leave", AccountID: 8})
	select {
	case kind := <-done:
		if kind != "player.leave" {
			t.Fatal(kind)
		}
	case <-time.After(time.Second):
		t.Fatal("control blocked by move budget")
	}
	refill <- time.Now()
	m, _ := c.nextDelivery(ping, refill)
	if m.AccountID != 7 {
		t.Fatal("pending latest lost")
	}
}
