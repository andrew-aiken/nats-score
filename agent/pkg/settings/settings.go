package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
)

type StaticConf struct {
	// Add fields for static configuration here
	TeamNumber int16  // TeamNumber
	IPv6Bit    string // IPv6Bit
}

func TemplateDefinition(def any, static StaticConf) ([]byte, error) {
	definitionJSON, err := json.Marshal(def)
	if err != nil {
		return nil, fmt.Errorf("error marshaling definition: %s", err)
	}

	tmpl, err := template.New("").Parse(string(definitionJSON))
	if err != nil {
		return nil, err
	}

	var definitionBuffer bytes.Buffer
	err = tmpl.Execute(&definitionBuffer, static)
	if err != nil {
		return nil, err
	}

	return definitionBuffer.Bytes(), nil
}
