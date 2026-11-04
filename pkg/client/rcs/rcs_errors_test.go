// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

// rcs_errors_test.go unit-tests the RCSClient's error arms: a non-2XX response
// from the plain HTTP endpoints, a websocket dial failure for the
// console-streaming endpoints, a failure to close the console connection,
// malformed response bodies, websocketDialError's classification of failed
// dials, and the rejection of a malformed client URI or an invalid node ID.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/openchami/ochami/pkg/client"
)

// TestListConsoles_HTTPError verifies a non-2XX response is returned as an error.
func TestListConsoles_HTTPError(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	})
	defer srv.Close()

	if _, err := c.ListConsoles(context.Background(), "tok"); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestShowConsole_DialError verifies ShowConsole returns an error when the
// websocket dial fails (server rejects the upgrade).
func TestShowConsole_DialError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do not upgrade; respond with an error status so the dial fails.
		http.Error(w, "in use", http.StatusConflict)
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.ShowConsole(ctx, "x0c0s1b0n0", false, 10, "tok", &bytes.Buffer{}); err == nil {
		t.Fatal("expected a dial error, got nil")
	}
}

// mkResp builds a minimal *http.Response with the given status and body.
func mkResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// TestWebsocketDial_Error verifies that websocketDialError wraps a dial failure
// with no response, reports a 409 as the server's message or as the node's
// console already being in use, and includes any message from other statuses.
func TestWebsocketDial_Error(t *testing.T) {
	// nil response -> wraps the dial error.
	if err := websocketDialError("n0", nil, io.EOF); err == nil {
		t.Error("nil response = nil, want error")
	}

	// 409 with a message -> returns the message.
	if err := websocketDialError("n0", mkResp(http.StatusConflict, "busy"), nil); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Errorf("409 with message = %v, want it to contain the message", err)
	}

	// 409 without a message -> "already in use" for the node.
	if err := websocketDialError("n0", mkResp(http.StatusConflict, ""), nil); err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Errorf("409 without message = %v, want already-in-use", err)
	}

	// Non-409 with a message.
	if err := websocketDialError("n0", mkResp(http.StatusInternalServerError, "oops"), nil); err == nil || !strings.Contains(err.Error(), "oops") {
		t.Errorf("500 with message = %v, want it to contain the message", err)
	}

	// Non-409 without a message.
	if err := websocketDialError("n0", mkResp(http.StatusBadGateway, ""), nil); err == nil {
		t.Error("502 without message = nil, want error")
	}
}

// TestDialWebSocket_MalformedURI verifies dialWebSocket returns an error
// instead of dereferencing a nil *url.URL when the constructed console URI
// fails to parse (e.g. a raw control character in the query string, which
// url.URL.String() writes verbatim into RawQuery without escaping).
func TestDialWebSocket_MalformedURI(t *testing.T) {
	c, err := NewClient("http://example.com")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.dialWebSocket(context.Background(), "n0", "mode=tail\nfollow=true", nil)
	if err == nil {
		t.Fatal("dialWebSocket with malformed query = nil error, want error")
	}
	if !strings.Contains(err.Error(), "failed to parse console URI") {
		t.Errorf("dialWebSocket error = %v, want it to mention URI parsing", err)
	}
}

// TestDialWebSocket_InvalidNodeID verifies that the console methods reject a
// node ID that is blank or is not a single path segment, wrapping
// client.InvalidArgumentError, without attempting a connection.
func TestDialWebSocket_InvalidNodeID(t *testing.T) {
	var requests atomic.Int32
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
	})
	defer srv.Close()

	for _, id := range []string{"", " ", ".", "..", "a/b"} {
		if err := c.ShowConsole(context.Background(), id, false, 10, "", io.Discard); !errors.Is(err, client.InvalidArgumentError) {
			t.Errorf("ShowConsole(%q) error = %v, want it to wrap client.InvalidArgumentError", id, err)
		}
		if err := c.ConnectConsole(context.Background(), id, "", strings.NewReader(""), io.Discard); !errors.Is(err, client.InvalidArgumentError) {
			t.Errorf("ConnectConsole(%q) error = %v, want it to wrap client.InvalidArgumentError", id, err)
		}
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("request count = %d, want 0", got)
	}
}

// TestGetStatus_HTTPError verifies that GetStatus returns an error for a
// non-2XX response.
func TestGetStatus_HTTPError(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	if _, err := c.GetStatus(context.Background(), "tok"); err == nil {
		t.Error("GetStatus with HTTP error = nil, want error")
	}
}

// TestGetStatus_MalformedBody verifies that GetStatus returns an error when the
// response body isn't valid JSON.
func TestGetStatus_MalformedBody(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	})
	defer srv.Close()

	if _, err := c.GetStatus(context.Background(), "tok"); err == nil {
		t.Error("GetStatus with malformed body = nil, want error")
	}
}

// TestListConsoles_MalformedBody verifies that ListConsoles returns an error
// when the response body isn't valid JSON.
func TestListConsoles_MalformedBody(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	})
	defer srv.Close()

	if _, err := c.ListConsoles(context.Background(), "tok"); err == nil {
		t.Error("ListConsoles with malformed body = nil, want error")
	}
}

