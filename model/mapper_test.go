package model

import (
	"testing"
)

func TestNewModelMapper(t *testing.T) {
	models := []Info{
		{Name: "model-a", Id: "org/model-a", File: "model-a.gguf"},
		{Name: "model-b", Id: "org/model-b", File: "model-b.gguf"},
	}
	mapper := NewModelMapper(models, "/models")
	if mapper == nil {
		t.Fatal("expected non-nil Mapper")
	}

	for _, m := range models {
		got := mapper.UseModel(m.Name)
		if got == nil {
			t.Fatalf("UseModel(%q) returned nil", m.Name)
		}
	}
}

func TestNewModelMapper_Empty(t *testing.T) {
	mapper := NewModelMapper([]Info{}, "/models")
	if mapper == nil {
		t.Fatal("expected non-nil Mapper")
	}
	got := mapper.UseModel("anything")
	if got != nil {
		t.Errorf("expected nil for unknown model, got %q", *got)
	}
}

func TestUseModel_Found(t *testing.T) {
	dest := "/var/models"
	mapper := NewModelMapper([]Info{
		{Name: "gemma", Id: "org/gemma", File: "gemma.gguf"},
	}, dest)

	got := mapper.UseModel("gemma")
	if got == nil {
		t.Fatal("expected non-nil path")
	}
	want := dest + "/" + "gemma.gguf"
	if *got != want {
		t.Errorf("got %q, want %q", *got, want)
	}
}

func TestUseModel_NotFound(t *testing.T) {
	mapper := NewModelMapper([]Info{
		{Name: "gemma", Id: "org/gemma", File: "gemma.gguf"},
	}, "/models")

	got := mapper.UseModel("nonexistent")
	if got != nil {
		t.Errorf("expected nil, got %q", *got)
	}
}

func TestUseModel_MultipleModels(t *testing.T) {
	dest := "/data"
	models := []Info{
		{Name: "alpha", Id: "org/alpha", File: "alpha.gguf"},
		{Name: "beta", Id: "org/beta", File: "beta.gguf"},
		{Name: "gamma", Id: "org/gamma", File: "gamma.gguf"},
	}
	mapper := NewModelMapper(models, dest)

	for _, m := range models {
		got := mapper.UseModel(m.Name)
		if got == nil {
			t.Fatalf("UseModel(%q) returned nil", m.Name)
		}
		want := dest + "/" + m.File
		if *got != want {
			t.Errorf("model %q: got %q, want %q", m.Name, *got, want)
		}
	}
}

func TestDestinationPath(t *testing.T) {
	tests := []struct {
		dest string
		file string
		want string
	}{
		{"/models", "model.gguf", "/models/model.gguf"},
		{"/models/", "model.gguf", "/models/model.gguf"},
		{"", "model.gguf", "model.gguf"},
	}
	for _, tt := range tests {
		info := Info{File: tt.file}
		got := info.DestinationPath(tt.dest)
		if got != tt.want {
			t.Errorf("DestinationPath(%q): got %q, want %q", tt.dest, got, tt.want)
		}
	}
}
