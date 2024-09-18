package client

import (
	"encoding/json"
	"fmt"

	"github.com/OpenCHAMI/bss/pkg/bssTypes"
)

// BSSClient is an OchamiClient that has its BasePath set configured to the one
// that BSS uses.
type BSSClient struct {
	*OchamiClient
}

const (
	serviceNameBSS = "BSS"
	basePathBSS    = "/boot/v1"

	BSSRelpathBootParams = "/bootparameters"
	BSSRelpathBootScript = "/bootscript"
)

// NewBSSClient takes a baseURI and basePath and returns a pointer to a new
// BSSClient. If an error occurred creating the embedded OchamiClient, it is
// returned. If insecure is true, TLS certificates will not be verified.
func NewBSSClient(baseURI string, insecure bool) (*BSSClient, error) {
	oc, err := NewOchamiClient(serviceNameBSS, baseURI, basePathBSS, insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create OchamiClient for %s: %v", serviceNameBSS, err)
	}
	bc := &BSSClient{
		OchamiClient: oc,
	}

	return bc, err
}

// PostBootParams is a wrapper function around BSSClient.PostData that takes a
// bssTypes.BootParams struct (bp) and a token, puts the token in the request
// headers as an authorization bearer, marshals bp as JSON and sets it as the
// request body, then passes it to BSSClient.PostData.
func (bc *BSSClient) PostBootParams(bp bssTypes.BootParams, token string) (HTTPEnvelope, error) {
	var (
		henv    HTTPEnvelope
		headers *HTTPHeaders
		body    HTTPBody
		err     error
	)
	if body, err = json.Marshal(bp); err != nil {
		return henv, fmt.Errorf("PostBootParams(): failed to marshal BootParams: %v", err)
	}
	headers = NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("PostBootParams(): error setting token in HTTP headers")
		}
	}
	henv, err = bc.PostData(BSSRelpathBootParams, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PostBootParams(): failed to POST boot parameters to BSS: %v", err)
	}

	return henv, err
}

// PutBootParams is a wrapper function around BSSClient.PutData that takes a
// bssTypes.BootParams struct (bp) and a token, puts token in the request
// headers as an authorization bearer, marshals bp as JSON and sets it as the
// request body, then passes it to BSSClient.PutData.
func (bc *BSSClient) PutBootParams(bp bssTypes.BootParams, token string) (HTTPEnvelope, error) {
	var (
		henv    HTTPEnvelope
		headers *HTTPHeaders
		body    HTTPBody
		err     error
	)
	if body, err = json.Marshal(bp); err != nil {
		return henv, fmt.Errorf("PutBootParams(): failed to marshal BootParams: %v", err)
	}
	headers = NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("PutBootParams(): error setting token in HTTP headers")
		}
	}
	henv, err = bc.PutData(BSSRelpathBootParams, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PutBootParams(): failed to PUT boot parameters to BSS: %v", err)
	}

	return henv, err
}

// PatchBootParams is a wrapper function around BSSClient.PatchData that takes a
// bssTypes.BootParams struct (bp) and a token, puts token in the request
// headers as an authorization bearer, marshals bp as JSON and sets it as the
// request body, then passes it to BSSClient.PatchData.
func (bc *BSSClient) PatchBootParams(bp bssTypes.BootParams, token string) (HTTPEnvelope, error) {
	var (
		henv    HTTPEnvelope
		headers *HTTPHeaders
		body    HTTPBody
		err     error
	)
	if body, err = json.Marshal(bp); err != nil {
		return henv, fmt.Errorf("PatchBootParams(): failed to marshal BootParams: %v", err)
	}
	headers = NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("PatchBootParams(): error setting token in HTTP headers")
		}
	}
	henv, err = bc.PatchData(BSSRelpathBootParams, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PatchBootParams(): failed to PATCH boot parameters to BSS: %v", err)
	}

	return henv, err
}

// DeleteBootParams is a wrapper function around BSSClient.DeleteData that takes
// a bssTypes.BootParams struct (bp) and a token, puts token in the request
// headers as an authorization bearer, marshals bp as JSON and sets it as the
// request body, then passes it to BSSClient.DeleteData.
func (bc *BSSClient) DeleteBootParams(bp bssTypes.BootParams, token string) (HTTPEnvelope, error) {
	var (
		henv    HTTPEnvelope
		headers *HTTPHeaders
		body    HTTPBody
		err     error
	)
	if body, err = json.Marshal(bp); err != nil {
		return henv, fmt.Errorf("DeleteBootParams(): failed to marshal BootParams: %v", err)
	}
	headers = NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("DeleteBootParams(): error setting token in HTTP headers")
		}
	}
	henv, err = bc.DeleteData(BSSRelpathBootParams, "", headers, body)
	if err != nil {
		err = fmt.Errorf("DeleteBootParams(): failed to DELETE boot parameters to BSS: %v", err)
	}

	return henv, err
}

// GetBootParams is a wrapper function around BSSClient.GetData that takes an
// optional query string (without the "?") and a token. It sets token as the
// authorization bearer in the headers and passes the query string and headers
// to BSSClient.GetData, using /bootparameters as the API endpoint.
func (bc *BSSClient) GetBootParams(query, token string) (HTTPEnvelope, error) {
	var (
		henv    HTTPEnvelope
		headers *HTTPHeaders
		err     error
	)
	headers = NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetBootParams(): error setting token in HTTP headers")
		}
	}
	henv, err = bc.GetData(BSSRelpathBootParams, query, headers)
	if err != nil {
		err = fmt.Errorf("GetBootParams(): error getting boot parameters: %v", err)
	}

	return henv, err
}

// GetBootScript is a wrapper function around BSSClient.GetData that takes a
// query string (without the "?") and passes it to BSSClient.GetData, using
// /bootscript as the API endpoint.
func (bc *BSSClient) GetBootScript(query string) (HTTPEnvelope, error) {
	henv, err := bc.GetData(BSSRelpathBootScript, query, nil)
	if err != nil {
		err = fmt.Errorf("GetBootScript(): error getting boot script: %v", err)
	}

	return henv, err
}
