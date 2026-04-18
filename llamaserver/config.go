package llamaserver

type Config struct {
	Executable string   `yaml:"executable"`
	Args       []string `yaml:"args,omitempty"`
}
