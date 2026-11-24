package main

import (
	"container/list"
	"time"
)

const pendingMoveLimit = 256

// Control messages retain FIFO order in send. Moves use a bounded keyed FIFO.
// All move-list access is protected by moveMu; each connection has one writer.
func (c *Client) enqueueMove(m WebSocketMessage) bool {
	c.moveMu.Lock()
	defer c.moveMu.Unlock()
	if c.movesClosed {
		return false
	}
	if c.moves == nil {
		c.moves = make(map[int64]*list.Element)
	}
	if e := c.moves[m.AccountID]; e != nil {
		e.Value = m
		c.movesReplaced.Add(1)
	} else {
		if len(c.moves) >= pendingMoveLimit {
			old := c.moveOrder.Front()
			delete(c.moves, old.Value.(WebSocketMessage).AccountID)
			c.moveOrder.Remove(old)
			c.movesEvicted.Add(1)
		}
		c.moves[m.AccountID] = c.moveOrder.PushBack(m)
	}
	select {
	case c.moveWake <- struct{}{}:
	default:
	}
	return true
}
func (c *Client) takeMove() (WebSocketMessage, bool) {
	c.moveMu.Lock()
	defer c.moveMu.Unlock()
	e := c.moveOrder.Front()
	if e == nil {
		return WebSocketMessage{}, false
	}
	m := e.Value.(WebSocketMessage)
	c.moveOrder.Remove(e)
	delete(c.moves, m.AccountID)
	return m, true
}
func (c *Client) discardMove(id int64) {
	c.moveMu.Lock()
	defer c.moveMu.Unlock()
	if e := c.moves[id]; e != nil {
		c.moveOrder.Remove(e)
		delete(c.moves, id)
	}
}
func (c *Client) closeMoves() {
	c.moveMu.Lock()
	defer c.moveMu.Unlock()
	c.movesClosed = true
	c.moves = nil
	c.moveOrder.Init()
	select {
	case c.moveWake <- struct{}{}:
	default:
	}
}
func (c *Client) moveCount() int { c.moveMu.Lock(); defer c.moveMu.Unlock(); return len(c.moves) }

// Controls have priority between writes. A write already in progress is not interrupted.
func (c *Client) nextDelivery(ping <-chan time.Time, refills ...<-chan time.Time) (WebSocketMessage, string) {
	var refill <-chan time.Time
	if len(refills) > 0 {
		refill = refills[0]
	}
	for {
		select {
		case m, ok := <-c.send:
			if !ok {
				return m, "close"
			}
			return m, "data"
		default:
		}
		select {
		case <-ping:
			return WebSocketMessage{}, "ping"
		default:
		}
		if refill != nil {
			select {
			case <-refill:
				c.moveSlots = 6
			default:
			}
		}
		if refill == nil || c.moveSlots > 0 {
			if m, ok := c.takeMove(); ok {
				if refill != nil {
					c.moveSlots--
				}
				return m, "data"
			}
		}
		wake := c.moveWake
		if refill != nil && c.moveSlots == 0 {
			wake = nil
		}
		select {
		case m, ok := <-c.send:
			if !ok {
				return m, "close"
			}
			return m, "data"
		case <-ping:
			return WebSocketMessage{}, "ping"
		case <-refill:
			c.moveSlots = 6
		case <-wake:
		}
	}
}
