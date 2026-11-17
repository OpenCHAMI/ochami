// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"fmt"
	"net/netip"
	"strings"

	gj "github.com/jryberg/gojinja"
)

// nocloudPlatform is the platform cloud-init's NoCloud datasource reports
// outside LXD.
const nocloudPlatform = "nocloud"

// metadataUnknown is cloud-init's sources.METADATA_UNKNOWN.
const metadataUnknown = "unknown"

// cloudIDRegionPrefixes mirrors sources.CLOUD_ID_REGION_PREFIX_MAP, in order.
var cloudIDRegionPrefixes = []struct {
	prefix, cloudID, cloudName string
}{
	{"cn-", "aws-china", "aws"},
	{"us-gov-", "aws-gov", "aws"},
	{"china", "azure-china", "azure"},
}

// buildInstanceData returns the instance data cloud-init would write for req,
// before convert_jinja_instance_data. InstanceData is used as given. Otherwise
// the data is what the NoCloud datasource writes for Metadata, including the
// standardized v1 keys that can be derived from metadata. ExtraVars then
// replace top-level keys.
func buildInstanceData(req RenderRequest) (*gj.OrderedDict, error) {
	var data *gj.OrderedDict
	var err error
	if len(req.InstanceData.Data) > 0 {
		if len(req.Metadata.Data) > 0 {
			return nil, &RenderError{Stage: StageInstanceData, Err: fmt.Errorf("instance data and metadata are mutually exclusive")}
		}
		if data, err = decodeDocument(req.InstanceData); err != nil {
			return nil, &RenderError{Stage: StageInstanceData, Err: err}
		}
	} else {
		metadata, err := decodeDocument(req.Metadata)
		if err != nil {
			return nil, &RenderError{Stage: StageMetadata, Err: err}
		}
		data = nocloudInstanceData(metadata)
	}
	extra, err := decodeDocument(req.ExtraVars)
	if err != nil {
		return nil, &RenderError{Stage: StageExtraVars, Err: err}
	}
	for _, key := range extra.Keys() {
		value, _ := extra.Get(key)
		data.Set(key, value)
	}
	return jsonRoundTrip(data).(*gj.OrderedDict), nil
}

// nocloudInstanceData builds the instance data that cloud-init's NoCloud
// datasource writes for metadata. Keys that only a booted node knows, such as
// v1.distro and sys_info, are omitted.
func nocloudInstanceData(metadata *gj.OrderedDict) *gj.OrderedDict {
	ds := gj.NewOrderedDict()
	ds.Set("meta_data", metadata)
	data := gj.NewOrderedDict()
	data.Set("ds", ds)
	data.Set("v1", standardizedMetadata(metadata))
	return data
}

// standardizedMetadata mirrors DataSource._get_standardized_metadata for the
// NoCloud datasource, limited to keys derived from metadata.
func standardizedMetadata(md *gj.OrderedDict) *gj.OrderedDict {
	az := availabilityZone(md)
	name := cloudName(md)
	region, _ := md.Get("region")
	var instanceID any = "iid-datasource"
	if id, ok := md.Get("instance-id"); ok {
		instanceID = id
	}

	v1 := gj.NewOrderedDict()
	v1.Set("_beta_keys", []any{"subplatform"})
	v1.Set("availability-zone", az)
	v1.Set("availability_zone", az)
	v1.Set("cloud_id", canonicalCloudID(name, region, nocloudPlatform))
	v1.Set("cloud-name", name)
	v1.Set("cloud_name", name)
	v1.Set("instance-id", instanceID)
	v1.Set("instance_id", instanceID)
	if hostname, ok := localHostname(md); ok {
		v1.Set("local-hostname", hostname)
		v1.Set("local_hostname", hostname)
	}
	v1.Set("platform", nocloudPlatform)
	v1.Set("public_ssh_keys", publicKeys(md))
	v1.Set("region", region)
	return v1
}

// availabilityZone mirrors DataSource.availability_zone.
func availabilityZone(md *gj.OrderedDict) any {
	top, ok := md.Get("availability-zone")
	if !ok {
		top, _ = md.Get("availability_zone")
	}
	if pyTruthy(top) {
		return top
	}
	if placement, ok := md.Get("placement"); ok {
		if mapping, ok := placement.(*gj.OrderedDict); ok {
			value, _ := mapping.Get("availability-zone")
			return value
		}
	}
	return nil
}

// cloudName mirrors DataSource.cloud_name. NoCloud reports unknown unless
// metadata names the cloud.
func cloudName(md *gj.OrderedDict) string {
	if name, ok := md.Get("cloud-name"); ok && pyTruthy(name) {
		if s, ok := name.(string); ok {
			return strings.ToLower(s)
		}
	}
	return metadataUnknown
}

// canonicalCloudID mirrors sources.canonical_cloud_id.
func canonicalCloudID(name string, region any, platform string) string {
	if name == "" {
		name = metadataUnknown
	}
	r, _ := region.(string)
	if r == "" {
		r = metadataUnknown
	}
	if r == metadataUnknown {
		if name != metadataUnknown {
			return name
		}
		return platform
	}
	for _, p := range cloudIDRegionPrefixes {
		if strings.HasPrefix(r, p.prefix) && name == p.cloudName {
			return p.cloudID
		}
	}
	if name != metadataUnknown {
		return name
	}
	return platform
}

// localHostname mirrors DataSource.get_hostname with metadata that names the
// host. Without local-hostname a node falls back to its own hostname, which
// ochami cannot know.
func localHostname(md *gj.OrderedDict) (string, bool) {
	value, _ := md.Get("local-hostname")
	lhost, ok := value.(string)
	if !ok || lhost == "" {
		return "", false
	}
	if addr, err := netip.ParseAddr(lhost); err == nil && addr.Is4() {
		return "ip-" + strings.ReplaceAll(lhost, ".", "-"), true
	}
	host, _, _ := strings.Cut(lhost, ".")
	return host, true
}

// publicKeys mirrors sources.normalize_pubkey_data for public-keys.
func publicKeys(md *gj.OrderedDict) []any {
	value, _ := md.Get("public-keys")
	keys := []any{}
	if !pyTruthy(value) {
		return keys
	}
	switch v := value.(type) {
	case string:
		for _, line := range pySplitlines(v) {
			keys = append(keys, line)
		}
	case []any:
		keys = append(keys, v...)
	case *gj.OrderedDict:
		for _, name := range v.Keys() {
			klist, _ := v.Get(name)
			if s, ok := klist.(string); ok {
				klist = []any{s}
			}
			if items, ok := klist.([]any); ok {
				for _, item := range items {
					if pyTruthy(item) {
						keys = append(keys, item)
					}
				}
			}
		}
	}
	return keys
}
