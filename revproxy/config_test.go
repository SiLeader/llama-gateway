package revproxy

import (
	"testing"
)

func TestListenPort_Default(t *testing.T) {
	c := &ServerConfig{}
	if got := c.ListenPort(); got != 8080 {
		t.Errorf("ListenPort() = %d, want 8080", got)
	}
}

func TestListenPort_Custom(t *testing.T) {
	c := &ServerConfig{Listen: listen{Port: 9090}}
	if got := c.ListenPort(); got != 9090 {
		t.Errorf("ListenPort() = %d, want 9090", got)
	}
}

func TestListenHost_Default(t *testing.T) {
	c := &ServerConfig{}
	if got := c.ListenHost(); got != "0.0.0.0" {
		t.Errorf("ListenHost() = %q, want 0.0.0.0", got)
	}
}

func TestListenHost_Custom(t *testing.T) {
	c := &ServerConfig{Listen: listen{Host: "127.0.0.1"}}
	if got := c.ListenHost(); got != "127.0.0.1" {
		t.Errorf("ListenHost() = %q, want 127.0.0.1", got)
	}
}

func TestListenAddress(t *testing.T) {
	c := &ServerConfig{Listen: listen{Host: "127.0.0.1", Port: 9090}}
	if got := c.ListenAddress(); got != "127.0.0.1:9090" {
		t.Errorf("ListenAddress() = %q, want 127.0.0.1:9090", got)
	}
}

func TestListenAddress_Defaults(t *testing.T) {
	c := &ServerConfig{}
	if got := c.ListenAddress(); got != "0.0.0.0:8080" {
		t.Errorf("ListenAddress() = %q, want 0.0.0.0:8080", got)
	}
}

func TestAdminKeyEnv_Default(t *testing.T) {
	c := &ServerConfig{}
	if got := c.AdminKeyEnv(); got != "LLAMA_GATEWAY_ADMIN_KEY" {
		t.Errorf("AdminKeyEnv() = %q, want LLAMA_GATEWAY_ADMIN_KEY", got)
	}
}

func TestAdminKeyEnv_Custom(t *testing.T) {
	envName := "MY_CUSTOM_KEY"
	c := &ServerConfig{Auth: auth{AdminKeyEnv: &envName}}
	if got := c.AdminKeyEnv(); got != envName {
		t.Errorf("AdminKeyEnv() = %q, want %q", got, envName)
	}
}

func TestAdminKey_Found(t *testing.T) {
	t.Setenv("LLAMA_GATEWAY_ADMIN_KEY", "secret123")
	c := &ServerConfig{}
	key, err := c.AdminKey()
	if err != nil {
		t.Fatalf("AdminKey() unexpected error: %v", err)
	}
	if key != "secret123" {
		t.Errorf("AdminKey() = %q, want secret123", key)
	}
}

func TestAdminKey_NotFound(t *testing.T) {
	envName := "LLAMA_GATEWAY_ADMIN_KEY_MISSING_ENV_THAT_DOES_NOT_EXIST_XYZ"
	c := &ServerConfig{Auth: auth{AdminKeyEnv: &envName}}
	_, err := c.AdminKey()
	if err == nil {
		t.Error("AdminKey() expected error when env var not set")
	}
}

func TestIsAdminApiEnabled(t *testing.T) {
	enabled := &ServerConfig{Apis: api{AddModels: true}}
	if !enabled.Apis.IsAdminApiEnabled() {
		t.Error("IsAdminApiEnabled() = false, want true when addModels is true")
	}

	disabled := &ServerConfig{Apis: api{AddModels: false}}
	if disabled.Apis.IsAdminApiEnabled() {
		t.Error("IsAdminApiEnabled() = true, want false when addModels is false")
	}
}
