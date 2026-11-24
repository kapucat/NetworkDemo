package main

import (
	"sync/atomic"
	"time"
)

type wsSlowWriteEvent struct {
	AccountID   int64   `json:"account_id"`
	Kind        string  `json:"write_kind"`
	StartedAt   string  `json:"started_at"`
	CompletedAt string  `json:"completed_at"`
	DurationMS  float64 `json:"duration_ms"`
	Failed      bool    `json:"failed"`
}

var wsSlowWrites = make(chan wsSlowWriteEvent, 1024)
var wsSlowWritesDropped atomic.Uint64

type wsAllWriteCounters struct {
	data, control, errors, over10, over50, over100 atomic.Uint64
	maxNS, maxStart, maxEnd                        atomic.Int64
}
type wsAllWriteSnapshot struct {
	Data     uint64  `json:"data_calls"`
	Control  uint64  `json:"control_calls"`
	Errors   uint64  `json:"error_calls"`
	Over10   uint64  `json:"ge_10ms"`
	Over50   uint64  `json:"ge_50ms"`
	Over100  uint64  `json:"ge_100ms"`
	MaxMS    float64 `json:"max_ms"`
	MaxStart string  `json:"max_started_at,omitempty"`
	MaxEnd   string  `json:"max_completed_at,omitempty"`
}

func (s *wsAllWriteCounters) observe(d time.Duration, failed bool, kind string, start, end time.Time, accountID int64) {
	if kind == "data" {
		s.data.Add(1)
	} else {
		s.control.Add(1)
	}
	if failed {
		s.errors.Add(1)
	}
	if d >= 10*time.Millisecond {
		s.over10.Add(1)
	}
	if d >= 50*time.Millisecond {
		s.over50.Add(1)
	}
	if d >= 100*time.Millisecond {
		s.over100.Add(1)
	}
	if wsAtomicMax(&s.maxNS, d.Nanoseconds()) {
		s.maxStart.Store(start.UnixNano())
		s.maxEnd.Store(end.UnixNano())
	}
	if d >= 50*time.Millisecond {
		e := wsSlowWriteEvent{accountID, kind, start.UTC().Format(time.RFC3339Nano), end.UTC().Format(time.RFC3339Nano), float64(d.Nanoseconds()) / 1e6, failed}
		select {
		case wsSlowWrites <- e:
		default:
			wsSlowWritesDropped.Add(1)
		}
	}
}
func (s *wsAllWriteCounters) snapshot() wsAllWriteSnapshot {
	return wsAllWriteSnapshot{s.data.Load(), s.control.Load(), s.errors.Load(), s.over10.Load(), s.over50.Load(), s.over100.Load(), float64(s.maxNS.Load()) / 1e6, wsDiagnosticTime(s.maxStart.Load()), wsDiagnosticTime(s.maxEnd.Load())}
}
func wsMergeAllWrites(dst *wsAllWriteSnapshot, src wsAllWriteSnapshot) {
	dst.Data += src.Data
	dst.Control += src.Control
	dst.Errors += src.Errors
	dst.Over10 += src.Over10
	dst.Over50 += src.Over50
	dst.Over100 += src.Over100
	if src.MaxMS > dst.MaxMS {
		dst.MaxMS = src.MaxMS
		dst.MaxStart = src.MaxStart
		dst.MaxEnd = src.MaxEnd
	}
}
