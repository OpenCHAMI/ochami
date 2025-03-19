// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package pcs

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/OpenCHAMI/ochami/pkg/client"
)

const (
	serviceNamePCS = "PCS"

	PCSRelpathLiveness  = "/liveness"
	PCSRelpathReadiness = "/readiness"
	PCSRelpathHealth    = "/health"
	PCSTransitions      = "/transitions"
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

type transitionBody struct {
	Operation    string          `json:"operation"`
	TaskDeadline *int            `json:"taskDeadlineMinutes"`
	Location     []locationEntry `json:"location"`
}

type locationEntry struct {
	Xname string `json:"xname"`
}

// CreateTransition is a wrapper function around OchamiClient.PostData to
// hit the /transitions endpoint
func (pc *PCSClient) CreateTransition(operation string, taskDeadline *int, xnames []string, token string) (client.HTTPEnvelope, error) {
	var henv client.HTTPEnvelope

	headers := client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("CreateTransition(): error setting token in HTTP headers: %w", err)
		}
	}

	// Create the request body
	location := []locationEntry{}
	for i := 0; i < len(xnames); i++ {
		location = append(location, locationEntry{Xname: xnames[i]})
	}

	body := transitionBody{
		Operation:    operation,
		TaskDeadline: taskDeadline,
		Location:     location,
	}

	// Marshal the transition body
	bytes, err := json.Marshal(body)
	if err != nil {
		return henv, fmt.Errorf("CreateTransition(): failed to marshal body into JSON: %w", err)
	}

	// Now create the HTTPBody
	httpBody, err := client.BytesToHTTPBody(bytes, "json")
	if err != nil {
		return henv, fmt.Errorf("CreateTransition(): failed to create HTTPBody: %w", err)
	}

	henv, err = pc.PostData(PCSTransitions, "", headers, httpBody)
	if err != nil {
		err = fmt.Errorf("CreateTransition(): error creating PCS health: %w", err)
	}

	return henv, err
}

// GetTransitions is a wrapper function around OchamiClient.GetData to
// hit the /transitions endpoint
func (pc *PCSClient) GetTransitions(token string) (client.HTTPEnvelope, error) {
	var (
		henv client.HTTPEnvelope
		err  error
	)

	headers := client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetTransitions(): error setting token in HTTP headers: %w", err)
		}
	}

	henv, err = pc.GetData(PCSTransitions, "", headers)
	if err != nil {
		err = fmt.Errorf("GetTransitions(): error getting PCS transitions: %w", err)
	}

	return henv, err
}

// GetTransitions is a wrapper function around OchamiClient.GetData to
// hit the /transitions/{transitionID} endpoint
func (pc *PCSClient) GetTransition(id string, token string) (client.HTTPEnvelope, error) {
	var (
		henv                   client.HTTPEnvelope
		err                    error
		pcsTransitionsEndpoint string
	)

	headers := client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetTransition(): error setting token in HTTP headers: %w", err)
		}
	}

	pcsTransitionsEndpoint, err = url.JoinPath(PCSTransitions, id)
	if err != nil {
		err = fmt.Errorf("GetTransition(): error joining PCS transitions endpoint: %w", err)
		return henv, err
	}

	henv, err = pc.GetData(pcsTransitionsEndpoint, "", headers)
	if err != nil {
		err = fmt.Errorf("GetTransition(): error getting PCS transition: %w", err)
	}

	return henv, err
}

// DeleteTransitions is a wrapper function around OchamiClient.DeleteData to
// hit the /transitions/{transitionID} endpoint
func (pc *PCSClient) DeleteTransition(id string, token string) (client.HTTPEnvelope, error) {
	var (
		henv                  client.HTTPEnvelope
		err                   error
		pcsTransitionEndpoint string
	)

	headers := client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("DeleteTransition(): error setting token in HTTP headers: %w", err)
		}
	}

	pcsTransitionEndpoint, err = url.JoinPath(PCSTransitions, id)
	if err != nil {
		err = fmt.Errorf("DeleteTransition(): error joining PCS transition endpoint: %w", err)
		return henv, err
	}

	henv, err = pc.DeleteData(pcsTransitionEndpoint, "", headers, nil)
	if err != nil {
		err = fmt.Errorf("DeleteTransition(): error deleting PCS transition: %w", err)
	}

	return henv, err
}
