package slay

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// withTempDir creates a temp directory, chdirs into it, and restores after the test.
func withTempDir(t *testing.T) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if dir != "." {
		os.MkdirAll(dir, 0o755)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGetMainSourceFile_MainCpp(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)
	got := GetMainSourceFile(nil)
	if got != "main.cpp" {
		t.Errorf("expected main.cpp, got %q", got)
	}
}

func TestGetMainSourceFile_MainC(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.c", `int main() { return 0; }`)
	got := GetMainSourceFile(nil)
	if got != "main.c" {
		t.Errorf("expected main.c, got %q", got)
	}
}

func TestGetMainSourceFile_NamedFile(t *testing.T) {
	withTempDir(t)
	writeFile(t, "app.cpp", `#include <iostream>
int main() { return 0; }`)
	got := GetMainSourceFile(nil)
	if got != "app.cpp" {
		t.Errorf("expected app.cpp, got %q", got)
	}
}

func TestGetMainSourceFile_NoMain(t *testing.T) {
	withTempDir(t)
	writeFile(t, "lib.cpp", `void foo() {}`)
	got := GetMainSourceFile(nil)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestGetMainSourceFile_ExcludesTests(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)
	writeFile(t, "foo_test.cpp", `int main() { return 0; }`)
	got := GetMainSourceFile([]string{"foo_test.cpp"})
	if got != "main.cpp" {
		t.Errorf("expected main.cpp, got %q", got)
	}
}

func TestGetTestSources(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)
	writeFile(t, "foo_test.cpp", `int main() { return 0; }`)
	writeFile(t, "bar_test.cc", `int main() { return 0; }`)
	tests := getTestSources()
	if len(tests) != 2 {
		t.Errorf("expected 2 test files, got %d: %v", len(tests), tests)
	}
}

func TestGetTestSources_TestDotCpp(t *testing.T) {
	withTempDir(t)
	writeFile(t, "test.cpp", `int main() { return 0; }`)
	tests := getTestSources()
	if len(tests) != 1 {
		t.Fatalf("expected 1 test file, got %d: %v", len(tests), tests)
	}
	if tests[0] != "test.cpp" {
		t.Errorf("expected test.cpp, got %q", tests[0])
	}
}

func TestIsTestFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"foo_test.cpp", true},
		{"bar_test.cc", true},
		{"test.cpp", true},
		{"test.c", true},
		{"main.cpp", false},
		{"helper.cpp", false},
		{"testing.cpp", false},
	}
	for _, tc := range cases {
		got := isTestFile(tc.path)
		if got != tc.want {
			t.Errorf("isTestFile(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestGetDepSources(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)
	writeFile(t, "util.cpp", `void util() {}`)
	writeFile(t, "foo_test.cpp", `int main() { return 0; }`)
	deps := getDepSources("main.cpp", []string{"foo_test.cpp"})
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep, got %d: %v", len(deps), deps)
	}
	if deps[0] != "util.cpp" {
		t.Errorf("expected util.cpp, got %q", deps[0])
	}
}

func TestGetDepSources_WithCommon(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)
	writeFile(t, "common/helper.cpp", `void helper() {}`)
	deps := getDepSources("main.cpp", nil)
	found := slices.Contains(deps, filepath.Join("common", "helper.cpp"))
	if !found {
		t.Errorf("expected common/helper.cpp in deps, got %v", deps)
	}
}

func TestContainsMain(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		content string
		want    bool
	}{
		{"int main() { return 0; }", true},
		{"int main(int argc, char** argv) {}", true},
		{"void foo() {}", false},
		{"// main(", false},
		{"int\nmain() { return 0; }", true},
	}
	for i, tc := range cases {
		path := filepath.Join(dir, "test"+string(rune('0'+i))+".cpp")
		os.WriteFile(path, []byte(tc.content), 0o644)
		got := containsMain(path)
		if got != tc.want {
			t.Errorf("containsMain(%q) = %v, want %v", tc.content, got, tc.want)
		}
	}
}

