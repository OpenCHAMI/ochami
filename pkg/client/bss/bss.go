package bss

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/OpenCHAMI/bss/pkg/bssTypes"
	"github.com/OpenCHAMI/ochami/pkg/client"
)

const (
	serviceNameBSS = "BSS"

	BSSRelpathBootParams      = "/bootparameters"
	BSSRelpathBootScript      = "/bootscript"
	BSSRelpathService         = "/service"
	BSSRelpathDumpState       = "/dumpstate"
	BSSRelpathEndpointHistory = "/endpoint-history"
	BSSRelpathHosts           = "/hosts"
)

// BSSClient is an OchamiClient that has its BasePath set configured to the one
// that BSS uses.
type BSSClient struct {
	*client.OchamiClient
}

// NewClient takes a baseURI and returns a pointer to a new BSSClient. If an
// error occurred creating the embedded OchamiClient, it is returned. If
// insecure is true, TLS certificates will not be verified.
func NewClient(baseURI string, insecure bool) (*BSSClient, error) {
	oc, err := client.NewOchamiClient(serviceNameBSS, baseURI, insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create OchamiClient for %s: %w", serviceNameBSS, err)
	}
	bc := &BSSClient{
		OchamiClient: oc,
	}

	return bc, err
}

// PostBootParams is a wrapper function around OchamiClient.PostData that takes a
// bssTypes.BootParams struct (bp) and a token, puts the token in the request
// headers as an authorization bearer, marshals bp as JSON and sets it as the
// request body, then passes it to OchamiClient.PostData.
func (bc *BSSClient) PostBootParams(bp bssTypes.BootParams, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		body    client.HTTPBody
		err     error
	)
	if body, err = json.Marshal(bp); err != nil {
		return henv, fmt.Errorf("PostBootParams(): failed to marshal BootParams: %w", err)
	}
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("PostBootParams(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = bc.PostData(BSSRelpathBootParams, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PostBootParams(): failed to POST boot parameters to BSS: %w", err)
	}

	return henv, err
}

// PutBootParams is a wrapper function around OchamiClient.PutData that takes a
// bssTypes.BootParams struct (bp) and a token, puts token in the request
// headers as an authorization bearer, marshals bp as JSON and sets it as the
// request body, then passes it to OchamiClient.PutData.
func (bc *BSSClient) PutBootParams(bp bssTypes.BootParams, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		body    client.HTTPBody
		err     error
	)
	if body, err = json.Marshal(bp); err != nil {
		return henv, fmt.Errorf("PutBootParams(): failed to marshal BootParams: %w", err)
	}
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("PutBootParams(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = bc.PutData(BSSRelpathBootParams, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PutBootParams(): failed to PUT boot parameters to BSS: %w", err)
	}

	return henv, err
}

// PatchBootParams is a wrapper function around OchamiClient.PatchData that
// takes a bssTypes.BootParams struct (bp) and a token, puts token in the
// request headers as an authorization bearer, marshals bp as JSON and sets it
// as the request body, then passes it to OchamiClient.PatchData.
func (bc *BSSClient) PatchBootParams(bp bssTypes.BootParams, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		body    client.HTTPBody
		err     error
	)
	if body, err = json.Marshal(bp); err != nil {
		return henv, fmt.Errorf("PatchBootParams(): failed to marshal BootParams: %w", err)
	}
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("PatchBootParams(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = bc.PatchData(BSSRelpathBootParams, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PatchBootParams(): failed to PATCH boot parameters to BSS: %w", err)
	}

	return henv, err
}

// DeleteBootParams is a wrapper function around OchamiClient.DeleteData that
// takes a bssTypes.BootParams struct (bp) and a token, puts token in the
// request headers as an authorization bearer, marshals bp as JSON and sets it
// as the request body, then passes it to OchamiClient.DeleteData.
func (bc *BSSClient) DeleteBootParams(bp bssTypes.BootParams, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		body    client.HTTPBody
		err     error
	)
	if body, err = json.Marshal(bp); err != nil {
		return henv, fmt.Errorf("DeleteBootParams(): failed to marshal BootParams: %w", err)
	}
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("DeleteBootParams(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = bc.DeleteData(BSSRelpathBootParams, "", headers, body)
	if err != nil {
		err = fmt.Errorf("DeleteBootParams(): failed to DELETE boot parameters to BSS: %w", err)
	}

	return henv, err
}

