//go:build ignore

package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandle(t *testing.T) {
	rec := httptest.NewRecorder()
	Handle(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), "Hello") {
		t.Errorf("unexpected body: %q", rec.Body.String())
	}
}
