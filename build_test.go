package slay

import (
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestAssembleFlags_DefaultBuild(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `#include <iostream>
int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{})

	if flags.Compiler == "" {
		t.Fatal("no compiler found")
	}
	if flags.Std == "" {
		t.Fatal("no std flag set")
	}
	assertFlagPresent(t, flags.CFlags, "-O2")
	assertFlagPresent(t, flags.CFlags, "-pipe")
	assertFlagPresent(t, flags.CFlags, "-fPIC")
	assertFlagPresent(t, flags.CFlags, "-Wall")
	assertFlagPresent(t, flags.CFlags, "-Wshadow")
	assertFlagPresent(t, flags.CFlags, "-Wpedantic")
}

func TestAssembleFlags_DebugBuild(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Debug: true})

	assertFlagPresent(t, flags.CFlags, "-O0")
	assertFlagPresent(t, flags.CFlags, "-g")
	assertFlagPresent(t, flags.CFlags, "-fno-omit-frame-pointer")
	assertFlagPresent(t, flags.LDFlags, "-fsanitize=address")
}

func TestAssembleFlags_DebugNoSan(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Debug: true, NoSanitizers: true})

	assertFlagPresent(t, flags.CFlags, "-O0")
	assertFlagAbsent(t, flags.LDFlags, "-fsanitize=address")
}

func TestAssembleFlags_OptBuild(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Opt: true})

	if isEffectivelyClang(flags.Compiler) {
		// clang 17+ deprecated -Ofast; the code uses -O3 -ffast-math instead
		assertFlagPresent(t, flags.CFlags, "-O3")
		assertFlagPresent(t, flags.CFlags, "-ffast-math")
	} else {
		assertFlagPresent(t, flags.CFlags, "-Ofast")
	}
	assertFlagPresent(t, flags.CFlags, "-flto")
}

func TestAssembleFlags_SmallBuild(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Small: true})

	assertFlagPresent(t, flags.CFlags, "-Os")
	assertFlagPresent(t, flags.CFlags, "-ffunction-sections")
	assertFlagPresent(t, flags.CFlags, "-fdata-sections")
	assertFlagAbsent(t, flags.CFlags, "-fPIC")
}

func TestAssembleFlags_TinyBuild(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Small: true, Tiny: true})

	assertFlagPresent(t, flags.CFlags, "-fno-rtti")
	if !isDarwin() {
		// -s is added on non-Darwin platforms
		assertFlagPresent(t, flags.CFlags, "-s")
	}
}

func TestAssembleFlags_StrictBuild(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Strict: true})

	assertFlagPresent(t, flags.CFlags, "-Wextra")
	assertFlagPresent(t, flags.CFlags, "-Wconversion")
	assertFlagPresent(t, flags.CFlags, "-Weffc++")
}

func TestAssembleFlags_SloppyBuild(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Sloppy: true})

	assertFlagPresent(t, flags.CFlags, "-fpermissive")
	assertFlagPresent(t, flags.CFlags, "-w")
	assertFlagAbsent(t, flags.CFlags, "-Wall")
}

func TestAssembleFlags_CProject(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.c", `int main() { return 0; }`)

	proj := detectProject()
	if !proj.IsC {
		t.Fatal("expected IsC to be true")
	}
	flags := assembleFlags(proj, BuildOptions{})

	if runtime.GOOS == "linux" {
		validCStds := map[string]bool{"c11": true, "c17": true, "c18": true, "c23": true, "c2x": true}
		if !validCStds[flags.Std] {
			t.Errorf("expected a valid C standard for C on linux, got %q", flags.Std)
		}
	}
}

func TestAssembleFlags_OpenMP(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `#pragma omp parallel
int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{})

	assertFlagPresent(t, flags.CFlags, "-fopenmp")
	assertFlagPresent(t, flags.CFlags, "-O3")
	assertFlagPresent(t, flags.LDFlags, "-fopenmp")
}

func TestAssembleFlags_LinuxHardening(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{})

	assertFlagPresent(t, flags.CFlags, "-fno-plt")
	assertFlagPresent(t, flags.CFlags, "-fstack-protector-strong")
	assertFlagPresent(t, flags.CFlags, "-fstack-clash-protection")
	assertFlagPresent(t, flags.CFlags, "-fcf-protection")
}

