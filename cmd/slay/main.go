package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/xyproto/env/v2"
	. "github.com/xyproto/slay"
)

const versionString = "slay 1.3.6"

func printHelp() {
	fmt.Printf(`%s

Usage: slay [modifiers...] [action] [args...]

Modifiers (combinable):
  clang       - use clang/clang++ compiler
  zap         - use zapcc++ compiler
  debug       - enable debug flags and sanitizers
  nosan       - disable sanitizers (use with debug)
  opt         - enable optimizations (-Ofast/-O3, -flto)
  strict      - enable strict warning flags
  sloppy      - enable permissive flags
  small       - optimize for size (-Os)
  tiny        - minimize size (-Os + sstrip/upx)
  win64       - cross-compile for 64-bit Windows

Actions:
  build       - compile the project (default)
  run         - build and run
  debug       - debug build and launch debugger
  rebuild     - clean and build
  clean       - clean all build artifacts
  fastclean   - only remove executable and *.o
  test        - build and run tests
  testbuild   - build tests (without running)
  pgo         - profile-guided optimization (build, run, rebuild)
  fmt         - format source code with clang-format
  generate    - generate CMakeLists.txt
  makefile    - generate a standalone Makefile
  cmake       - build with cmake (prefers ninja, falls back to make)
  make        - build with make (falls back to cmake+make)
  ninja       - build with ninja (falls back to cmake+ninja)
  install     - install the project (PREFIX, DESTDIR)
  pkg         - package the project into pkg/
  export      - export a standalone Makefile and build.sh
  script      - generate build.sh and clean.sh
  valgrind    - build and profile with valgrind
  pro         - generate QtCreator project file
  version     - show version

Compound actions:
  ninjainstall  - install from ninja build
  ninjaclean    - clean ninja build
  makeinstall   - install from make/cmake+make build
  makeclean     - clean make/cmake+make build

Environment variables:
  CC, CXX     - the C and C++ compiler to use
  CFLAGS      - extra flags when compiling C
  CXXFLAGS    - extra flags when compiling C++
  CPPFLAGS    - extra flags for both C and C++
  LDFLAGS     - extra flags when linking

Modifiers and variables can also be given the cxx way, as "name=value",
like "strict=1", "std=c++17", "CXX=clang++" or "CXXFLAGS=-O0 -g".

Examples:
  slay                  - standard build
  slay clang            - build with clang
  slay clang strict     - build with clang and strict warnings
  slay debug            - debug build and launch debugger
  slay debug build      - debug build (without launching debugger)
  slay clang debug      - clang debug build and launch debugger
  slay opt run          - optimized build and run
  slay small win64      - size-optimized cross-compile for Windows
  slay std=c++17        - build with a specific C++ standard
  slay -C <dir> ...    - run in the given directory
`, versionString)
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// isModifier returns true if the word is a recognized modifier.
func isModifier(word string) bool {
	switch word {
	case "clang", "zap", "opt", "strict", "sloppy", "small", "tiny",
		"win64", "win", "debug", "nosan", "nosanitizers":
		return true
	}
	return false
}

// isAction returns true if the word is a recognized action.
func isAction(word string) bool {
	switch word {
	case "build", "run", "rebuild", "clean", "fastclean",
		"test", "testbuild", "pgo", "rec", "fmt", "generate",
		"cmakelists", "cmakelist", "cmakelists.txt", "CMakeLists.txt",
		"cmake", "make", "ninja", "install", "pkg", "export", "script",
		"valgrind", "pro", "makefile", "Makefile",
		"ninjainstall", "ninja_install", "ninjaclean", "ninja_clean",
		"makeinstall", "make_install", "makeclean", "make_clean",
		"debug", "version":
		return true
	}
	return false
}

// expandLegacy expands legacy compound commands into modifier+action pairs.
func expandLegacy(word string) []string {
	switch word {
	case "debugbuild":
		return []string{"debug", "build"}
	case "debugnosan":
		return []string{"debug", "nosan", "build"}
	case "clangdebug":
		return []string{"clang", "debug"}
	case "clangstrict":
		return []string{"clang", "strict", "build"}
	case "clangsloppy":
		return []string{"clang", "sloppy", "build"}
	case "clangrebuild":
		return []string{"clang", "rebuild"}
	case "clangtest":
		return []string{"clang", "test"}
	case "smallwin64", "smallwin":
		return []string{"small", "win64", "build"}
	case "tinywin64", "tinywin":
		return []string{"tiny", "win64", "build"}
	}
	return nil
}

// isVariableName returns true if the word can be an environment variable name.
func isVariableName(word string) bool {
	if word == "" {
		return false
	}
	for i, r := range word {
		letter := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if !letter && !(i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// appendEnvFlags appends a flag to an environment variable, like CXXFLAGS.
func appendEnvFlags(name, flag string) {
	if existing := os.Getenv(name); existing != "" {
		flag = existing + " " + flag
	}
	os.Setenv(name, flag)
}

// applyAssignment handles a cxx-style "NAME=value" argument, either by setting
// an environment variable or by returning the equivalent slay modifier tokens.
func applyAssignment(name, value string) []string {
	enabled := value != "" && value != "0"
	switch name {
	case "clang", "zap", "opt", "strict", "sloppy", "small", "tiny", "win64", "win", "nosan":
		if enabled {
			return []string{name}
		}
	case "debug":
		// "debug=1" is a debug build, while the "debug" action starts a debugger
		if enabled {
			return []string{"debug", "build"}
		}
	case "std":
		if strings.HasPrefix(value, "c++") || strings.HasPrefix(value, "gnu++") {
			appendEnvFlags("CXXFLAGS", "-std="+value)
		} else {
			appendEnvFlags("CFLAGS", "-std="+value)
		}
	default:
		// CFLAGS, CXXFLAGS, CC, CXX, PREFIX, DESTDIR, pkgdir and friends
		os.Setenv(name, value)
	}
	return nil
}

// extractAssignments converts cxx-style "NAME=value" arguments into modifier
// tokens and environment variables. Arguments that follow an action which takes
// trailing arguments are left alone, since they belong to the executable.
func extractAssignments(args []string) []string {
	var modifiers, rest []string
	for i, arg := range args {
		name, value, isAssignment := strings.Cut(arg, "=")
		if !isAssignment || !isVariableName(name) {
			rest = append(rest, arg)
			if arg == "run" || arg == "pgo" || arg == "rec" {
				rest = append(rest, args[i+1:]...)
				break
			}
			continue
		}
		modifiers = append(modifiers, applyAssignment(name, value)...)
	}
	return append(modifiers, rest...)
}

func main() {
	args := os.Args[1:]

	// Handle -C <dir>
	// Use slices.Index which returns -1 if not found
	if argpos := slices.Index(args, "-C"); argpos >= 0 && (argpos+1) < len(args) {
		if err := os.Chdir(args[argpos+1]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		// Delete removes elements from [argpos : argpos+2]
		args = slices.Delete(args, argpos, argpos+2)
	}

	// Handle cxx-style arguments like "std=c++17" or "CXXFLAGS=-O0 -g"
	args = extractAssignments(args)
	env.Load() // pick up any environment variables that were just set

	// Handle help/version before parsing
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			printHelp()
			return
		case "--version":
			fmt.Println(versionString)
			return
		}
	}

	// No args = default build
	if len(args) == 0 {
		exitOnErr(NewConfig().Build())
		return
	}

	// Expand legacy compound commands into modifier+action tokens
	var tokens []string
	for _, arg := range args {
		if expanded := expandLegacy(arg); expanded != nil {
			tokens = append(tokens, expanded...)
		} else {
			tokens = append(tokens, arg)
		}
	}

	// Parse tokens into modifiers, action, and trailing args
	cfg := &Config{}
	action := ""
	var actionArgs []string

	for i, tok := range tokens {
		// Once we find an action that takes trailing args, capture the rest
		if action == "run" || action == "pgo" {
			actionArgs = tokens[i:]
			break
		}

		// Apply modifiers
		switch tok {
		case "clang":
			cfg.Clang = true
			continue
		case "zap":
			cfg.Zap = true
			continue
		case "opt":
			if isModifier(tok) && (i < len(tokens)-1 || action != "") {
				cfg.Opt = true
				continue
			}
			// Standalone "opt" with no following action: modifier + default build
			cfg.Opt = true
			continue
		case "strict":
			cfg.Strict = true
			continue
		case "sloppy":
			cfg.Sloppy = true
			continue
		case "small":
			cfg.Small = true
			continue
		case "tiny":
			cfg.Small = true
			cfg.Tiny = true
			continue
		case "win64", "win":
			cfg.Win64 = true
			continue
		case "nosan", "nosanitizers":
			cfg.NoSanitizers = true
			continue
		}

		// "debug" is both a modifier and an action
		if tok == "debug" {
			cfg.Debug = true
			if action == "" {
				action = "debug" // tentative: launch debugger unless overridden
			}
			continue
		}

		// Recognized actions
		if isAction(tok) {
			action = tok
			// For run/pgo/rec, remaining tokens are passed through
			if tok == "run" || tok == "pgo" || tok == "rec" {
				action = tok
				actionArgs = tokens[i+1:]
				break
			}
			continue
		}

		// Unrecognized word: if an action taking args was already set, it's a trailing arg.
		// Otherwise it's an error.
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", tok)
		printHelp()
		os.Exit(1)
	}

	// Default action
	if action == "" {
		action = "build"
	}

	// Dispatch
	switch action {
	case "version":
		fmt.Println(versionString)
	case "build":
		if cfg.Tiny {
			exitOnErr(cfg.TinyBuild())
		} else {
			exitOnErr(cfg.Build())
		}
	case "run":
		if cfg.Tiny {
			if err := cfg.TinyBuild(); err != nil {
				exitOnErr(err)
			}
			// Run the built binary (Build was already called by TinyBuild)
			exitOnErr(cfg.RunBuilt(actionArgs...))
		} else {
			exitOnErr(cfg.Run(actionArgs...))
		}
	case "debug":
		// "debug" as action means: launch the debugger
		exitOnErr(cfg.LaunchDebugger())
	case "rebuild":
		exitOnErr(cfg.Rebuild())
	case "clean":
		cfg.Clean()
	case "fastclean":
		cfg.FastClean()
	case "test":
		exitOnErr(cfg.Test())
	case "testbuild":
		exitOnErr(cfg.TestBuild())
	case "pgo", "rec":
		exitOnErr(cfg.Rec(actionArgs...))
	case "fmt":
		exitOnErr(cfg.Fmt())
	case "generate", "cmakelists", "cmakelist", "cmakelists.txt", "CMakeLists.txt":
		exitOnErr(cfg.Generate())
	case "makefile", "Makefile":
		exitOnErr(cfg.GenerateMakefile())
	case "cmake":
		exitOnErr(cfg.CMakeBuild())
	case "make":
		exitOnErr(cfg.Make())
	case "ninja":
		exitOnErr(cfg.Ninja())
	case "ninjainstall", "ninja_install":
		exitOnErr(cfg.NinjaInstall())
	case "ninjaclean", "ninja_clean":
		cfg.NinjaClean()
	case "makeinstall", "make_install":
		exitOnErr(cfg.CMakeMakeInstall())
	case "makeclean", "make_clean":
		cfg.CMakeMakeClean()
	case "install":
		exitOnErr(cfg.Install())
	case "pkg":
		exitOnErr(cfg.Pkg())
	case "export":
		exitOnErr(cfg.Export())
	case "script":
		exitOnErr(cfg.Script())
	case "valgrind":
		exitOnErr(cfg.Valgrind())
	case "pro":
		exitOnErr(cfg.Pro())
	}
}
