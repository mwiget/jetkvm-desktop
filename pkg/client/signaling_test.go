package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
