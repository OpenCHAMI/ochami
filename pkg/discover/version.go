// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package discover

import (
	"fmt"
	"strconv"
)

type DiscoveryVersion int

const (
	DiscoveryMethodV1 DiscoveryVersion = iota + 1 // 1
	DiscoveryMethodV2                             // 2
)

var (
	DiscoveryVersionHelp = map[int]string{
		int(DiscoveryMethodV1): "Discovery with additional request to add EthernetInterfaces",
		int(DiscoveryMethodV2): "Discovery without request to add EthernetInterfaces",
	}
)

func (dv DiscoveryVersion) String() string {
	return fmt.Sprintf("%d", (dv))
}

func (dv *DiscoveryVersion) Set(v string) error {
	i, err := strconv.Atoi(v)
	if err != nil {
		return err
	}
	switch DiscoveryVersion(i) {
	case DiscoveryMethodV1, DiscoveryMethodV2:
		*dv = DiscoveryVersion(i)
		return nil
	default:
		return fmt.Errorf("must be one of %v", []DiscoveryVersion{
			DiscoveryMethodV1,
			DiscoveryMethodV2,
		})
	}
}

func (dv DiscoveryVersion) Type() string {
	return "DiscoveryVersion"
}
