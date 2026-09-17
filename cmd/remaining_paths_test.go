// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestMetacommandPathsPrintUsage(t *testing.T) {
	paths := [][]string{
		{},
		{"boot"}, {"boot", "bmc"}, {"boot", "config"}, {"boot", "node"}, {"boot", "service"},
		{"bss"}, {"bss", "boot"}, {"bss", "boot", "image"}, {"bss", "boot", "params"},
		{"bss", "boot", "script"}, {"bss", "hosts"}, {"bss", "service"},
		{"cloud-init"}, {"cloud-init", "defaults"}, {"cloud-init", "group"}, {"cloud-init", "group", "get"}, {"cloud-init", "node"}, {"cloud-init", "node", "get"}, {"cloud-init", "service"},
		{"config"}, {"config", "cluster"}, {"discover"},
		{"metadata"}, {"metadata", "defaults"}, {"metadata", "group"}, {"metadata", "instance"}, {"metadata", "peer"}, {"metadata", "service"},
		{"pcs"}, {"pcs", "service"}, {"pcs", "status"}, {"pcs", "transition"},
		{"rcs"}, {"rcs", "console"}, {"rcs", "service"},
		{"smd"}, {"smd", "compep"}, {"smd", "component"}, {"smd", "group"}, {"smd", "group", "member"},
		{"smd", "iface"}, {"smd", "rfe"}, {"smd", "service"},
	}

	for _, path := range paths {
		name := "root"
		if len(path) > 0 {
			name = strings.Join(path, " ")
		}
		t.Run(name, func(t *testing.T) {
			args := append(append([]string{}, path...), "--ignore-config")
			res := runOchami(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v", res.err)
			}
			if !strings.Contains(res.stdout, "Usage:") {
				t.Errorf("stdout = %q, want usage", res.stdout)
			}
		})
	}
}

func TestCloudInitGroupGetRemainingPaths(t *testing.T) {
	for _, subcommand := range []string{"config", "meta-data"} {
		t.Run(subcommand, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{}`) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "cloud-init", "group", "get", subcommand,
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

func TestRemainingServicePaths(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		body     string
	}{
		{"bss version", []string{"bss", "service", "version"}, "/service/version", `{"version":"1.0"}`},
		{"cloud-init version", []string{"cloud-init", "service", "version"}, "/version", `{"version":"1.0"}`},
		{"metadata status", []string{"metadata", "service", "status"}, "/health", `{}`},
		{"rcs status", []string{"rcs", "service", "status", "--token", "t"}, "/health", `{"status":"ok"}`},
		{"deprecated smd status", []string{"smd", "status"}, "/service/ready", `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.body) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			args := append(append([]string{}, tc.args...), "--ignore-config", "--uri", srv.URL)
			res := runOchami(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}

func TestMetadataPatchPathsAndArrayOperations(t *testing.T) {
	for _, resource := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(resource, func(t *testing.T) {
			var gotContentType, gotBody string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotContentType = r.Header.Get("Content-Type")
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				gotBody = string(body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"metadata":{"uid":"some-uid","name":"thing"},"spec":{}}`) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", resource, "patch", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "--add", "items=value")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if gotContentType != "application/json-patch+json" {
				t.Errorf("Content-Type = %q, want application/json-patch+json", gotContentType)
			}
			if !strings.Contains(gotBody, `"op":"add"`) || !strings.Contains(gotBody, `"path":"/items/-"`) {
				t.Errorf("body = %q, want RFC 6902 add operation", gotBody)
			}
		})
	}
}

func TestRCSConsoleConnect(t *testing.T) {
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("connected")); err != nil {
			t.Errorf("write console message: %v", err)
			return
		}
		if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			t.Errorf("write close message: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "", "rcs", "console", "connect", "x0c0s1b0n0",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
}
