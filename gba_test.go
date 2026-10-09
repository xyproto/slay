package slay

import (
	"slices"
	"testing"
)

func TestIsGBAInclude(t *testing.T) {
	for line, want := range map[string]bool{
		"#include <tonc.h>":      true,
		`#include "tonc.h"`:      true,
		"#include <gba.h>":       true,
		"#include <gba_video.h>": true,
		"#include <stdio.h>":     false,
		"#include <SDL3/SDL.h>":  false,
	} {
		if got := isGBAInclude(line); got != want {
			t.Errorf("isGBAInclude(%q) = %v, want %v", line, got, want)
		}
	}
}

func TestMakefileUsesGBA(t *testing.T) {
	if !makefileUsesGBA("LDFLAGS = -specs=gba.specs\n") {
		t.Error("expected gba.specs to be detected")
	}
	if !makefileUsesGBA("include $(DEVKITARM)/gba_rules\n") {
		t.Error("expected gba_rules to be detected")
	}
	if makefileUsesGBA("CC = gcc\n") {
		t.Error("did not expect a plain Makefile to be detected")
	}
}

func TestAssembleGBAFlags(t *testing.T) {
	proj := Project{IsC: true, HasGBA: true, Includes: []string{"tonc.h"}}
	flags := assembleFlags(proj, BuildOptions{})
	if !flags.GBA {
		t.Fatal("expected GBA build flags")
	}
	if flags.Std != "gnu17" {
		t.Errorf("Std = %q, want gnu17", flags.Std)
	}
	for _, want := range []string{"-specs=gba.specs", "-ltonc", "-mthumb"} {
		if !slices.Contains(flags.LDFlags, want) {
			t.Errorf("LDFlags %q is missing %q", flags.LDFlags, want)
		}
	}
	if !slices.Contains(flags.CFlags, "-mcpu=arm7tdmi") {
		t.Errorf("CFlags %q is missing -mcpu=arm7tdmi", flags.CFlags)
	}
}

func TestGBATitle(t *testing.T) {
	for rom, want := range map[string]string{
		"gba-hello.gba":             "GBA-HELLO",
		"a-very-long-game-name.gba": "A-VERY-LONG-",
		"/some/dir/tinyhat.gba":     "TINYHAT",
	} {
		if got := gbaTitle(rom); got != want {
			t.Errorf("gbaTitle(%q) = %q, want %q", rom, got, want)
		}
	}
}

func TestGBAELFName(t *testing.T) {
	if got := gbaELFName("game.gba"); got != "game.elf" {
		t.Errorf("gbaELFName = %q, want game.elf", got)
	}
}
