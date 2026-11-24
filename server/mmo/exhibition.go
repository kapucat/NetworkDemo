package main

import (
	"net/http"
	"os"
	"strings"
)

// Keep admission below the initial registration queue capacity (256).
const exhibitionConnectionLimit = 200

var exhibitionSlots = make(chan struct{}, exhibitionConnectionLimit)

func tryReserveExhibitionSlot() bool {
	select {
	case exhibitionSlots <- struct{}{}:
		return true
	default:
		return false
	}
}
func releaseExhibitionSlot() { <-exhibitionSlots }

func allowedExhibitionOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	} // Native Unity clients do not send Origin.
	expected := strings.TrimRight(os.Getenv("PUBLIC_ORIGIN"), "/")
	return expected != "" && origin == expected
}
