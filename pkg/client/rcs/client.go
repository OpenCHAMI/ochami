// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"

	"github.com/OpenCHAMI/ochami/pkg/client"
)

// ctrlCByte is the byte value for Ctrl+C in raw terminal mode.
const ctrlCByte = byte(0x03)

// HealthResponse represents the response from the /health endpoint of the Remote Console Service.
type HealthResponse struct {
	NumberConsoles     string `json:"consoles"`
	LastHardwareUpdate string `json:"hardwareupdate"`
}

// ConsolesResponse represents the response from the /consoles endpoint, containing a list of available consoles.
type ConsolesResponse struct {
	Consoles []NodeConsoleInfo `json:"consoles"`
}

// NodeConsoleInfo represents the information about a single console for a node, including connection details.
type NodeConsoleInfo struct {
	ID                  string `json:"id" yaml:"id"`
	ConnectionType      string `json:"connectionType" yaml:"connectionType"`
	ConnectionHost      string `json:"connectionHost" yaml:"connectionHost"`
	ConnectionPort      int    `json:"connectionPort,omitempty" yaml:"connectionPort,omitempty"`
	ConsoleEntryCommand string `json:"consoleEntryCommand,omitempty" yaml:"consoleEntryCommand,omitempty"`
}

type RCSClient struct {
	*client.OchamiClient
}

// NewClient creates a new RCSClient with the given base URI and TLS settings.
func NewClient(baseURI string, insecure bool) (*RCSClient, error) {
	oc, err := client.NewOchamiClient("Remote Console", baseURI, insecure)
	if err != nil {
		return nil, err
	}
	return &RCSClient{oc}, nil
}

// headersForToken creates HTTP headers with the given token for authentication.
func headersForToken(token string) (*client.HTTPHeaders, error) {
	headers := client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return nil, fmt.Errorf("failed to set token in HTTP headers: %w", err)
		}
	}

	return headers, nil
}

// dialWebSocket constructs the websocket URL for the console endpoint and attempts to establish a connection with the appropriate headers.
func (c *RCSClient) dialWebSocket(ctx context.Context, nodeID string, query string, headers *client.HTTPHeaders) (*websocket.Conn, error) {
	endpoint := fmt.Sprintf("/consoles/%s", nodeID)
	uriStr, err := c.GetURI(endpoint, query)
	if err != nil {
		return nil, err
	}

	u, _ := url.Parse(uriStr)
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else if u.Scheme == "http" {
		u.Scheme = "ws"
	}

	dialer := websocket.DefaultDialer
	var requestHeaders http.Header
	if headers != nil {
		requestHeaders = http.Header(*headers)
	}

	conn, resp, err := dialer.DialContext(ctx, u.String(), requestHeaders)
	if err != nil {
		return nil, websocketDialError(nodeID, resp, err)
	}

	return conn, nil
}

// websocketDialError constructs an error message based on the HTTP response from a failed websocket dial attempt.
func websocketDialError(nodeID string, resp *http.Response, err error) error {
	if resp == nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("failed to dial websocket: %s", resp.Status)
	}

	msg := strings.TrimSpace(string(body))
	if resp.StatusCode == http.StatusConflict {
		if msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return fmt.Errorf("interactive console for %s is already in use", nodeID)
	}

	if msg != "" {
		return fmt.Errorf("failed to dial websocket: %s: %s", resp.Status, msg)
	}

	return fmt.Errorf("failed to dial websocket: %s", resp.Status)
}

// GetStatus retrieves the health status of the Remote Console Service using the /health endpoint.
func (c *RCSClient) GetStatus(token string) (*HealthResponse, error) {
	headers, err := headersForToken(token)
	if err != nil {
		return nil, err
	}

	he, err := c.GetData("/health", "", headers)
	if err != nil {
		return nil, err
	}

	var resp HealthResponse
	if err := json.Unmarshal(he.Body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health response: %w", err)
	}
	return &resp, nil
}

// ListConsoles retrieves the list of available consoles from the Remote Console Service using the /consoles endpoint.
func (c *RCSClient) ListConsoles(token string) ([]NodeConsoleInfo, error) {
	headers, err := headersForToken(token)
	if err != nil {
		return nil, err
	}

	he, err := c.GetData("/consoles", "", headers)
	if err != nil {
		return nil, err
	}

	var resp ConsolesResponse
	if err := json.Unmarshal(he.Body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal consoles response: %w", err)
	}
	return resp.Consoles, nil
}

// ShowConsole connects to the console for the specified node and streams its output to the provided writer.
func (c *RCSClient) ShowConsole(ctx context.Context, nodeID string, follow bool, lines int, token string, output io.Writer) error {
	headers, err := headersForToken(token)
	if err != nil {
		return err
	}

	conn, err := c.dialWebSocket(ctx, nodeID, fmt.Sprintf("mode=tail&follow=%t&lines=%d", follow, lines), headers)
	if err != nil {
		return err
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if isNormalWebSocketClose(err) {
				return nil
			}
			return err
		}
		output.Write(message)
	}
}

// isNormalWebSocketClose reports whether err is a clean websocket close.
func isNormalWebSocketClose(err error) bool {
	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) {
		return false
	}

	return closeErr.Code == websocket.CloseNormalClosure
}

