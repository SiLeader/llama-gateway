package model

import (
	"testing"
)

func TestValidate_Valid(t *testing.T) {
	cases := []Info{
		{Name: "gemma3", Id: "org/repo", File: "model.gguf"},
		{Name: "my-model-1", Id: "org/repo-name", File: "sub/model.gguf"},
		{Name: "Model_A", Id: "Org/Repo.Name", File: "a.gguf"},
		{Name: "a", Id: "org/r", File: "f.gguf"},
	}
	for _, info := range cases {
		if err := info.Validate(); err != nil {
			t.Errorf("Validate(%+v) unexpected error: %v", info, err)
		}
	}
}

func TestValidate_EmptyFields(t *testing.T) {
	cases := []Info{
		{Name: "", Id: "org/repo", File: "model.gguf"},
		{Name: "model", Id: "", File: "model.gguf"},
		{Name: "model", Id: "org/repo", File: ""},
	}
	for _, info := range cases {
		if err := info.Validate(); err == nil {
			t.Errorf("Validate(%+v) expected error for empty field", info)
		}
	}
}

func TestValidate_InvalidName(t *testing.T) {
	cases := []string{"my model", "model!", "model/name", "model.name"}
	for _, name := range cases {
		info := Info{Name: name, Id: "org/repo", File: "model.gguf"}
		if err := info.Validate(); err == nil {
			t.Errorf("Validate with name=%q expected error", name)
		}
	}
}

func TestValidate_InvalidId(t *testing.T) {
	cases := []string{"/leading-slash", "trailing-slash/", "org//double", "org/repo/extra"}
	for _, id := range cases {
		info := Info{Name: "model", Id: id, File: "model.gguf"}
		if err := info.Validate(); err == nil {
			t.Errorf("Validate with id=%q expected error", id)
		}
	}
}

func TestValidate_PathTraversal(t *testing.T) {
	cases := []string{"../evil.gguf", "../../etc/passwd", "sub/../../../evil"}
	for _, file := range cases {
		info := Info{Name: "model", Id: "org/repo", File: file}
		if err := info.Validate(); err == nil {
			t.Errorf("Validate with file=%q expected error for path traversal", file)
		}
	}
}

func TestIsValidPath(t *testing.T) {
	valid := []string{"model.gguf", "sub/model.gguf", "a/b/c.gguf"}
	for _, p := range valid {
		if !isValidPath(p) {
			t.Errorf("isValidPath(%q) = false, want true", p)
		}
	}

	invalid := []string{"", "../evil", "../../etc", "/absolute/path", "sub/../../escape"}
	for _, p := range invalid {
		if isValidPath(p) {
			t.Errorf("isValidPath(%q) = true, want false", p)
		}
	}
}
