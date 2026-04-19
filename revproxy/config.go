package revproxy

import "fmt"

type ServerConfig struct {
	Listen listen `yaml:"listen"`
	Apis   api    `yaml:"apis"`
}

type listen struct {
	Host string `yaml:"host,omitempty"`
	Port int    `yaml:"port,omitempty"`
}

type api struct {
	AddModels bool `yaml:"addModels"`
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
