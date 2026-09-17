// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
)

type errorTransport struct{}

func (errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("transport failed")
}

func TestDeleteDataErrorNamesDelete(t *testing.T) {
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

func TestUseCACertRejectsInvalidPEM(t *testing.T) {
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