func TestAssembleFlags_NoLinuxHardeningWhenSloppy(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Sloppy: true})

	assertFlagAbsent(t, flags.CFlags, "-fno-plt")
	assertFlagAbsent(t, flags.CFlags, "-fstack-protector-strong")
}

func TestAssembleFlags_ProfileGenerate(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Opt: true, ProfileGenerate: true})

	assertFlagPresent(t, flags.CFlags, "-fprofile-generate")
}

func TestAssembleFlags_ProfileUse(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Opt: true, ProfileUse: true})

	assertFlagPresent(t, flags.CFlags, "-fprofile-use")
}

func TestAssembleFlags_CXXFLAGS(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	os.Setenv("CXXFLAGS", "-DTEST_FLAG -march=native")
	defer os.Unsetenv("CXXFLAGS")

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{})

	assertFlagPresent(t, flags.CFlags, "-DTEST_FLAG")
	assertFlagPresent(t, flags.CFlags, "-march=native")
}

func TestAssembleFlags_CFLAGS(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.c", `int main() { return 0; }`)

	os.Setenv("CFLAGS", "-DTEST_CFLAG -march=native")
	defer os.Unsetenv("CFLAGS")

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{})

	assertFlagPresent(t, flags.CFlags, "-DTEST_CFLAG")
	assertFlagPresent(t, flags.CFlags, "-march=native")
}

func TestAssembleFlags_LDFLAGS(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	os.Setenv("LDFLAGS", "-Wl,-z,relro,-z,now")
	defer os.Unsetenv("LDFLAGS")

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{})

	assertFlagPresent(t, flags.LDFlags, "-Wl,-z,relro,-z,now")
}

func TestAssembleFlags_CPPFLAGS(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	t.Setenv("CPPFLAGS", "-DTEST_CPPFLAG")

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{})

	assertFlagPresent(t, flags.CFlags, "-DTEST_CPPFLAG")
}

func TestAssembleFlags_UserStd(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `int main() { return 0; }`)

	t.Setenv("CXXFLAGS", "-std=c++14 -DTEST_FLAG")

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{})

	if flags.Std != "c++14" {
		t.Errorf("expected std c++14, got %q", flags.Std)
	}
	assertFlagPresent(t, flags.CFlags, "-DTEST_FLAG")
	assertFlagAbsent(t, flags.CFlags, "-std=c++14")
}

// C projects fall back to CXXFLAGS, like cxx does, minus the C++-only flags
func TestUserCompileFlags_CFallback(t *testing.T) {
	t.Setenv("CXXFLAGS", "-std=c++20 -fno-rtti -DTEST_FLAG")

	flags, std := userCompileFlags(true)

	if std != "" {
		t.Errorf("expected no std, got %q", std)
	}
	assertFlagPresent(t, flags, "-DTEST_FLAG")
	assertFlagAbsent(t, flags, "-fno-rtti")
}

// Unstable flags would trigger needless rebuilds
func TestDirDefines_Stable(t *testing.T) {
	withTempDir(t)
	for _, dir := range []string{"img", "data", "shaders", "res", "scripts"} {
		os.Mkdir(dir, 0o755)
	}

	first := dirDefines()
	for range 5 {
		if got := dirDefines(); !slices.Equal(got, first) {
			t.Fatalf("dirDefines() is not stable: %v != %v", got, first)
		}
	}
}

