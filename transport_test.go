package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBotHTTPTransport(t *testing.T) {
	const token = "123456789:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghi"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/bot"+token+"/getMe" {
			t.Errorf("unexpected Telegram request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"result":{"id":123456789,"is_bot":true,"first_name":"Synthetic","username":"synthetic_bot"}}`)
	}))
	defer server.Close()
	bot, err := NewBot(token, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	me, err := bot.GetMe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if me.Username != "synthetic_bot" {
		t.Fatalf("unexpected bot identity: %q", me.Username)
	}
}
