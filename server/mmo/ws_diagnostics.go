package main

import (
	"encoding/json"
	"log"
	"math"
	"os"
	"runtime/metrics"
	"sync"
	"sync/atomic"
	"time"
)

const wsDiagnosticSampleEvery = 64

var wsDiagnosticsEnabled = os.Getenv("MMO_WS_DIAGNOSTICS") == "1"
var wsDiagnosticClients sync.Map

// One writer goroutine owns each client's counters; the logger reads atomically.
// Peak counters are cumulative for a connection, not per logging interval.
type wsDiagnosticCounters struct {
	all         wsAllWriteCounters
	samples     atomic.Uint64
	writeErrors atomic.Uint64
	queueSumNS  atomic.Int64
	writeSumNS  atomic.Int64
	queueMaxNS  atomic.Int64
	writeMaxNS  atomic.Int64
	queueMaxAt  atomic.Int64
	writeMaxAt  atomic.Int64
	depthMax    atomic.Int64
}

func wsAtomicMax(v *atomic.Int64, n int64) bool {
	for old := v.Load(); n > old; old = v.Load() {
		if v.CompareAndSwap(old, n) {
			return true
		}
	}
	return false
}
func (s *wsDiagnosticCounters) observeQueue(d time.Duration, depth int, at time.Time) {
	s.queueSumNS.Add(d.Nanoseconds())
	if wsAtomicMax(&s.queueMaxNS, d.Nanoseconds()) {
		s.queueMaxAt.Store(at.UnixNano())
	}
	wsAtomicMax(&s.depthMax, int64(depth))
}
func (s *wsDiagnosticCounters) observeWrite(d time.Duration, failed bool, at time.Time) {
	s.writeSumNS.Add(d.Nanoseconds())
	if wsAtomicMax(&s.writeMaxNS, d.Nanoseconds()) {
		s.writeMaxAt.Store(at.UnixNano())
	}
	if failed {
		s.writeErrors.Add(1)
	}
	s.samples.Add(1)
}

type wsDiagnosticSnapshot struct {
	Samples    uint64             `json:"sampled_writes"`
	Errors     uint64             `json:"sampled_write_errors"`
	QueueSumMS float64            `json:"queue_wait_sum_ms"`
	WriteSumMS float64            `json:"write_sum_ms"`
	QueueMaxMS float64            `json:"queue_wait_max_ms"`
	WriteMaxMS float64            `json:"write_max_ms"`
	QueueMaxAt string             `json:"queue_peak_at,omitempty"`
	WriteMaxAt string             `json:"write_peak_at,omitempty"`
	DepthMax   int64              `json:"sampled_dequeue_depth_max"`
	AllWrites  wsAllWriteSnapshot `json:"all_writes"`
}

func wsDiagnosticTime(n int64) string {
	if n == 0 {
		return ""
	}
	return time.Unix(0, n).UTC().Format(time.RFC3339Nano)
}
func (s *wsDiagnosticCounters) snapshot() wsDiagnosticSnapshot {
	return wsDiagnosticSnapshot{s.samples.Load(), s.writeErrors.Load(), float64(s.queueSumNS.Load()) / 1e6, float64(s.writeSumNS.Load()) / 1e6, float64(s.queueMaxNS.Load()) / 1e6, float64(s.writeMaxNS.Load()) / 1e6, wsDiagnosticTime(s.queueMaxAt.Load()), wsDiagnosticTime(s.writeMaxAt.Load()), s.depthMax.Load(), s.all.snapshot()}
}
func wsDiagnosticLog(record any) {
	b, err := json.Marshal(record)
	if err == nil {
		log.Printf("ws_diag %s", b)
	}
}
func wsLogClientDiagnostic(c *Client) {
	wsDiagnosticLog(struct {
		Kind      string               `json:"kind"`
		At        string               `json:"at"`
		AccountID int64                `json:"account_id"`
		Counters  wsDiagnosticSnapshot `json:"counters"`
		Outbox    map[string]uint64    `json:"outbox"`
	}{"client_closed", time.Now().UTC().Format(time.RFC3339Nano), c.accountID, c.diagStats.snapshot(), map[string]uint64{"replaced": c.movesReplaced.Load(), "evicted": c.movesEvicted.Load()}})
}

// Histograms are copied after each read; runtime owns and may reuse their storage.
type wsHistogramState map[string][]uint64