func TestSanitizeFlags(t *testing.T) {
	// A broken .pc file can emit "-L" with no path, swallowing the next flag
	got := sanitizeFlags([]string{"-I/does/not/exist", "-L", "-lffts", "-lm"})
	want := []string{"-lffts", "-lm"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// A separated path is joined with its flag
	got = sanitizeFlags([]string{"-I", os.TempDir()})
	want = []string{"-I" + os.TempDir()}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildCompileArgs(t *testing.T) {
	flags := BuildFlags{
		Compiler: "g++",
		Std:      "c++17",
		CFlags:   []string{"-O2", "-Wall"},
		LDFlags:  []string{"-lm"},
		Defines:  []string{"-DFOO"},
		IncPaths: []string{"include"},
	}
	args := buildCompileArgs(flags, []string{"main.cpp"}, "myapp")
	joined := strings.Join(args, " ")
	for _, want := range []string{"-std=c++17", "-O2", "-Wall", "-DFOO", "-Iinclude", "-o", "myapp", "main.cpp", "-lm"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args missing %q: %s", want, joined)
		}
	}
}

func TestIsCompilerGCC(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/usr/bin/g++", true},
		{"/usr/bin/gcc", true},
		{"/usr/bin/x86_64-w64-mingw32-g++", true},
		{"/usr/bin/clang++", false},
		{"/usr/bin/cc", false},
	}
	for _, tc := range cases {
		if got := isCompilerGCC(tc.path); got != tc.want {
			t.Errorf("isCompilerGCC(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestIsCompilerClang(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/usr/bin/clang++", true},
		{"/usr/bin/clang", true},
		{"/usr/bin/g++", false},
		{"/usr/bin/cc", false},
	}
	for _, tc := range cases {
		if got := isCompilerClang(tc.path); got != tc.want {
			t.Errorf("isCompilerClang(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestDetectMinWinVersion(t *testing.T) {
	withTempDir(t)

	cases := []struct {
		name    string
		source  string
		wantVer int
	}{
		{"no win api", `int main() { return 0; }`, 0},
		{"vista api", `#include <windows.h>
int main() { DWORD m; GetConsoleMode(h, &m); m |= ENABLE_VIRTUAL_TERMINAL_PROCESSING; }`, 0x0600},
		{"win7 api", `#include <windows.h>
int main() { SetProcessDPIAware(); }`, 0x0601},
		{"win10 api", `#include <windows.h>
int main() { CreatePseudoConsole(size, in, out, 0, &hpc); }`, 0x0A00},
		{"mixed picks highest", `#include <windows.h>
int main() { GetTickCount64(); CreatePseudoConsole(size, in, out, 0, &hpc); }`, 0x0A00},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writeFile(t, "test_src.c", tc.source)
			got := detectMinWinVersion([]string{"test_src.c"})
			if got != tc.wantVer {
				t.Errorf("detectMinWinVersion() = 0x%04X, want 0x%04X", got, tc.wantVer)
			}
			os.Remove("test_src.c")
		})
	}
}

func TestAssembleFlags_TinyBuild_CProject_NoRtti(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.c", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Small: true, Tiny: true})

	assertFlagPresent(t, flags.CFlags, "-fno-ident")
	assertFlagPresent(t, flags.CFlags, "-fomit-frame-pointer")
	assertFlagAbsent(t, flags.CFlags, "-fno-rtti")
}

func TestDirDefines(t *testing.T) {
	withTempDir(t)
	os.MkdirAll("img", 0o755)
	os.MkdirAll("data", 0o755)

	defs := dirDefines()
	foundImg := false
	foundData := false
	for _, d := range defs {
		if strings.Contains(d, "IMGDIR") {
			foundImg = true
		}
		if strings.Contains(d, "DATADIR") {
			foundData = true
		}
	}
	if !foundImg {
		t.Error("expected IMGDIR define")
	}
	if !foundData {
		t.Error("expected DATADIR define")
	}
}

func TestMergeFlags(t *testing.T) {
	// Use real, existing directories so sanitizeFlags does not drop the
	// -I/-L flags for pointing at a nonexistent path. Hardcoding paths like
	// /usr/include fails on macOS and Windows, where they do not exist.
	incDir := t.TempDir()
	libDir := t.TempDir()

	cflags := []string{"-O2"}
	ldflags := []string{}
	flags := "-I" + incDir + " -lm -L" + libDir + " -Wl,--as-needed"
	cflags, ldflags = mergeFlags(cflags, ldflags, flags)
	assertFlagPresent(t, cflags, "-I"+incDir)
	assertFlagPresent(t, ldflags, "-lm")
	assertFlagPresent(t, ldflags, "-L"+libDir)
	assertFlagPresent(t, ldflags, "-Wl,--as-needed")
}

func TestDotSlash(t *testing.T) {
	sep := string(os.PathSeparator)
	cases := []struct {
		input, want string
	}{
		{"myapp", "." + sep + "myapp"},
		{"./myapp", "./myapp"},
	}
	if runtime.GOOS != "windows" {
		cases = append(cases, struct{ input, want string }{"/usr/bin/app", "/usr/bin/app"})
	}
	for _, tc := range cases {
		got := dotSlash(tc.input)
		if got != tc.want {
			t.Errorf("dotSlash(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// helpers
func assertFlagPresent(t *testing.T, flags []string, flag string) {
	t.Helper()
	if slices.Contains(flags, flag) {
		return
	}
	t.Errorf("expected flag %q in %v", flag, flags)
}

func assertFlagAbsent(t *testing.T, flags []string, flag string) {
	t.Helper()
	if slices.Contains(flags, flag) {
		t.Errorf("unexpected flag %q in %v", flag, flags)
		return
	}
}

func TestAssembleFlags_SloppyCProject_NoFpermissive(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.c", `int main() { return 0; }`)

	proj := detectProject()
	flags := assembleFlags(proj, BuildOptions{Sloppy: true})

	assertFlagAbsent(t, flags.CFlags, "-fpermissive")
	assertFlagPresent(t, flags.CFlags, "-w")
}

func TestCStdToCMakeStd(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"c23", "23"},
		{"c2x", "23"},
		{"c17", "17"},
		{"c18", "17"},
		{"c11", "11"},
		{"c99", "99"},
		{"c90", "90"},
		{"c89", "90"},
		{"unknown", "11"},
	}
	for _, tc := range cases {
		got := cStdToCMakeStd(tc.input)
		if got != tc.want {
			t.Errorf("cStdToCMakeStd(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCxxStdToCMakeStd(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"c++26", "26"},
		{"c++2c", "26"},
		{"c++23", "23"},
		{"c++2b", "23"},
		{"c++20", "20"},
		{"c++2a", "20"},
		{"c++17", "17"},
		{"c++14", "14"},
		{"c++11", "11"},
		{"c++98", "98"},
		{"c++03", "98"},
		{"unknown", "17"},
	}
	for _, tc := range cases {
		got := cxxStdToCMakeStd(tc.input)
		if got != tc.want {
			t.Errorf("cxxStdToCMakeStd(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDoCMake_ValidStandards(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.cpp", `#include <iostream>
int main() { std::cout << "hello"; return 0; }`)

	err := doGenerate(BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("CMakeLists.txt")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// CXX_STANDARD must be a valid CMake value
	validCxxStds := []string{"98", "11", "14", "17", "20", "23", "26"}
	foundValidCxx := false
	for _, std := range validCxxStds {
		if strings.Contains(content, "CXX_STANDARD "+std) {
			foundValidCxx = true
			break
		}
	}
	if !foundValidCxx {
		t.Error("CMakeLists.txt does not contain a valid CXX_STANDARD")
	}

	// C_STANDARD must be a valid CMake value
	validCStds := []string{"90", "99", "11", "17", "23"}
	foundValidC := false
	for _, std := range validCStds {
		if strings.Contains(content, "C_STANDARD "+std) {
			foundValidC = true
			break
		}
	}
	if !foundValidC {
		t.Error("CMakeLists.txt does not contain a valid C_STANDARD")
	}

	// Must NOT contain invalid standards
	if strings.Contains(content, "C_STANDARD 18") {
		t.Error("CMakeLists.txt contains invalid C_STANDARD 18")
	}
}

func TestDoCMake_CProject_ValidStandards(t *testing.T) {
	withTempDir(t)
	writeFile(t, "main.c", `int main() { return 0; }`)

	err := doGenerate(BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("CMakeLists.txt")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Should NOT have CXX_STANDARD for a C project
	if strings.Contains(content, "CXX_STANDARD") {
		t.Error("CMakeLists.txt should not set CXX_STANDARD for a C project")
	}

	// C_STANDARD must be valid
	validCStds := []string{"90", "99", "11", "17", "23"}
	foundValidC := false
	for _, std := range validCStds {
		if strings.Contains(content, "C_STANDARD "+std) {
			foundValidC = true
			break
		}
	}
	if !foundValidC {
		t.Error("CMakeLists.txt does not contain a valid C_STANDARD")
	}
}
