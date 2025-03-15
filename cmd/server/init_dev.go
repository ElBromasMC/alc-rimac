//go:build dev

package main

import (
	"net/http"
)

func init() {
	// Live reload
	http.Get("http://localhost:8020")
}
