package main

import (
	"net/http/httptest"
	"testing"
)

func TestExhibitionAdmission(t *testing.T) {
	if exhibitionConnectionLimit >= 256 {
		t.Fatal("admission must leave space in initial queue")
	}
	for i := 0; i < exhibitionConnectionLimit; i++ {
		if !tryReserveExhibitionSlot() {
			t.Fatal("early admission refusal")
		}
	}
	if tryReserveExhibitionSlot() {
		t.Fatal("capacity exceeded")
	}
	releaseExhibitionSlot()
	if !tryReserveExhibitionSlot() {
		t.Fatal("released slot unavailable")
	}
	for i := 0; i < exhibitionConnectionLimit; i++ {
		releaseExhibitionSlot()
	}
}

func TestExhibitionOrigin(t *testing.T) {
	t.Setenv("PUBLIC_ORIGIN", "https://49.212.160.97")
	for _, c := range []struct {
		origin string
		want   bool
	}{
		{"", true}, {"https://49.212.160.97", true}, {"https://other.example", false}, {"null", false},
	} {
		r := httptest.NewRequest("GET", "/ws", nil)
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		if allowedExhibitionOrigin(r) != c.want {
			t.Errorf("origin %q", c.origin)
		}
	}
}