// GetBootParams is a wrapper function around OchamiClient.GetData that takes an
// optional query string (without the "?") and a token. It sets token as the
// authorization bearer in the headers and passes the query string and headers
// to OchamiClient.GetData, using /bootparameters as the API endpoint.
func (bc *BSSClient) GetBootParams(query, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetBootParams(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = bc.GetData(BSSRelpathBootParams, query, headers)
	if err != nil {
		err = fmt.Errorf("GetBootParams(): error getting boot parameters: %w", err)
	}

	return henv, err
}

// GetBootScript is a wrapper function around OchamiClient.GetData that takes a
// query string (without the "?") and passes it to OchamiClient.GetData, using
// /bootscript as the API endpoint.
func (bc *BSSClient) GetBootScript(query string) (client.HTTPEnvelope, error) {
	henv, err := bc.GetData(BSSRelpathBootScript, query, nil)
	if err != nil {
		err = fmt.Errorf("GetBootScript(): error getting boot script: %w", err)
	}

	return henv, err
}

// GetStatus is a wrapper function around OchamiClient.GetData that takes an
// optional component and uses it to determine which subpath of the BSS /service
// endpoint to query. If empty, the /service/status endpoint is queried.
// Otherwise:
//
// "all"     -> "/service/status/all"
// "storage" -> "/service/storage/status"
// "smd"     -> "/service/hsm"
// "version" -> "/service/version"
func (bc *BSSClient) GetStatus(component string) (client.HTTPEnvelope, error) {
	var (
		henv              client.HTTPEnvelope
		err               error
		bssStatusEndpoint string
	)
	switch component {
	case "":
		bssStatusEndpoint = path.Join(BSSRelpathService, "status")
	case "all":
		bssStatusEndpoint = path.Join(BSSRelpathService, "status/all")
	case "storage":
		bssStatusEndpoint = path.Join(BSSRelpathService, "storage/status")
	case "smd":
		bssStatusEndpoint = path.Join(BSSRelpathService, "hsm")
	case "version":
		bssStatusEndpoint = path.Join(BSSRelpathService, "version")
	default:
		return henv, fmt.Errorf("GetStatus(): unknown status component: %s", component)
	}

	henv, err = bc.GetData(bssStatusEndpoint, "", nil)
	if err != nil {
		err = fmt.Errorf("GetStatus(): error getting BSS all status: %w", err)
	}

	return henv, err
}

// GetDumpState is a wrapper function around OchamiClient.GetData that queries the
// /dumpstate endpoint and returns its response and an error, if one occurred.
func (bc *BSSClient) GetDumpState() (client.HTTPEnvelope, error) {
	henv, err := bc.GetData(BSSRelpathDumpState, "", nil)
	if err != nil {
		err = fmt.Errorf("GetDumpState(): error getting dump state: %w", err)
	}

	return henv, err
}

// GetEndpointHistory is a wrapper function around OchamiClient.GetData that
// queries /endpoint-history and appends an optional query string (without the
// "?").
func (bc *BSSClient) GetEndpointHistory(query string) (client.HTTPEnvelope, error) {
	henv, err := bc.GetData(BSSRelpathEndpointHistory, query, nil)
	if err != nil {
		err = fmt.Errorf("GetEndpointHistory(): error getting endpoint history: %w", err)
	}

	return henv, err
}

// GetHosts is a wrapper function around OchamiClient.GetData that queries /hosts
// and appends an optional query string (without the "?").
func (bc *BSSClient) GetHosts(query string) (client.HTTPEnvelope, error) {
	henv, err := bc.GetData(BSSRelpathHosts, query, nil)
	if err != nil {
		err = fmt.Errorf("GetHosts(): error getting hosts: %w", err)
	}

	return henv, err
}
