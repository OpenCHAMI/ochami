package format

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
)

// DataType represents the supported data formats
type DataFormat string

const (
	DataFormatJson       DataFormat = "json"
	DataFormatJsonPretty DataFormat = "json-pretty"
	DataFormatYaml       DataFormat = "yaml"
)

var (
	DataFormatHelp = map[string]string{
		string(DataFormatJson):       "One-line JSON format",
		string(DataFormatJsonPretty): "Unindented JSON format",
		string(DataFormatYaml):       "YAML format",
	}
)

func (df DataFormat) String() string {
	return string(df)
}

func (df *DataFormat) Set(v string) error {
	switch DataFormat(v) {
	case DataFormatJson,
		DataFormatJsonPretty,
		DataFormatYaml:
		*df = DataFormat(v)
		return nil
	default:
		return fmt.Errorf("must be one of %v", []DataFormat{
			DataFormatJson,
			DataFormatJsonPretty,
			DataFormatYaml,
		})
	}
}

func (df DataFormat) Type() string {
	return "DataFormat"
}

// FormatData marshals arbitrary data into a byte slice formatted as outFormat.
// If a marshalling error occurrs or outFormat is unknown, an error is returned.
//
// Supported values are: json, json-pretty, yaml
func FormatData(data interface{}, outFormat DataFormat) ([]byte, error) {
	switch outFormat {
	case DataFormatJson:
		if bytes, err := json.Marshal(data); err != nil {
			return nil, fmt.Errorf("failed to marshal data into JSON: %w", err)
		} else {
			return bytes, nil
		}
	case DataFormatJsonPretty:
		if bytes, err := json.MarshalIndent(data, "", "  "); err != nil {
			return nil, fmt.Errorf("failed to marshal data into pretty JSON: %w", err)
		} else {
			return bytes, nil
		}
	case DataFormatYaml:
		if bytes, err := yaml.Marshal(data); err != nil {
			return nil, fmt.Errorf("failed to marshal data into YAML: %w", err)
		} else {
			return bytes, nil
		}
	default:
		return nil, fmt.Errorf("unknown data format: %s", outFormat)
	}
}