// terminalInputState restores stdin after raw terminal mode has been enabled.
type terminalInputState struct {
	file  *os.File
	state *term.State
}

func (t terminalInputState) Restore() error {
	if t.file == nil || t.state == nil {
		return nil
	}

	return term.Restore(int(t.file.Fd()), t.state)
}

// terminalInputFile checks if stdin is a terminal and returns the file if so.
func terminalInputFile(stdin io.Reader) (*os.File, bool) {
	stdinFile, ok := stdin.(*os.File)
	if !ok {
		return nil, false
	}

	if !term.IsTerminal(int(stdinFile.Fd())) {
		return nil, false
	}

	return stdinFile, true
}

func enableRawTerminalMode(stdinFile *os.File) (*term.State, error) {
	oldState, err := term.GetState(int(stdinFile.Fd()))
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal state: %w", err)
	}

	if _, err := term.MakeRaw(int(stdinFile.Fd())); err != nil {
		return nil, fmt.Errorf("failed to set terminal raw mode: %w", err)
	}

	return oldState, nil
}

// streamRawConsoleInput reads from stdin in raw mode and forwards keystrokes to the websocket connection, translating Ctrl+C into an interrupt signal.
func streamRawConsoleInput(stdin io.Reader, conn *websocket.Conn, interrupt chan os.Signal, errChan chan error) {
	buf := make([]byte, 1)
	for {
		bytesRead, err := stdin.Read(buf)
		if err != nil {
			if err != io.EOF {
				errChan <- err
			}
			return
		}

		if bytesRead == 0 {
			continue
		}

		// In raw mode, Ctrl+C arrives as the ETX byte instead of a signal.
		if buf[0] == ctrlCByte {
			interrupt <- syscall.SIGINT
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, buf[:bytesRead]); err != nil {
			errChan <- err
			return
		}
	}
}

func streamBufferedConsoleInput(stdin io.Reader, conn *websocket.Conn, errChan chan error) {
	buf := make([]byte, 1024)
	for {
		bytesRead, err := stdin.Read(buf)
		if err != nil {
			if err != io.EOF {
				errChan <- err
			}
			return
		}

		if bytesRead == 0 {
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, buf[:bytesRead]); err != nil {
			errChan <- err
			return
		}
	}
}

// startConsoleInputStream starts stdin forwarding and returns terminal state for cleanup.
func startConsoleInputStream(stdin io.Reader, conn *websocket.Conn, interrupt chan os.Signal, errChan chan error) (terminalInputState, error) {

	// If stdin is a terminal, enable raw mode for immediate keystroke forwarding and interrupt handling. Otherwise, stream input in buffered mode.
	stdinFile, ok := terminalInputFile(stdin)
	if !ok {
		// Piped or redirected input should stay buffered so non-interactive input still works.
		go streamBufferedConsoleInput(stdin, conn, errChan)

		return terminalInputState{}, nil
	}

	// Raw mode lets us forward keystrokes immediately instead of waiting for line buffering.
	oldState, err := enableRawTerminalMode(stdinFile)
	if err != nil {
		return terminalInputState{}, err
	}

	go streamRawConsoleInput(stdin, conn, interrupt, errChan)

	return terminalInputState{file: stdinFile, state: oldState}, nil
}

func streamConsoleOutput(stdout io.Writer, conn *websocket.Conn, errChan chan error, done chan struct{}) {
	defer close(done)
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			errChan <- err
			return
		}
		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			if _, err := stdout.Write(message); err != nil {
				errChan <- err
				return
			}
		}
	}
}

// startConsoleOutputStream starts websocket output forwarding to stdout.
func startConsoleOutputStream(stdout io.Writer, conn *websocket.Conn, errChan chan error, done chan struct{}) {
	go streamConsoleOutput(stdout, conn, errChan, done)
}

// waitForConsoleExit waits for shutdown, an interrupt, or an I/O error.
func waitForConsoleExit(ctx context.Context, conn *websocket.Conn, interrupt chan os.Signal, done chan struct{}, errChan chan error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-interrupt:
		// Translate local interrupt into a clean websocket close so the remote side can shut down cleanly.
		if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			return nil
		}
		// Give the read goroutine a moment to observe the close before returning.
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *RCSClient) ConnectConsole(ctx context.Context, nodeID string, token string, stdin io.Reader, stdout io.Writer) error {
	headers, err := headersForToken(token)
	if err != nil {
		return err
	}

	conn, err := c.dialWebSocket(ctx, nodeID, "mode=interactive", headers)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Set up interrupt handling to allow Ctrl+C to cleanly close the console connection.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	errChan := make(chan error, 2)
	done := make(chan struct{})

	restoreTerminal, err := startConsoleInputStream(stdin, conn, interrupt, errChan)
	if err != nil {
		return err
	}

	// Restore the terminal when the console session ends, even if there are errors or interrupts.
	defer func() {
		_ = restoreTerminal.Restore()
	}()

	startConsoleOutputStream(stdout, conn, errChan, done)

	/// Wait for the console session to end due to shutdown, interrupt, or an I/O error.
	return waitForConsoleExit(ctx, conn, interrupt, done, errChan)
}
