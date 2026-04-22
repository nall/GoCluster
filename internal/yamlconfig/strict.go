package yamlconfig

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Path identifies a YAML-owned setting that must be present and non-null.
// Use "*" to require the remaining path under every sequence item.
type Path []string

// DecodeFile reads a YAML file, validates required YAML-owned paths, rejects
// unknown struct fields, and decodes into out.
func DecodeFile(path string, out any, required []Path) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("YAML config path is required")
	}
	bs, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return DecodeBytes(path, bs, out, required)
}

// DecodeBytes validates and decodes a YAML document from memory.
func DecodeBytes(name string, bs []byte, out any, required []Path) error {
	if err := ValidateRequiredPaths(name, bs, required); err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(bs))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}

// ValidateRequiredPaths verifies that each required setting path exists and is
// not YAML null. Values such as false, 0, "", [], and {} are still explicit.
func ValidateRequiredPaths(name string, bs []byte, required []Path) error {
	if len(required) == 0 {
		return nil
	}
	var doc yaml.Node
	dec := yaml.NewDecoder(bytes.NewReader(bs))
	if err := dec.Decode(&doc); err != nil {
		return err
	}
	if len(doc.Content) == 0 {
		return fmt.Errorf("%s: YAML document is empty", name)
	}
	root := doc.Content[0]
	for _, path := range required {
		if len(path) == 0 {
			continue
		}
		if err := requirePath(root, path, nil); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func requirePath(node *yaml.Node, path Path, rendered []string) error {
	if len(path) == 0 {
		if isNull(node) {
			return fmt.Errorf("required YAML setting %q must not be null", renderPath(rendered))
		}
		return nil
	}
	if isNull(node) {
		return fmt.Errorf("required YAML setting %q must not be null", renderPath(rendered))
	}
	part := path[0]
	if part == "*" {
		switch node.Kind {
		case yaml.SequenceNode:
			for i, child := range node.Content {
				itemPath := appendIndex(rendered, i)
				if err := requirePath(child, path[1:], itemPath); err != nil {
					return err
				}
			}
			return nil
		case yaml.MappingNode:
			for i := 0; i+1 < len(node.Content); i += 2 {
				key := node.Content[i].Value
				childPath := append(rendered, key)
				if err := requirePath(node.Content[i+1], path[1:], childPath); err != nil {
					return err
				}
			}
			return nil
		default:
			return fmt.Errorf("required YAML setting %q must be a sequence or mapping", renderPath(rendered))
		}
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("required YAML setting %q is missing", renderPath(append(rendered, part)))
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == part {
			return requirePath(node.Content[i+1], path[1:], append(rendered, part))
		}
	}
	return fmt.Errorf("required YAML setting %q is missing", renderPath(append(rendered, part)))
}

func isNull(node *yaml.Node) bool {
	return node == nil || (node.Kind == yaml.ScalarNode && node.Tag == "!!null")
}

func appendIndex(path []string, index int) []string {
	out := append([]string(nil), path...)
	if len(out) == 0 {
		out = append(out, fmt.Sprintf("[%d]", index))
		return out
	}
	out[len(out)-1] = fmt.Sprintf("%s[%d]", out[len(out)-1], index)
	return out
}

func renderPath(path []string) string {
	if len(path) == 0 {
		return "<root>"
	}
	return strings.Join(path, ".")
}
