package ci

import (
	"fmt"
	"net/url"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
)

// CIDataType is an enum that represents the types of cloud-init data: user,
// meta, and vendor.
type CIDataType string

// CloudInitClient is an OchamiClient that has its BasePath configured to the
// one that the cloud-init service uses.
type CloudInitClient struct {
	*client.OchamiClient
}

const (
	serviceNameCloudInit = "cloud-init"
)

// The different types of cloud-init data.
const (
	CloudInitUserData   CIDataType = "user-data"
	CloudInitMetaData   CIDataType = "meta-data"
	CloudInitVendorData CIDataType = "vendor-data"
)

// NewClient takes a baseURI and returns a pointer to a new CloudInitClient. If
// an error occurred creating the embedded OchamiClient, it is returned. If
// insecure is true, TLS certificates will not be verified.
func NewClient(baseURI string, insecure bool) (*CloudInitClient, error) {
	oc, err := client.NewOchamiClient(serviceNameCloudInit, baseURI, insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create OchamiClient for %s: %w", serviceNameCloudInit, err)
	}
	cic := &CloudInitClient{
		OchamiClient: oc,
	}

	return cic, err
}
