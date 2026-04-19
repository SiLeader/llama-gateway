package llamaserver

type Config struct {
	Executable string   `yaml:"executable"`
	Args       []string `yaml:"args,omitempty"`
	Threads    *int     `yaml:"threads,omitempty"`
}
