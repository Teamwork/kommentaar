// Package yamlout writes YAML that YAML 1.1 parsers, such as PyYAML, read back
// as the same values.
package yamlout

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// Marshal encodes v as YAML with an indent of two spaces.
//
// yaml.v3 quotes the YAML 1.1 booleans and sexagesimal numbers, but leaves the
// strings "=" and "<<" plain, which parsers read as a !!value and a !!merge.
func Marshal(v any) ([]byte, error) {
	b, err := encode(v)
	if err != nil {
		return nil, err
	}

	// Not Node.Encode: it emits at indent 4, where yaml.v3 writes a block
	// scalar with leading spaces under a wrong indentation indicator.
	var n yaml.Node
	if err := yaml.Unmarshal(b, &n); err != nil {
		return nil, err
	}
	if !quoteIndicators(&n) {
		return b, nil
	}
	return encode(&n)
}

func encode(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func quoteIndicators(n *yaml.Node) bool {
	quoted := false
	if n.Kind == yaml.ScalarNode && n.Style == 0 &&
		(n.Tag == "!!str" && n.Value == "=" || n.Tag == "!!merge") {
		n.Tag = "!!str"
		n.Style = yaml.DoubleQuotedStyle
		quoted = true
	}
	for _, c := range n.Content {
		if quoteIndicators(c) {
			quoted = true
		}
	}
	return quoted
}
