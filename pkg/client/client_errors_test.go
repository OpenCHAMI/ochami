// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type errorTransport struct{}

func (errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("transport failed")
}

// TestDeleteData_NamesDelete verifies DeleteData's error messages say DELETE,
// not PATCH (a copy-paste artifact from an earlier version of the method).
func TestDeleteData_NamesDelete(t *testing.T) {
	c, err := NewOchamiClient("test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	c.Client = &http.Client{Transport: errorTransport{}}
	_, err = c.DeleteData("items", "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "DELETE") || strings.Contains(err.Error(), "PATCH") {
		t.Fatalf("DeleteData error = %v", err)
	}
}

// TestUseCACert_RejectsInvalidPEM verifies UseCACert reports an error instead
// of silently succeeding when the given file contains no valid certificates.
func TestUseCACert_RejectsInvalidPEM(t *testing.T) {
	c, err := NewOchamiClient("test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/ca.pem"
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := c.UseCACert(path); err == nil {
		t.Fatal("UseCACert accepted invalid PEM")
	}
}

// closedServerClient returns a client whose base URI points at a server that
// has been closed, so every request fails at the transport layer.
func closedServerClient(t *testing.T) *OchamiClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	oc, err := NewOchamiClient("test", url, WithInsecure(true))
	if err != nil {
		t.Fatalf("NewOchamiClient: %v", err)
	}
	return oc
}

func TestDataWrappers_RequestErrors(t *testing.T) {
	oc := closedServerClient(t)

	if _, err := oc.GetData("/x", "", nil); err == nil {
		t.Error("GetData against closed server = nil, want error")
	}
	if _, err := oc.PostData("/x", "", nil, []byte(`{}`)); err == nil {
		t.Error("PostData against closed server = nil, want error")
	}
	if _, err := oc.PutData("/x", "", nil, []byte(`{}`)); err == nil {
		t.Error("PutData against closed server = nil, want error")
	}
	if _, err := oc.PatchData("/x", "", nil, []byte(`{}`)); err == nil {
		t.Error("PatchData against closed server = nil, want error")
	}
	if _, err := oc.DeleteData("/x", "", nil, nil); err == nil {
		t.Error("DeleteData against closed server = nil, want error")
	}
}
