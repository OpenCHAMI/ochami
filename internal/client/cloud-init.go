package client

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/OpenCHAMI/ochami/internal/log"
)

// CloudInitClient is an OchamiClient that has its BasePath configured to the
// one that the cloud-init service uses.
type CloudInitClient struct {
	*OchamiClient
}

const (
	serviceNameCloudInit = "cloud-init"
	// cloud-init doesn't have a service prefix and has two separate
	// endpoints. To mitigate this, we treat the service root as '/' and use
	// the relative paths as the service endpoints.
	basePathCloudInit      = "/"
	cloudInitRelpathOpen   = "/cloud-init"
	cloudInitRelpathSecure = "/cloud-init-secure"
)

// NewCloudInitClient takes a baseURI and basePath and returns a pointer to a
// new CloudInitClient. If an error occurred creating the embedded
// OchamiClient, it is returned. If insecure is true, TLS certificates will not
// be verified.
func NewCloudInitClient(baseURI string, insecure bool) (*CloudInitClient, error) {
	oc, err := NewOchamiClient(serviceNameCloudInit, baseURI, basePathCloudInit, insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create OchamiClient for %s: %w", serviceNameCloudInit, err)
	}
	cic := &CloudInitClient{
		OchamiClient: oc,
	}

	return cic, err
}

// GetConfigs is a wrapper function around OchamiClient.GetData that determines
// whether to use only the cloud-init base path or it appended with an id and
// calls GetData on the endpoint, returning the result. If an error occurs in
// the function or via HTTP, it is returned as well. If id is blank, all configs
// are returned. Otherwise, just the config for the id is returned.
func (cic *CloudInitClient) GetConfigs(id string) (HTTPEnvelope, error) {
	finalEP := cloudInitRelpathOpen
	if id != "" {
		var err error
		finalEP, err = url.JoinPath(cloudInitRelpathOpen, id)
		if err != nil {
			return HTTPEnvelope{}, fmt.Errorf("GetConfigs(): failed to join cloud-init open path (%s) with id %s: %w", cloudInitRelpathOpen, id, err)
		}
	}
	henv, err := cic.GetData(finalEP, "", nil)
	if err != nil {
		err = fmt.Errorf("GetConfigs(): error getting cloud-init configs: %w", err)
	}

	return henv, err
}

// GetConfigsSecure is like GetConfigs except that it uses the secure cloud-init
// endpoint and thus requires a token.
func (cic *CloudInitClient) GetConfigsSecure(id, token string) (HTTPEnvelope, error) {
	headers := NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return HTTPEnvelope{}, fmt.Errorf("GetConfigsSecure(): error setting token in HTTP headers")
		}
	}
	finalEP := cloudInitRelpathSecure
	if id != "" {
		var err error
		finalEP, err = url.JoinPath(cloudInitRelpathSecure, id)
		if err != nil {
			return HTTPEnvelope{}, fmt.Errorf("GetConfigsSecure(): failed to join cloud-init secure path (%s) with id %s: %w", cloudInitRelpathSecure, id, err)
		}
	}
	henv, err := cic.GetData(finalEP, "", headers)
	if err != nil {
		err = fmt.Errorf("GetConfigsSecure(): error getting secure cloud-init configs: %w", err)
	}

	return henv, err
}

