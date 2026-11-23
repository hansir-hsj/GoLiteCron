package golitecron

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFromYAML reads a YAML config; unknown keys are rejected so typos are not ignored.
func LoadFromYAML(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	// An empty document yields io.EOF, which means an empty config.
	if err := decoder.Decode(&config); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	return &config, nil
}

// LoadFromYaml is deprecated. Use LoadFromYAML instead.
func LoadFromYaml(path string) (*Config, error) {
	return LoadFromYAML(path)
}

// LoadFromJSON reads a JSON config; unknown keys are rejected so typos are not ignored.
func LoadFromJSON(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	// Reject trailing content after the first JSON value, as json.Unmarshal does.
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("unexpected data after JSON value")
	}

	return &config, nil
}

// LoadFromJson is deprecated. Use LoadFromJSON instead.
func LoadFromJson(path string) (*Config, error) {
	return LoadFromJSON(path)
}
