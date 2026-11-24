package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"
)

var capturedRoomSnapshots []json.RawMessage

// Optional live-room test hook: captures the same fixed SSH command used by WinForms.
func captureAdminRoom(t *testing.T, id, phase string, count int) {
	if os.Getenv("NETWORKDEMO_ROOM_MONITOR_LIVE_TEST") != "1" {
		return
	}
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ssh", "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=8", "hookrunner-vps", "docker", "exec", "hookrunner-exhibition-api-1", "/server", "--admin-rooms").Output()
	if err != nil {
		t.Fatal("room monitor read failed:", err)
	}
	var snapshot adminRoomSnapshot
	if err = json.Unmarshal(output, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Version != 1 || snapshot.ServerID == "" {
		t.Fatal("invalid monitor header")
	}
	found := false
	for _, r := range snapshot.Rooms {
		if r.ID == id {
			found = true
			if r.Phase != phase || len(r.Members) != count {
				t.Fatalf("monitor phase=%s members=%d", r.Phase, len(r.Members))
			}
		}
	}
	if !found {
		t.Fatal("room missing in monitor")
	}
	capturedRoomSnapshots = append(capturedRoomSnapshots, json.RawMessage(output))
	data, _ := json.Marshal(capturedRoomSnapshots)
	if err = os.WriteFile("../../live-room-monitor-snapshots.json", data, 0600); err != nil {
		t.Fatal(err)
	}
}