// PostConfigs is a wrapper function around OchamiClient.PostData that takes a
// slice of citypes.CI structs and a token. It iteratively passes these to
// PostData and returns an HTTPEnvelope and error for each, contained within
// separate slices. If an error in the function itself occurs, a separate error
// is returned.
func (cic *CloudInitClient) PostConfigs(data []citypes.CI, token string) ([]HTTPEnvelope, []error, error) {
	var (
		henvs   []HTTPEnvelope
		headers *HTTPHeaders
		body    HTTPBody
		errors  []error
	)
	if len(data) == 0 {
		return nil, []error{}, fmt.Errorf("PostConfigs(): no data passed")
	}
	headers = NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return nil, []error{}, fmt.Errorf("PostConfigs(): error setting token in HTTP headers")
		}
	}
	for _, ciData := range data {
		var err error
		body, err = json.Marshal(ciData)
		if err != nil {
			newErr := fmt.Errorf("PostConfigs(): failed to marshal open cloud-init data for %s: %w", ciData.Name, err)
			henvs = append(henvs, HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := cic.PostData(cloudInitRelpathOpen, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PostConfigs(): error posting open cloud-init config %s: %w", ciData.Name, err)
			log.Logger.Debug().Err(err).Msgf("failed to add open cloud-init config %s", ciData.Name)
			errors = append(errors, newErr)
			continue
		}
		log.Logger.Debug().Msgf("successfully added open cloud-init config %s", ciData.Name)
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PostConfigsSecure is like PostConfigs except that it uses the secure
// cloud-init endpoint.
func (cic *CloudInitClient) PostConfigsSecure(data []citypes.CI, token string) ([]HTTPEnvelope, []error, error) {
	var (
		henvs   []HTTPEnvelope
		headers *HTTPHeaders
		body    HTTPBody
		errors  []error
	)
	if len(data) == 0 {
		return nil, []error{}, fmt.Errorf("PostConfigsSecure(): no data passed")
	}
	headers = NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return nil, []error{}, fmt.Errorf("PostConfigsSecure(): error setting token in HTTP headers")
		}
	}
	for _, ciData := range data {
		var err error
		body, err = json.Marshal(data)
		if err != nil {
			newErr := fmt.Errorf("PostConfigsSecure(): failed to marshal secure cloud-init data for %s: %w", ciData.Name, err)
			henvs = append(henvs, HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := cic.PostData(cloudInitRelpathSecure, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PostConfigsSecure(): error posting secure cloud-init config %s: %w", ciData.Name, err)
			log.Logger.Debug().Err(err).Msgf("failed to add secure cloud-init config %s", ciData.Name)
			errors = append(errors, newErr)
			continue
		}
		log.Logger.Debug().Msgf("successfully added secure cloud-init config %s", ciData.Name)
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PutConfigs is a wrapper function around OchamiClient.PutData that takes a
// slice of citypes.CI structs and a token. It iteratively passes these to
// PutData and returns an HTTPEnvelope and error for each, contained within
// separate slices. If an error in the function itself occurs, a separate error
// is returned.
func (cic *CloudInitClient) PutConfigs(data []citypes.CI, token string) ([]HTTPEnvelope, []error, error) {
	var (
		henvs   []HTTPEnvelope
		headers *HTTPHeaders
		body    HTTPBody
		errors  []error
	)
	if len(data) == 0 {
		return nil, []error{}, fmt.Errorf("PutConfigs(): no data passed")
	}
	headers = NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return nil, []error{}, fmt.Errorf("PutConfigs(): error setting token in HTTP headers")
		}
	}
	for _, ciData := range data {
		if ciData.Name == "" {
			newErr := fmt.Errorf("PutConfigsSecure(): CI.Name field cannot be empty")
			henvs = append(henvs, HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		finalEP, err := url.JoinPath(cloudInitRelpathOpen, ciData.Name)
		if err != nil {
			newErr := fmt.Errorf("PutConfigs(): failed to join cloud-init open path (%s) with cloud-init config ID %s: %w", cloudInitRelpathOpen, ciData.Name, err)
			henvs = append(henvs, HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		body, err = json.Marshal(ciData)
		if err != nil {
			newErr := fmt.Errorf("PutConfigs(): failed to marshal cloud-init data for %s: %w", ciData.Name, err)
			henvs = append(henvs, HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := cic.PutData(finalEP, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PutConfigs(): error putting open cloud-init config %s: %w", ciData.Name, err)
			log.Logger.Debug().Err(err).Msgf("failed to set open cloud-init config %s", ciData.Name)
			errors = append(errors, newErr)
			continue
		}
		log.Logger.Debug().Msgf("successfully set open cloud-init config %s", ciData.Name)
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PutConfigsSecure is like PutConfigs except that it uses the secure cloud-init
// endpoint.
func (cic *CloudInitClient) PutConfigsSecure(data []citypes.CI, token string) ([]HTTPEnvelope, []error, error) {
	var (
		henvs   []HTTPEnvelope
		headers *HTTPHeaders
		body    HTTPBody
		errors  []error
	)
	if len(data) == 0 {
		return nil, []error{}, fmt.Errorf("PutConfigsSecure(): no data passed")
	}
	headers = NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return nil, []error{}, fmt.Errorf("PutConfigsSecure(): error setting token in HTTP headers")
		}
	}
	for _, ciData := range data {
		if ciData.Name == "" {
			newErr := fmt.Errorf("PutConfigsSecure(): CI.Name field cannot be empty")
			henvs = append(henvs, HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		finalEP, err := url.JoinPath(cloudInitRelpathSecure, ciData.Name)
		if err != nil {
			newErr := fmt.Errorf("PutConfigs(): failed to join cloud-init secure path (%s) with cloud-init config ID %s: %w", cloudInitRelpathSecure, ciData.Name, err)
			henvs = append(henvs, HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		body, err = json.Marshal(ciData)
		if err != nil {
			newErr := fmt.Errorf("PutConfigsSecure(): failed to marshal secure cloud-init data for %s: %w", ciData.Name, err)
			henvs = append(henvs, HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := cic.PutData(finalEP, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PutConfigsSecure(): error putting secure cloud-init config %s: %w", ciData.Name, err)
			log.Logger.Debug().Err(err).Msgf("failed to set secure cloud-init config %s", ciData.Name)
			errors = append(errors, newErr)
			continue
		}
		log.Logger.Debug().Msgf("successfully set secure cloud-init config %s", ciData.Name)
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// DeleteConfigs is a wrapper function around OchamiClient.DeleteData that takes
// a token and one or more ids and passes them to DeleteData, using the
// unsecured cloud-init endpoint as the target.
func (cic *CloudInitClient) DeleteConfigs(token string, ids ...string) ([]HTTPEnvelope, []error, error) {
	var (
		henvs   []HTTPEnvelope
		headers *HTTPHeaders
		errors  []error
	)
	if len(ids) == 0 {
		return nil, []error{}, fmt.Errorf("DeleteConfigs(): no ids passed")
	}
	headers = NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return nil, []error{}, fmt.Errorf("DeleteConfigs(): error setting token in HTTP headers")
		}
	}
	for _, id := range ids {
		finalEP, err := url.JoinPath(cloudInitRelpathOpen, id)
		if err != nil {
			newErr := fmt.Errorf("DeleteConfigs(): failed to join cloud-init open path (%s) with cloud-init config ID %s: %w", cloudInitRelpathOpen, id, err)
			henvs = append(henvs, HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := cic.DeleteData(finalEP, "", headers, nil)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("DeleteConfigs(): failed to DELETE cloud-init config %s: %w", id, err)
			log.Logger.Debug().Err(err).Msgf("failed to delete cloud-init config %s", id)
			errors = append(errors, newErr)
			continue
		}
		log.Logger.Debug().Msgf("successfully deleted cloud-init config %s", id)
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// DeleteConfigsSecure is like DeleteConfigs except that it uses the secure
// cloud-init endpoint.
func (cic *CloudInitClient) DeleteConfigsSecure(token string, ids ...string) ([]HTTPEnvelope, []error, error) {
	var (
		henvs   []HTTPEnvelope
		headers *HTTPHeaders
		errors  []error
	)
	if len(ids) == 0 {
		return nil, []error{}, fmt.Errorf("DeleteConfigsSecure(): no ids passed")
	}
	headers = NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return nil, []error{}, fmt.Errorf("DeleteConfigsSecure(): error setting token in HTTP headers")
		}
	}
	for _, id := range ids {
		finalEP, err := url.JoinPath(cloudInitRelpathSecure, id)
		if err != nil {
			newErr := fmt.Errorf("DeleteConfigsSecure(): failed to join cloud-init secure path (%s) with cloud-init config ID %s: %w", cloudInitRelpathSecure, id, err)
			henvs = append(henvs, HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := cic.DeleteData(finalEP, "", headers, nil)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("DeleteConfigsSecure(): failed to DELETE cloud-init config %s: %w", id, err)
			log.Logger.Debug().Err(err).Msgf("failed to delete cloud-init config %s", id)
			errors = append(errors, newErr)
			continue
		}
		log.Logger.Debug().Msgf("successfully deleted cloud-init config %s", id)
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}
