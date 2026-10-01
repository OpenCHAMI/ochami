// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	_, err = c.DeleteData(context.Background(), "items", "", nil, nil)
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

// generateTestCA creates a self-signed CA certificate/key pair for testing,
// returning the parsed certificate (for signing leaf certs), the private key,
// and the CA certificate PEM-encoded.
func generateTestCA(t *testing.T) (*x509.Certificate, *rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}
	return cert, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// generateExpiredLeafCert creates a server certificate for 127.0.0.1, signed
// by ca/caKey, whose validity period has already elapsed.
func generateExpiredLeafCert(t *testing.T, ca *x509.Certificate, caKey *rsa.PrivateKey) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-2 * time.Hour),
		NotAfter:     time.Now().Add(-time.Hour), // expired
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf cert: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("build tls.Certificate: %v", err)
	}
	return tlsCert
}

// TestUseCACert_RejectsExpiredServerCertificate verifies that a request made
// with a CA certificate loaded via UseCACert still fails when the server's own
// certificate has expired. UseCACert only parses the CA's PEM and never
// inspects certificate validity dates, so this failure can only be observed at
// the TLS handshake, not at UseCACert's call site.
func TestUseCACert_RejectsExpiredServerCertificate(t *testing.T) {
	ca, caKey, caPEM := generateTestCA(t)
	leafCert := generateExpiredLeafCert(t, ca, caKey)

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{leafCert}}
	srv.StartTLS()
	defer srv.Close()

	caPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caPath, caPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	c, err := NewOchamiClient("test", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UseCACert(caPath); err != nil {
		t.Fatalf("UseCACert(valid CA) = %v, want nil", err)
	}

	_, err = c.MakeRequest(context.Background(), http.MethodGet, srv.URL, nil, nil)
	if err == nil {
		t.Fatal("MakeRequest against server with expired certificate = nil, want error")
	}
	var certErr x509.CertificateInvalidError
	if !errors.As(err, &certErr) || certErr.Reason != x509.Expired {
		t.Errorf("MakeRequest error = %v, want x509.CertificateInvalidError{Reason: Expired}", err)
	}
}

// closedServerClient returns a client whose base URI points at an address
// nothing listens on, so every request fails at the transport layer.
func closedServerClient(t *testing.T) *OchamiClient {
	t.Helper()
	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused
	oc, err := NewOchamiClient("test", url, WithInsecure(true))
	if err != nil {
		t.Fatalf("NewOchamiClient: %v", err)
	}
	return oc
}

// TestDataWrappers_RequestErrors verifies that GetData, PostData, PutData,
// PatchData, and DeleteData return an error when the server can't be reached.
func TestDataWrappers_RequestErrors(t *testing.T) {
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
