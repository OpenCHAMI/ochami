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
	DisocveryVersionHelp = map[int]string{
		int(DiscoveryMethodV1): "One-line JSON format",
		int(DiscoveryMethodV2): "Unindented JSON format",
	}
)

func (dv DiscoveryVersion) String() string {
	return fmt.Sprintf("v%d", (dv))
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
	return "DiscoveryVersion "
}
