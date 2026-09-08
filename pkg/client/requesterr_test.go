// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

// requesterr_test.go covers the request-error arms of the GetData/PostData/
// PutData/PatchData/DeleteData wrappers by pointing the client at a closed
// server so the underlying transport fails.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestDataWrappersRequestErrors(t *testing.T) {
	oc := closedServerClient(t)

	if _, err := oc.GetData(context.Background(), "/x", "", nil); err == nil {
		t.Error("GetData against closed server = nil, want error")
	}
	if _, err := oc.PostData(context.Background(), "/x", "", nil, []byte(`{}`)); err == nil {
		t.Error("PostData against closed server = nil, want error")
	}
	if _, err := oc.PutData(context.Background(), "/x", "", nil, []byte(`{}`)); err == nil {
		t.Error("PutData against closed server = nil, want error")
	}
	if _, err := oc.PatchData(context.Background(), "/x", "", nil, []byte(`{}`)); err == nil {
		t.Error("PatchData against closed server = nil, want error")
	}
	if _, err := oc.DeleteData(context.Background(), "/x", "", nil, nil); err == nil {
		t.Error("DeleteData against closed server = nil, want error")
	}
}
