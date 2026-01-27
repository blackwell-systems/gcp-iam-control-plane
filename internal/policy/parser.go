package policy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Policy represents the policy.yaml structure
type Policy struct {
	Roles    map[string]Role       `yaml:"roles"`
	Groups   map[string]Group      `yaml:"groups"`
	Projects map[string]Project    `yaml:"projects"`
}

// Role represents a custom role with permissions
type Role struct {
	Permissions []string `yaml:"permissions"`
}

// Group represents a group with members
type Group struct {
	Members []string `yaml:"members"`
}

// Project represents a project with IAM bindings
type Project struct {
	Bindings []Binding `yaml:"bindings"`
}

// Binding represents an IAM binding
type Binding struct {
	Role      string     `yaml:"role"`
	Members   []string   `yaml:"members"`
	Condition *Condition `yaml:"condition,omitempty"`
}

// Condition represents a CEL condition
type Condition struct {
	Expression  string `yaml:"expression"`
	Title       string `yaml:"title,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// Load loads and parses a policy file
func Load(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy YAML: %w", err)
	}

	return &policy, nil
}

// Save saves policy to file
func Save(policy *Policy, path string) error {
	data, err := yaml.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write policy file: %w", err)
	}

	return nil
}
