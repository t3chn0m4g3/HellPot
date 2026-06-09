package config

import (
	"bytes"
	"errors"
	"flag"
	"testing"
)

func TestParseArgsHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	opts, err := ParseArgs([]string{"--help"}, &stdout, &stderr)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("ParseArgs() error = %v, want flag.ErrHelp", err)
	}
	if opts != (CLIOptions{}) {
		t.Fatalf("ParseArgs() options = %#v, want zero value", opts)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("Usage:")) {
		t.Fatalf("help output %q does not contain Usage", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestParseArgsAliases(t *testing.T) {
	var stdout, stderr bytes.Buffer

	opts, err := ParseArgs([]string{"-c", "config.toml", "-vv", "--nocolor"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if opts.ConfigPath != "config.toml" {
		t.Fatalf("ConfigPath = %q, want config.toml", opts.ConfigPath)
	}
	if !opts.Trace {
		t.Fatal("Trace = false, want true")
	}
	if !opts.NoColor {
		t.Fatal("NoColor = false, want true")
	}
}