func (previous wsHistogramState) delta(name string, h *metrics.Float64Histogram) map[string]any {
	old := previous[name]
	var count uint64
	maxUpper := 0.0
	for i, n := range h.Counts {
		var before uint64
		if i < len(old) {
			before = old[i]
		}
		if n > before {
			count += n - before
			maxUpper = h.Buckets[i+1]
		}
	}
	previous[name] = append([]uint64(nil), h.Counts...)
	var upper any = maxUpper * 1000
	if math.IsInf(maxUpper, 1) {
		upper = "+Inf"
	}
	return map[string]any{"observations": count, "max_bucket_upper_ms": upper}
}
func init() {
	if !wsDiagnosticsEnabled {
		return
	}
	go func() {
		samples := []metrics.Sample{{Name: "/gc/cycles/total:gc-cycles"}, {Name: "/memory/classes/heap/objects:bytes"}, {Name: "/gc/pauses:seconds"}, {Name: "/sched/latencies:seconds"}}
		history := wsHistogramState{}
		// Prime histogram history before the first interval, excluding startup history.
		metrics.Read(samples)
		for _, s := range samples {
			if s.Value.Kind() == metrics.KindFloat64Histogram {
				history.delta(s.Name, s.Value.Float64Histogram())
			}
		}
		wsDiagnosticLog(map[string]any{"kind": "enabled", "at": time.Now().UTC().Format(time.RFC3339Nano), "sample_every": wsDiagnosticSampleEvery, "interval_seconds": 2})
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			record := map[string]any{"kind": "interval", "at": time.Now().UTC().Format(time.RFC3339Nano), "sample_every": wsDiagnosticSampleEvery}
			active := 0
			depth := 0
			var writes uint64
			var failures uint64
			var qsum, wsum float64
			peak := wsDiagnosticSnapshot{}
			var qAccount, wAccount int64
			wsDiagnosticClients.Range(func(key, value any) bool {
				c := key.(*Client)
				active++
				if n := len(c.send); n > depth {
					depth = n
				}
				s := c.diagStats.snapshot()
				writes += s.Samples
				failures += s.Errors
				qsum += s.QueueSumMS
				wsum += s.WriteSumMS
				wsMergeAllWrites(&peak.AllWrites, s.AllWrites)
				if s.QueueMaxMS > peak.QueueMaxMS {
					peak.QueueMaxMS = s.QueueMaxMS
					peak.QueueMaxAt = s.QueueMaxAt
					qAccount = c.accountID
				}
				if s.WriteMaxMS > peak.WriteMaxMS {
					peak.WriteMaxMS = s.WriteMaxMS
					peak.WriteMaxAt = s.WriteMaxAt
					wAccount = c.accountID
				}
				if s.DepthMax > peak.DepthMax {
					peak.DepthMax = s.DepthMax
				}
				return true
			})
			peak.Samples = writes
			peak.Errors = failures
			peak.QueueSumMS = qsum
			peak.WriteSumMS = wsum
			record["active_clients"] = active
			record["current_depth_max"] = depth
			record["active_counters"] = peak
			record["queue_peak_account_id"] = qAccount
			record["write_peak_account_id"] = wAccount
			slow := make([]wsSlowWriteEvent, 0, len(wsSlowWrites))
			for n := len(wsSlowWrites); n > 0; n-- {
				select {
				case e := <-wsSlowWrites:
					slow = append(slow, e)
				default:
				}
			}
			record["slow_writes"] = slow
			record["slow_events_dropped_total"] = wsSlowWritesDropped.Load()
			record["write_timing"] = "all_data_and_control"
			tcp := make([]wsTCPFrame, 0, len(wsTCPFrames))
			for n := len(wsTCPFrames); n > 0; n-- {
				select {
				case f := <-wsTCPFrames:
					tcp = append(tcp, f)
				default:
				}
			}
			record["tcp_samples"] = tcp
			metrics.Read(samples)
			for _, s := range samples {
				switch s.Value.Kind() {
				case metrics.KindUint64:
					record[s.Name] = s.Value.Uint64()
				case metrics.KindFloat64Histogram:
					record[s.Name] = history.delta(s.Name, s.Value.Float64Histogram())
				default:
					record[s.Name] = "unsupported"
				}
			}
			wsDiagnosticLog(record)
		}
	}()
}
