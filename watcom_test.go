package slay

import (
	"os"
	"os/exec"
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

func TestWatcomEndToEnd(t *testing.T) {
	if findWatcomRoot() == "" {
		t.Skip("Open Watcom not installed")
	}
	if out, err := exec.Command("dosbox-x", "--version").CombinedOutput(); err != nil {
		t.Skipf("dosbox-x not usable: %v\n%s", err, out)
	}
	dir := t.TempDir()
	src := "#include <stdio.h>\n\nint main(void)\n{\n\tprintf(\"watcom-e2e-ok\\n\");\n\treturn 0;\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Build(dir, BuildOptions{Watcom: true})
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, result.OutputExecutable)
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 2 || string(data[:2]) != "MZ" {
		t.Fatalf("expected an MZ executable")
	}
	cmd := exec.Command("dosbox-x", "-defaultconf", "-c", "mount c .", "-c", "c:", "-c", result.OutputExecutable+" > out.txt", "-c", "exit")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "SDL_VIDEODRIVER=dummy")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dosbox-x failed: %v\n%s", err, out)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "watcom-e2e-ok") {
		t.Fatalf("unexpected program output %q", got)
	}
}

func TestScanSourceForFlagsWatcom(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "hello.c")
	if err := os.WriteFile(src, []byte("#include <conio.h>\n#include <i86.h>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var p Project
	scanSourceForFlags(src, &p)
	if !p.HasWatcom {
		t.Fatal("expected a watcom project")
	}
}

func TestAssembleWatcomFlagsIncludePath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("WATCOM", tmp)
	for _, d := range []string{"binl", "h"} {
		if err := os.MkdirAll(filepath.Join(tmp, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmp, "binl", "wcl"), []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	bf := assembleWatcomFlags(Project{IsC: true}, BuildOptions{})
	joined := strings.Join(bf.IncPaths, " ")
	if !strings.Contains(joined, filepath.Join(tmp, "h")) {
		t.Fatalf("missing the watcom include path in %q", joined)
	}
}

func TestAutoDetectWatcomBuild(t *testing.T) {
	if findWatcomRoot() == "" {
		t.Skip("no open watcom installation found")
	}
	dir := t.TempDir()
	src := "#include <conio.h>\n#include <i86.h>\n\nint main(void)\n{\n\tgetch();\n\treturn 0;\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Build(dir, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, result.OutputExecutable))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 2 || string(data[:2]) != "MZ" {
		t.Fatalf("expected an MZ executable")
	}
}
