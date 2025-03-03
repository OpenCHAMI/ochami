// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package pcs

import (
	"fmt"
	"github.com/OpenCHAMI/ochami/pkg/client"
)

const (
	serviceNamePCS = "PCS"

	PCSRelpathLiveness  = "/liveness"
	PCSRelpathReadiness = "/readiness"
	PCSRelpathHealth    = "/health"
)

// PCSClient is an OchamiClient that has its BasePath set configured to the one
// that PCSClient uses.
type PCSClient struct {
	*client.OchamiClient
}

// NewClient takes a baseURI and returns a pointer to a new PCSClient. If an
// error occurred creating the embedded OchamiClient, it is returned. If
// insecure is true, TLS certificates will not be verified.
func NewClient(baseURI string, insecure bool) (*PCSClient, error) {
	oc, err := client.NewOchamiClient(serviceNamePCS, baseURI, insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create OchamiClient for %s: %w", serviceNamePCS, err)
	}
	bc := &PCSClient{
		OchamiClient: oc,
	}

	return bc, err
}

// GetLiveness is a wrapper function around OchamiClient.GetData to
// hit the /liveness endpoint
func (pc *PCSClient) GetLiveness() (client.HTTPEnvelope, error) {
	var (
		henv client.HTTPEnvelope
		err  error
	)

	henv, err = pc.GetData(PCSRelpathLiveness, "", nil)
	if err != nil {
		err = fmt.Errorf("GetLiveness(): error getting PCS liveness: %w", err)
	}

	return henv, err
}

// GetReadiness is a wrapper function around OchamiClient.GetData to
// hit the /readiness endpoint
func (pc *PCSClient) GetReadiness() (client.HTTPEnvelope, error) {
	var (
		henv client.HTTPEnvelope
		err  error
	)

	henv, err = pc.GetData(PCSRelpathReadiness, "", nil)
	if err != nil {
		err = fmt.Errorf("GetReadiness(): error getting PCS liveness: %w", err)
	}

	return henv, err
}

// GetHealth is a wrapper function around OchamiClient.GetData to
// hit the /health endpoint
func (pc *PCSClient) GetHealth() (client.HTTPEnvelope, error) {
	var (
		henv client.HTTPEnvelope
		err  error
	)

	henv, err = pc.GetData(PCSRelpathHealth, "", nil)
	if err != nil {
		err = fmt.Errorf("GetHealth(): error getting PCS health: %w", err)
	}

	return henv, err
}
