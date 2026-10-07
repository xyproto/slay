package slay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindWatcomRootPrefersEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("WATCOM", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "binl"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "binl", "wcl"), []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findWatcomRoot(); got != tmp {
		t.Fatalf("expected %q, got %q", tmp, got)
	}
}

func TestFindWatcomCompilerSetsEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("WATCOM", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "binl"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "binl", "wcl"), []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	wcl := findWatcomCompiler()
	if wcl != filepath.Join(tmp, "binl", "wcl") {
		t.Fatalf("unexpected compiler path %q", wcl)
	}
	if os.Getenv("INCLUDE") != filepath.Join(tmp, "h") {
		t.Fatalf("unexpected INCLUDE %q", os.Getenv("INCLUDE"))
	}
	if !strings.HasPrefix(os.Getenv("PATH"), filepath.Join(tmp, "binl")) {
		t.Fatalf("PATH does not start with the OW binl dir: %q", os.Getenv("PATH"))
	}
}

func TestAssembleWatcomFlags(t *testing.T) {
	bf := assembleWatcomFlags(Project{IsC: true}, BuildOptions{})
	if !bf.Watcom || !isWatcomCompiler(bf.Compiler) {
		t.Fatalf("expected a watcom build, got %+v", bf)
	}
	joined := strings.Join(bf.CFlags, " ")
	for _, want := range []string{"-q", "-bt=dos", "-ms", "-za99", "-ox", "-w3"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in %q", want, joined)
		}
	}
	dbg := assembleWatcomFlags(Project{IsC: true}, BuildOptions{Debug: true, Strict: true})
	joined = strings.Join(dbg.CFlags, " ")
	for _, want := range []string{"-od", "-d2", "-w4"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in %q", want, joined)
		}
	}
	cxx := assembleWatcomFlags(Project{IsC: false}, BuildOptions{Small: true})
	joined = strings.Join(cxx.CFlags, " ")
	if strings.Contains(joined, "-za99") {
		t.Errorf("did not expect -za99 for C++: %q", joined)
	}
	if !strings.Contains(joined, "-os") {
		t.Errorf("missing %q in %q", "-os", joined)
	}
}

func TestWatcomArgs(t *testing.T) {
	flags := BuildFlags{Watcom: true, CFlags: []string{"-q", "-bt=dos", "-ms"}, Defines: []string{"-DFOO=1", "-DBAR"}, IncPaths: []string{"inc"}}
	args := strings.Join(watcomArgs(flags, []string{"main.c", "util.c"}, "hello.exe"), " ")
	for _, want := range []string{"-q", "-bt=dos", "-dFOO=1", "-dBAR", "-i=inc", "main.c util.c", "-fe=hello.exe"} {
		if !strings.Contains(args, want) {
			t.Errorf("missing %q in %q", want, args)
		}
	}
	if strings.Contains(args, "-std=") {
		t.Errorf("did not expect gcc-style std flag: %q", args)
	}
}
