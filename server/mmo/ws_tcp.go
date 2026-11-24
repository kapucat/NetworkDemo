package main

import (
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type wsTCPConn struct {
	AccountID int64  `json:"account_id"`
	Port      int    `json:"remote_port"`
	TX        uint64 `json:"tx_bytes"`
	RX        uint64 `json:"rx_bytes"`
}
type wsTCPFrame struct {
	At      string      `json:"at"`
	ScanMS  float64     `json:"scan_ms"`
	Matched int         `json:"matched"`
	TX      uint64      `json:"total_tx_bytes"`
	RX      uint64      `json:"total_rx_bytes"`
	Top     []wsTCPConn `json:"top"`
	Errors  int         `json:"read_errors"`
}

var wsTCPFrames = make(chan wsTCPFrame, 128)

func wsParseTCP(line string) (port int, tx, rx uint64, ok bool) {
	f := strings.Fields(line)
	if len(f) < 5 || (!strings.HasSuffix(f[1], ":1F90")) || (f[3] != "01" && f[3] != "08") {
		return
	}
	remote := strings.Split(f[2], ":")
	q := strings.Split(f[4], ":")
	if len(remote) != 2 || len(q) != 2 {
		return
	}
	p, e := strconv.ParseUint(remote[1], 16, 16)
	if e != nil {
		return
	}
	tx, e = strconv.ParseUint(q[0], 16, 64)
	if e != nil {
		return
	}
	rx, e = strconv.ParseUint(q[1], 16, 64)
	if e != nil {
		return
	}
	return int(p), tx, rx, true
}
func init() {
	if !wsDiagnosticsEnabled {
		return
	}
	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for range t.C {
			start := time.Now()
			ports := map[int]int64{}
			wsDiagnosticClients.Range(func(k, v any) bool {
				c := k.(*Client)
				_, p, e := net.SplitHostPort(c.remoteAddr)
				if e == nil {
					n, _ := strconv.Atoi(p)
					if _, exists := ports[n]; exists {
						ports[n] = 0
					} else {
						ports[n] = c.accountID
					}
				}
				return true
			})
			frame := wsTCPFrame{At: start.UTC().Format(time.RFC3339Nano)}
			conns := []wsTCPConn{}
			for _, file := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
				b, e := os.ReadFile(file)
				if e != nil {
					frame.Errors++
					continue
				}
				for _, line := range strings.Split(string(b), "\n") {
					p, tx, rx, ok := wsParseTCP(line)
					id := ports[p]
					if ok && id != 0 {
						frame.Matched++
						frame.TX += tx
						frame.RX += rx
						conns = append(conns, wsTCPConn{id, p, tx, rx})
					}
				}
			}
			sort.Slice(conns, func(i, j int) bool { return conns[i].TX > conns[j].TX })
			if len(conns) > 16 {
				conns = conns[:16]
			}
			frame.Top = conns
			frame.ScanMS = float64(time.Since(start).Nanoseconds()) / 1e6
			select {
			case wsTCPFrames <- frame:
			default:
				wsDiagnosticLog(map[string]any{"kind": "tcp_frame_dropped", "at": frame.At})
			}
		}
	}()
}
