package models

import (
	"encoding/json"
	"log"

	"gopkg.in/yaml.v2"
)

type PermissionsConfig struct {
	Routes    []RoutePermission    `yaml:"routes" json:"routes"`
	Templates []TemplatePermission `yaml:"templates" json:"templates"`
}

// String returns the string representation of the PermissionsConfig in JSON format
func (pc *PermissionsConfig) String() string {
	data, err := yaml.Marshal(pc)
	if err != nil {
		log.Fatalf("Failed to marshal permissions config: %v", err)
	}
	return string(data)
}

type RoutePermission struct {
	Path               string   `yaml:"path" json:"path"`
	Methods            []string `yaml:"methods" json:"methods"`
	RequiredPermission string   `yaml:"required_permission" json:"required_permission"`
}

func (rp *RoutePermission) String() string {
	data, err := json.Marshal(rp)
	if err != nil {
		log.Fatalf("Failed to marshal route permission: %v", err)
	}
	return string(data)
}

type TemplatePermission struct {
	Template string              `yaml:"template" json:"template"`
	Elements []ElementPermission `yaml:"elements" json:"elements"`
}

type ElementPermission struct {
	ID                 string `yaml:"id" json:"id"`
	RequiredPermission string `yaml:"required_permission" json:"required_permission"`
}

func LoadPermissionsFromBytes(data []byte) (*PermissionsConfig, error) {
	var config PermissionsConfig
	err := yaml.Unmarshal(data, &config)
	return &config, err
}