// TestRCSNewClientAndDial_RejectMalformedURIs verifies that NewClient rejects a
// malformed URI and that dialWebSocket fails when the client has no base URI.
func TestRCSNewClientAndDial_RejectMalformedURIs(t *testing.T) {
	if _, err := NewClient("https://example.com/%zz"); err == nil {
		t.Fatal("NewClient() accepted malformed URI")
	}
	c := &RCSClient{OchamiClient: &client.OchamiClient{}}
	if _, err := c.dialWebSocket(context.Background(), "x0", "", nil); err == nil {
		t.Fatal("dialWebSocket() accepted nil base URI")
	}
}

// TestShowConsole_ReturnsConnectionCloseError verifies that ShowConsole returns
// a failure to close the websocket connection after a normal close from the
// server.
func TestShowConsole_ReturnsConnectionCloseError(t *testing.T) {
	want := errors.New("close failed")
	conn := &fakeMessageConn{
		readErr:  &websocket.CloseError{Code: websocket.CloseNormalClosure},
		closeErr: want,
	}
	c, err := NewClient("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	c.dial = func(context.Context, string, http.Header) (messageConn, *http.Response, error) {
		return conn, nil, nil
	}
	if err := c.ShowConsole(context.Background(), "x0", false, 1, "", io.Discard); !errors.Is(err, want) {
		t.Fatalf("ShowConsole() error = %v, want close error", err)
	}
}

// closeUnblocksConn is a messageConn whose ReadMessage blocks until Close is
// called, like a websocket whose server never sends anything.
type closeUnblocksConn struct {
	*fakeMessageConn
	once   sync.Once
	closed chan struct{}
	closes atomic.Int32
}

func (c *closeUnblocksConn) ReadMessage() (int, []byte, error) {
	<-c.closed
	return 0, nil, net.ErrClosed
}

func (c *closeUnblocksConn) Close() error {
	c.closes.Add(1)
	c.once.Do(func() { close(c.closed) })
	return nil
}

// TestShowConsole_Cancelled verifies that cancelling the context returns a
// ShowConsole blocked waiting for output with context.Canceled, and closes the
// connection exactly once.
func TestShowConsole_Cancelled(t *testing.T) {
	conn := &closeUnblocksConn{fakeMessageConn: &fakeMessageConn{}, closed: make(chan struct{})}
	c, err := NewClient("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	c.dial = func(context.Context, string, http.Header) (messageConn, *http.Response, error) {
		return conn, nil, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(10*time.Millisecond, cancel)

	done := make(chan error, 1)
	go func() { done <- c.ShowConsole(ctx, "x0", true, 1, "", io.Discard) }()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ShowConsole() error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ShowConsole() did not return after its context was cancelled")
	}
	if n := conn.closes.Load(); n != 1 {
		t.Errorf("Close() called %d times, want 1", n)
	}
}

// stalledHandshakeServer starts a listener that accepts connections but never
// answers the websocket upgrade request, and returns its base URI and a
// channel that is closed when the first connection's client closes it.
func stalledHandshakeServer(t *testing.T) (string, <-chan struct{}) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	closed := make(chan struct{})
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Read the upgrade request and never reply; a read error means the
		// client closed the connection.
		_, _ = io.Copy(io.Discard, conn) //nolint:errcheck // the error is the signal
		close(closed)
	}()
	return "http://" + l.Addr().String(), closed
}

// TestRCSClient_HandshakeCancelled verifies that cancelling the context
// aborts a websocket handshake the server never answers: ShowConsole and
// ConnectConsole return context.Canceled promptly instead of waiting out the
// dialer's handshake timeout, and the connection to the server is closed.
func TestRCSClient_HandshakeCancelled(t *testing.T) {
	tests := []struct {
		name string
		run  func(ctx context.Context, c *RCSClient) error
	}{
		{name: "ShowConsole", run: func(ctx context.Context, c *RCSClient) error {
			return c.ShowConsole(ctx, "x0", true, 1, "", io.Discard)
		}},
		{name: "ConnectConsole", run: func(ctx context.Context, c *RCSClient) error {
			return c.ConnectConsole(ctx, "x0", "", strings.NewReader(""), io.Discard)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uri, closed := stalledHandshakeServer(t)
			c, err := NewClient(uri)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			time.AfterFunc(50*time.Millisecond, cancel)

			done := make(chan error, 1)
			go func() { done <- tc.run(ctx, c) }()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error = %v, want context.Canceled", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("did not return within 5s of cancellation")
			}
			select {
			case <-closed:
			case <-time.After(5 * time.Second):
				t.Error("connection to the server was not closed")
			}
		})
	}
}

// TestDialContext_CancelledWhileConnecting verifies that a cancellation
// arriving after the network connection is made, but before dialContext has
// recorded it, still aborts the handshake: the connection is closed and
// dialContext returns context.Canceled promptly rather than at the dialer's
// handshake timeout.
func TestDialContext_CancelledWhileConnecting(t *testing.T) {
	uri, closed := stalledHandshakeServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dialer := *websocket.DefaultDialer
	dialer.NetDialContext = func(dialCtx context.Context, network, addr string) (net.Conn, error) {
		c, err := (&net.Dialer{}).DialContext(dialCtx, network, addr)
		// Cancel between connecting and returning the connection, and give
		// the cancellation callback time to run before it is recorded.
		cancel()
		time.Sleep(100 * time.Millisecond)
		return c, err
	}

	done := make(chan error, 1)
	go func() {
		_, _, err := dialContext(ctx, dialer, "ws"+strings.TrimPrefix(uri, "http")+"/consoles/x0", nil)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("dialContext() error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("dialContext() did not return within 5s of cancellation")
	}
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Error("connection to the server was not closed")
	}
}
