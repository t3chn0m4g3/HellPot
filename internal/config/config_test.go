package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateConfigWritesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	written, err := GenerateConfig(path, CLIOptions{})
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}
	if written != path {
		t.Fatalf("GenerateConfig() path = %q, want %q", written, path)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !containsAll(
		string(body),
		"# Used as the HTTP \"Server\" response header.",
		"[http]",
		"bind_port = \"8080\"",
		"docker_logging = false",
		"console_time_format = \"3:04PM\"",
		"[logger]",
	) {
		t.Fatalf("generated config missing expected defaults:\n%s", string(body))
	}

	if err := Init(CLIOptions{ConfigPath: path}); err != nil {
		t.Fatalf("generated config is not loadable: %v", err)
	}
	if HTTPBind != "127.0.0.1" {
		t.Fatalf("HTTPBind = %q, want 127.0.0.1", HTTPBind)
	}
	if DockerLogging {
		t.Fatal("DockerLogging = true, want false in generated host config")
	}
}

func TestDockerConfigLoads(t *testing.T) {
	path := filepath.Join("..", "..", "docker_config.toml")

	if err := Init(CLIOptions{ConfigPath: path}); err != nil {
		t.Fatalf("docker_config.toml is not loadable: %v", err)
	}
	if HTTPBind != "0.0.0.0" {
		t.Fatalf("HTTPBind = %q, want 0.0.0.0", HTTPBind)
	}
	if !CatchAll {
		t.Fatal("CatchAll = false, want true")
	}
	if !DockerLogging {
		t.Fatal("DockerLogging = false, want true")
	}
}

func TestInitCustomConfigWithEnvAndCLIOverrides(t *testing.T) {
	t.Setenv("HELLPOT_HTTP_BIND__ADDR", "10.0.0.9")
	t.Setenv("HELLPOT_LOGGER_DEBUG", "false")

	cfg := filepath.Join(t.TempDir(), "custom.toml")
	if err := os.WriteFile(cfg, []byte(`
[http]
bind_addr = "127.0.0.1"
bind_port = "9090"

[logger]
debug = true
trace = false
`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := Init(CLIOptions{
		ConfigPath: cfg,
		NoColor:    true,
		Trace:      true,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if Filename != cfg {
		t.Fatalf("Filename = %q, want %q", Filename, cfg)
	}
	if HTTPBind != "10.0.0.9" {
		t.Fatalf("HTTPBind = %q, want env override", HTTPBind)
	}
	if HTTPPort != "9090" {
		t.Fatalf("HTTPPort = %q, want config value", HTTPPort)
	}
	if !NoColor {
		t.Fatal("NoColor = false, want CLI override")
	}
	if !Trace {
		t.Fatal("Trace = false, want CLI override")
	}
	if Debug {
		t.Fatal("Debug = true, want env override to false")
	}
}

func containsAll(s string, needles ...string) bool {
	for _, needle := range needles {
		if !contains(s, needle) {
			return false
		}
	}
	return true
}

func contains(s, needle string) bool {
	return strings.Contains(s, needle)
}
