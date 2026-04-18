package llamaserver

import "fmt"

type Presets struct {
	Global *Preset
	Models map[string]Preset
}

type Preset struct {
	Model   string
	Context *int
}

func (p Presets) String() string {
	s := "version = 1\n"
	if p.Global != nil {
		s += "\n[*]\n"
		s += p.Global.String()
	}
	for k, v := range p.Models {
		s += fmt.Sprintf("\n[%s]\n", k)
		s += v.String()
	}
	return s
}

func (p Preset) String() string {
	if p.Context == nil {
		return fmt.Sprintf("model = %s\n", p.Model)
	}
	return fmt.Sprintf("model = %s\nc = %d\n", p.Model, *p.Context)
}