// Flags and external includes may only be visible in the project headers
func TestDetectProject_ScansProjectHeaders(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", "#include \"util.h\"\nint main() { return 0; }")
	writeFile(t, "include/util.h", "#include <experimental/filesystem>\n#include <thread>\n#include <SDL2/SDL.h>\n")

	p := detectProject()

	assertTrue(t, p.HasExperimentalFS, "HasExperimentalFS")
	assertTrue(t, p.HasThreads, "HasThreads")
	assertContains(t, p.Includes, "SDL2/SDL.h")
	// experimental/ headers are part of the standard library, not packages
	if slices.Contains(p.Includes, "experimental/filesystem") {
		t.Errorf("experimental/filesystem should not be an external include: %v", p.Includes)
	}
}

func TestScanSourceForFlags(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
		check   func(t *testing.T, p Project)
	}{
		{
			name:    "OpenMP",
			content: "#pragma omp parallel\nint main() {}",
			check:   func(t *testing.T, p Project) { assertTrue(t, p.HasOpenMP, "HasOpenMP") },
		},
		{
			name:    "Boost",
			content: "#include <boost/filesystem.hpp>\nint main() {}",
			check: func(t *testing.T, p Project) {
				assertTrue(t, p.HasBoost, "HasBoost")
				assertContains(t, p.BoostLibs, "boost_filesystem")
			},
		},
		{
			name:    "Qt6",
			content: "#include <QApplication>\nint main() {}",
			check:   func(t *testing.T, p Project) { assertTrue(t, p.HasQt6, "HasQt6") },
		},
		{
			name:    "Filesystem",
			content: "#include <filesystem>\nint main() {}",
			check:   func(t *testing.T, p Project) { assertTrue(t, p.HasFS, "HasFS") },
		},
		{
			name:    "ExperimentalFilesystem",
			content: "#include <experimental/filesystem>\nint main() {}",
			check: func(t *testing.T, p Project) {
				assertTrue(t, p.HasExperimentalFS, "HasExperimentalFS")
			},
		},
		{
			name:    "Math",
			content: "#include <cmath>\nint main() {}",
			check:   func(t *testing.T, p Project) { assertTrue(t, p.HasMathLib, "HasMathLib") },
		},
		{
			name:    "Threads",
			content: "#include <thread>\nint main() {}",
			check:   func(t *testing.T, p Project) { assertTrue(t, p.HasThreads, "HasThreads") },
		},
		{
			name:    "Dlopen",
			content: "#include <dlfcn.h>\nint main() {}",
			check:   func(t *testing.T, p Project) { assertTrue(t, p.HasDlopen, "HasDlopen") },
		},
		{
			name:    "Win64",
			content: "#include <windows.h>\nint main() {}",
			check:   func(t *testing.T, p Project) { assertTrue(t, p.HasWin64, "HasWin64") },
		},
		{
			name:    "GLFW_Vulkan",
			content: "#define GLFW_INCLUDE_VULKAN\n#include <GLFW/glfw3.h>\nint main() {}",
			check:   func(t *testing.T, p Project) { assertTrue(t, p.HasGLFWVulkan, "HasGLFWVulkan") },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".cpp")
			os.WriteFile(path, []byte(tc.content), 0o644)
			var p Project
			scanSourceForFlags(path, &p)
			tc.check(t, p)
		})
	}
}

func TestDetectProject_HelloExample(t *testing.T) {
	orig, _ := os.Getwd()
	defer os.Chdir(orig)

	helloDir := filepath.Join(orig, "examples", "hello")
	if _, err := os.Stat(helloDir); os.IsNotExist(err) {
		t.Skip("examples/hello not found")
	}
	os.Chdir(helloDir)

	p := detectProject()
	if p.MainSource != "main.cpp" {
		t.Errorf("expected main.cpp, got %q", p.MainSource)
	}
	if p.IsC {
		t.Error("expected IsC=false for C++ project")
	}
	if len(p.TestSources) == 0 {
		t.Error("expected test sources in hello example")
	}
	if len(p.DepSources) == 0 {
		t.Error("expected dep sources from common/ in hello example")
	}
}

