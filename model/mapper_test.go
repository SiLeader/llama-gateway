package model

import (
	"testing"
)

func TestDestinationPath(t *testing.T) {
	tests := []struct {
		dest string
		file string
		want string
	}{
		{"/models", "model.gguf", "/models/model.gguf"},
		{"/models/", "model.gguf", "/models/model.gguf"},
		{"", "model.gguf", "/model.gguf"},
	}
	for _, tt := range tests {
		info := Info{File: tt.file}
		got := info.DestinationPath(tt.dest)
		if got != tt.want {
			t.Errorf("DestinationPath(%q): got %q, want %q", tt.dest, got, tt.want)
		}
	}
}
