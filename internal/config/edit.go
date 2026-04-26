package config

import (
	"bytes"
	"os"

	"github.com/difof/errors"
	"gopkg.in/yaml.v3"
)

func SetTheme(name string) (string, error) {
	loaded, err := LoadEffective()
	if err != nil {
		return "", errors.Wrap(err)
	}
	if !loaded.UserExists {
		if err := WriteDefaultsForTheme(loaded.Path, name); err != nil {
			return "", errors.Wrap(err)
		}
		return loaded.Path, nil
	}

	updated, err := updateThemeBytes(loaded.ConfigBytes, name)
	if err != nil {
		return "", errors.Wrap(err)
	}
	if err := os.WriteFile(loaded.Path, updated, 0o644); err != nil {
		return "", errors.Wrap(err)
	}
	return loaded.Path, nil
}

func updateThemeBytes(data []byte, name string) ([]byte, error) {
	var doc yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&doc); err != nil {
		return nil, errors.Wrap(err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("config root must be a YAML mapping")
	}

	root := doc.Content[0]
	updated := false
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i]
		if key.Value != "theme" {
			continue
		}
		root.Content[i+1].Kind = yaml.ScalarNode
		root.Content[i+1].Tag = "!!str"
		root.Content[i+1].Value = name
		updated = true
		break
	}
	if !updated {
		key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "theme"}
		value := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}
		root.Content = append([]*yaml.Node{key, value}, root.Content...)
	}

	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return nil, errors.Wrap(err)
	}
	if err := encoder.Close(); err != nil {
		return nil, errors.Wrap(err)
	}
	return buffer.Bytes(), nil
}
