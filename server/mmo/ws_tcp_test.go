package main

import "testing"

func TestTCPQueueParsing(t *testing.T) {
	for _, line := range []string{"1: 050012AC:1F90 010012AC:C001 01 00000100:00000020 00:0 0 0", "2: 0000000000000000FFFF0000050012AC:1F90 0000000000000000FFFF0000010012AC:C001 01 00000100:00000020 00:0 0 0"} {
		p, tx, rx, ok := wsParseTCP(line)
		if !ok || p != 49153 || tx != 256 || rx != 32 {
			t.Fatalf("bad parse %d %d %d %v", p, tx, rx, ok)
		}
	}
	if _, _, _, ok := wsParseTCP("header"); ok {
		t.Fatal("accepted header")
	}
}