func TestExecutableName(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)

	// Name from directory
	subDir := filepath.Join(dir, "myproject")
	os.MkdirAll(subDir, 0o755)
	os.Chdir(subDir)
	if got := executableName(); got != "myproject" {
		t.Errorf("expected myproject, got %q", got)
	}

	// "src" directory should return "main"
	srcDir := filepath.Join(dir, "src")
	os.MkdirAll(srcDir, 0o755)
	os.Chdir(srcDir)
	if got := executableName(); got != "main" {
		t.Errorf("expected main for src/ dir, got %q", got)
	}
}

func TestDirectScanIncludes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.cpp")
	content := `#include <iostream>
#include <SDL2/SDL.h>
#include "myheader.h"
#include <boost/filesystem.hpp>
`
	os.WriteFile(path, []byte(content), 0o644)
	includes := directScanIncludes(path)
	expected := []string{"iostream", "SDL2/SDL.h", "boost/filesystem.hpp"}
	if len(includes) != len(expected) {
		t.Fatalf("expected %d includes, got %d: %v", len(expected), len(includes), includes)
	}
	for i, want := range expected {
		if includes[i] != want {
			t.Errorf("include[%d] = %q, want %q", i, includes[i], want)
		}
	}
}

func TestUniqueStrings(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b"}
	got := uniqueStrings(input)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("uniqueStrings(%v) = %v", input, got)
	}
}

func TestAppendUnique(t *testing.T) {
	s := []string{"a", "b"}
	s = appendUnique(s, "b")
	if len(s) != 2 {
		t.Errorf("appendUnique should not duplicate, got %v", s)
	}
	s = appendUnique(s, "c")
	if len(s) != 3 {
		t.Errorf("appendUnique should append new, got %v", s)
	}
}

// helpers
func assertTrue(t *testing.T, val bool, name string) {
	t.Helper()
	if !val {
		t.Errorf("expected %s to be true", name)
	}
}

func assertContains(t *testing.T, slice []string, val string) {
	t.Helper()
	if slices.Contains(slice, val) {
		return
	}
	t.Errorf("expected %v to contain %q", slice, val)
}

func TestDetectGtkVersion(t *testing.T) {
	withTempDir(t)
	cases := []struct {
		src  string
		want int
	}{
		{"int main(int argc, char **argv) { gtk_init(&argc, &argv); GtkWidget *w = gtk_window_new(GTK_WINDOW_TOPLEVEL); gtk_container_add(GTK_CONTAINER(w), gtk_box_new(GTK_ORIENTATION_VERTICAL, 0)); gtk_widget_show_all(w); gtk_main(); }", 3},
		{"static void activate(GtkApplication *app, gpointer d) { GtkWidget *w = gtk_application_window_new(app); gtk_window_set_child(GTK_WINDOW(w), gtk_box_new(GTK_ORIENTATION_VERTICAL, 0)); gtk_window_present(GTK_WINDOW(w)); }", 4},
		{"int main(int argc, char **argv) { gtk_init(&argc, &argv); GtkWidget *w = gtk_window_new(GTK_WINDOW_TOPLEVEL); gtk_signal_connect(GTK_OBJECT(w), \"destroy\", NULL, NULL); gtk_main(); }", 2},
		{"int main(void) { gtk_init(); GtkWidget *w = gtk_window_new(); gtk_window_present(GTK_WINDOW(w)); }", 4},
		{"int main(void) { return 0; }", 0},
	}
	for i, tc := range cases {
		writeFile(t, "main.c", "#include <gtk/gtk.h>\n"+tc.src+"\n")
		if got := detectGtkVersion([]string{"main.c"}); got != tc.want {
			t.Errorf("case %d: detectGtkVersion = %d, want %d", i, got, tc.want)
		}
	}
}
