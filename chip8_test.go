package slay

import (
	"os"
	"testing"
)

func TestChip8Source(t *testing.T) {
	t.Chdir(t.TempDir())
	if got := chip8Source(); got != "" {
		t.Errorf("chip8Source() = %q in an empty directory, want an empty string", got)
	}
	for _, name := range []string{"pong.8o", "hello.8o"} {
		if err := os.WriteFile(name, []byte(": main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := chip8Source(); got != "hello.8o" {
		t.Errorf("chip8Source() = %q, want hello.8o", got)
	}
	if err := os.WriteFile("main.8o", []byte(": main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := chip8Source(); got != "main.8o" {
		t.Errorf("chip8Source() = %q, want main.8o", got)
	}
}
