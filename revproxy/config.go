package revproxy

import (
	"fmt"
	"os"
)

type ServerConfig struct {
	Listen listen `yaml:"listen"`
	Apis   api    `yaml:"apis"`
	Auth   auth   `yaml:"auth"`
}

type auth struct {
	AdminKeyEnv *string `yaml:"adminKeyEnv,omitempty"`
}

type listen struct {
	Host string `yaml:"host,omitempty"`
	Port int    `yaml:"port,omitempty"`
}

type api struct {
	AddModels bool `yaml:"addModels"`
	Reload    bool `yaml:"reload"`
}

func (c *ServerConfig) ListenPort() int {
	if c.Listen.Port == 0 {
		return 8080
	}
	return c.Listen.Port
}

func (c *ServerConfig) ListenHost() string {
	if len(c.Listen.Host) == 0 {
		return "0.0.0.0"
	}
	return c.Listen.Host
}

func (c *ServerConfig) ListenAddress() string {
	return fmt.Sprintf("%s:%d", c.ListenHost(), c.ListenPort())
}

func (c *ServerConfig) AdminKeyEnv() string {
	if c.Auth.AdminKeyEnv == nil {
		return "LLAMA_GATEWAY_ADMIN_KEY"
	}
	return *c.Auth.AdminKeyEnv
}

func (c *ServerConfig) AdminKey() (string, error) {
	value, exists := os.LookupEnv(c.AdminKeyEnv())
	if !exists {
		return "", fmt.Errorf("admin key not found in environment variable %s", c.AdminKeyEnv())
	}
	return value, nil
}

func (a *api) IsAdminApiEnabled() bool {
	return a.AddModels || a.Reload
}
