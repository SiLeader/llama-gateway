package llamaserver

import (
	"strings"
	"testing"
)

func intPtr(n int) *int { return &n }

func TestPresetString_WithContext(t *testing.T) {
	p := Preset{Model: "/models/gemma.gguf", Context: intPtr(4096)}
	got := p.String()
	want := "model = /models/gemma.gguf\nc = 4096\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPresetString_WithoutContext(t *testing.T) {
	p := Preset{Model: "/models/gemma.gguf", Context: nil}
	got := p.String()
	want := "model = /models/gemma.gguf\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPresetsString_Empty(t *testing.T) {
	ps := Presets{Global: nil, Models: map[string]Preset{}}
	got := ps.String()
	want := "version = 1\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPresetsString_NoGlobal(t *testing.T) {
	ps := Presets{
		Global: nil,
		Models: map[string]Preset{
			"gemma": {Model: "/models/gemma.gguf", Context: intPtr(4096)},
		},
	}
	got := ps.String()
	if !strings.HasPrefix(got, "version = 1\n") {
		t.Errorf("output does not start with 'version = 1\\n': %q", got)
	}
	if strings.Contains(got, "[*]") {
		t.Errorf("output should not contain [*] section when Global is nil")
	}
	if !strings.Contains(got, "[gemma]") {
		t.Errorf("output missing [gemma] section: %q", got)
	}
	if !strings.Contains(got, "model = /models/gemma.gguf") {
		t.Errorf("output missing model line: %q", got)
	}
}

func TestPresetsString_GlobalAndModels(t *testing.T) {
	ps := Presets{
		Global: &Preset{Model: "/models/default.gguf"},
		Models: map[string]Preset{
			"alpha": {Model: "/models/alpha.gguf", Context: intPtr(2048)},
			"beta":  {Model: "/models/beta.gguf"},
		},
	}
	got := ps.String()

	checks := []string{
		"version = 1\n",
		"[*]",
		"model = /models/default.gguf",
		"[alpha]",
		"model = /models/alpha.gguf",
		"c = 2048",
		"[beta]",
		"model = /models/beta.gguf",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("output missing %q:\n%s", c, got)
		}
	}
}

func TestPresetsString_GlobalOnly(t *testing.T) {
	ps := Presets{
		Global: &Preset{Model: "/models/default.gguf", Context: intPtr(8192)},
		Models: map[string]Preset{},
	}
	got := ps.String()
	if !strings.Contains(got, "[*]") {
		t.Errorf("output missing [*] section: %q", got)
	}
	if !strings.Contains(got, "c = 8192") {
		t.Errorf("output missing context line: %q", got)
	}
}
