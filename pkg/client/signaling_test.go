package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/lkarlslund/jetkvm-desktop/pkg/protocol/auth"
)

// Current firmware rejects the signaling websocket with 401 until the client
// logs in, and has no legacy /webrtc/session endpoint to fall back to.
func TestConnectReportsAuthErrorWhenWebsocketRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/webrtc/signaling/client" {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cl, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = cl.Connect(ctx)
	var authErr *auth.Error
	if !errors.As(err, &authErr) || authErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected auth error with status 401, got %v", err)
	}
}

// A device that stops answering midway through the handshake must not leave
// Connect waiting for good: closing the client, as a reconnect does, ends it.
func TestCloseEndsConnectStuckInSignaling(t *testing.T) {
	for _, tc := range []struct {
		name         string
		sendMetadata bool
	}{
		{name: "waiting for metadata"},
		{name: "waiting for answer", sendMetadata: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan struct{})
			defer close(done)
			upgrader := websocket.Upgrader{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/webrtc/signaling/client" {
					http.NotFound(w, r)
					return
				}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()
				if tc.sendMetadata {
					_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"device-metadata","data":{"deviceVersion":"0.4.0"}}`))
				}
				<-done
			}))
			defer srv.Close()

			cl, err := New(Config{BaseURL: srv.URL})
			if err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() { result <- cl.Connect(context.Background()) }()

			time.Sleep(200 * time.Millisecond)
			_ = cl.Close()
			select {
			case err := <-result:
				if err == nil {
					t.Fatal("expected Connect to fail after Close")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("Connect still waiting after Close")
			}
		})
	}
}
