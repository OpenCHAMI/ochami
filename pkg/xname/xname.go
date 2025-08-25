package xname

import (
	"fmt"

	"github.com/openchami/smd/pkg/schemas/csm"
)

func XNameComponentsToString(x csm.XNameComponents) string {
	switch x.Type {
	case "n":
		return fmt.Sprintf("x%dc%ds%db%dn%d", x.Cabinet, x.Chassis, x.Slot, x.BMCPosition, x.NodePosition)
	case "b":
		return fmt.Sprintf("x%dc%ds%db%d", x.Cabinet, x.Chassis, x.Slot, x.BMCPosition)
	}
	return ""
}

func StringToXname(xname string) csm.XNameComponents {
	var components csm.XNameComponents
	_, err := fmt.Sscanf(xname, "x%dc%ds%db%dn%d", &components.Cabinet, &components.Chassis, &components.Slot, &components.BMCPosition, &components.NodePosition)
	if err == nil {
		components.Type = "n"
		return components
	}
	_, err = fmt.Sscanf(xname, "x%dc%ds%db%d", &components.Cabinet, &components.Chassis, &components.Slot, &components.BMCPosition)
	if err == nil {
		components.Type = "b"
		return components
	}
	return components
}

func NodeXnameToBMCXname(xname string) (string, error) {
	bmcXname := StringToXname(xname)
	bmcXname.Type = "b"
	bmcXnameStr := XNameComponentsToString(bmcXname)
	if !csm.IsValidBMCXName(bmcXnameStr) {
		return "", fmt.Errorf("xname %s not a valid BMC xname", xname)
	}
	return bmcXnameStr, nil
}
