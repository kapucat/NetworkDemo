package main

import (
	"github.com/gorilla/websocket"
	"log"
	"time"
)

func (c *Client) writePump() {
	if wsDiagnosticsEnabled {
		defer func() { wsLogClientDiagnostic(c); wsDiagnosticClients.Delete(c) }()
	}
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	refill := time.NewTicker(10 * time.Millisecond)
	defer refill.Stop()
	for {
		m, kind := c.nextDelivery(ticker.C, refill.C)
		c.connection.SetWriteDeadline(time.Now().Add(writeWait))
		if !m.diagQueuedAt.IsZero() {
			at := time.Now()
			c.diagStats.observeQueue(at.Sub(m.diagQueuedAt), c.moveCount(), at)
		}
		var start time.Time
		if wsDiagnosticsEnabled {
			start = time.Now()
		}
		var err error
		switch kind {
		case "ping":
			err = c.connection.WriteMessage(websocket.PingMessage, nil)
		case "close":
			err = c.connection.WriteMessage(websocket.CloseMessage, nil)
		default:
			if len(m.encoded) > 0 {
				err = c.connection.WriteMessage(websocket.TextMessage, m.encoded)
			} else {
				err = c.connection.WriteJSON(m)
			}
		}
		if wsDiagnosticsEnabled {
			end := time.Now()
			d := end.Sub(start)
			c.diagStats.all.observe(d, err != nil, kind, start, end, c.accountID)
			if !m.diagQueuedAt.IsZero() {
				c.diagStats.observeWrite(d, err != nil, end)
			}
		}
		if kind == "close" {
			return
		}
		if err != nil {
			log.Printf("websocket write failed: %v", err)
			c.connection.Close()
			return
		}
	}
}
