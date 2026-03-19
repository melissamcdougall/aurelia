package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Node represents a remote aurelia daemon peer.
type Node struct {
	Name      string `yaml:"name"`
	Addr      string `yaml:"addr"`                // e.g. "aurelia.local:9090"
	Token     string `yaml:"token,omitempty"`      // inline token
	TokenFile string `yaml:"token_file,omitempty"` // path to token file
}

// LoadToken returns the bearer token for this node, reading from file if needed.
func (n Node) LoadToken() (string, error) {
	if n.Token != "" {
		return n.Token, nil
	}
	if n.TokenFile != "" {
		data, err := os.ReadFile(n.TokenFile)
		if err != nil {
			return "", fmt.Errorf("reading token file for node %s: %w", n.Name, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return "", fmt.Errorf("no token configured for node %s", n.Name)
}

// Config holds persistent daemon configuration loaded from ~/.aurelia/config.yaml.
type Config struct {
	RoutingOutput string `yaml:"routing_output"`
	APIAddr       string `yaml:"api_addr"`
	NodeName      string `yaml:"node_name,omitempty"`
	Nodes         []Node `yaml:"nodes,omitempty"`
}

// FindNode returns the node with the given name, or false if not found.
func (c *Config) FindNode(name string) (Node, bool) {
	for _, n := range c.Nodes {
		if n.Name == name {
			return n, true
		}
	}
	return Node{}, false
}

// DefaultPath returns the default config file path: ~/.aurelia/config.yaml.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".aurelia", "config.yaml")
}

// Load reads a YAML config file from path. If the file does not exist,
// it returns an empty Config and no error. An empty or all-comment file
// also returns an empty Config with no error.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.RoutingOutput = os.ExpandEnv(cfg.RoutingOutput)
	cfg.APIAddr = os.ExpandEnv(cfg.APIAddr)
	return cfg, nil
}
