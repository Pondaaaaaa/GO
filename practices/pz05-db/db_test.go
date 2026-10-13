package main

import (
	"context"
	"strings"
	"testing"
)

func TestOpenDBRequiresDatabaseURL(t *testing.T) {
	_, err := openDB(context.Background(), "  ")
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("openDB error = %v, want DATABASE_URL instruction", err)
	}
}
