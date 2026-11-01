package slay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssemblyTarget(t *testing.T) {
	tests := []struct {
		name, source, want string
	}{
		{"c64", `CHROUT = $FFD2
BORDER = $D020
start:  lda #0
        sta BORDER
        ldx #0
loop:   lda text,x
        beq done
        jsr CHROUT
        inx
        bne loop
done:   rts
`, "c64"},
		{"c128", `MMU = $FF00
start:  lda #$0E
        sta MMU
        lda #0
        sta $D020
        jsr $FFD2
        rts
`, "c128"},
		{"vic20", `start:  lda #8
        sta $900F
        ldx #0
loop:   lda text,x
        beq done
        jsr $FFD2
        inx
        bne loop
done:   rts
`, "vic20"},
		{"nes", `.segment "HEADER"
.byte "NES", $1A
reset:  sei
        ldx #$FF
        lda #0
        sta $2000
        sta $2001
        lda #$40
        sta $4017
        jmp reset
`, "nes"},
		{"snes", `.p816
        sep #$20
        lda #$0F
        sta $2100
        lda #$80
        sta $2115
        rts
`, "snes"},
		{"x86", `org 100h
start:
        mov ax, 13h
        int 10h
        jmp start
`, ""},
		{"c64 bitmap is not nes", `start:  lda #$3B
        sta $D011
        lda #0
        sta $2000,x
        inx
        bne start
        rts
`, "c64"},
	}
	for _, tt := range tests {
		if got := AssemblyTarget(tt.source); got != tt.want {
			t.Errorf("%s: AssemblyTarget() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestCC65IncludeTarget(t *testing.T) {
	for line, want := range map[string]string{
		"#include <c64.h>":   "c64",
		"#include <cbm.h>":   "c64",
		"#include <c128.h>":  "c128",
		`#include "vic20.h"`: "vic20",
		"#include <nes.h>":   "nes",
		"#include <stdio.h>": "",
	} {
		if got := cc65IncludeTarget(line); got != want {
			t.Errorf("cc65IncludeTarget(%q) = %q, want %q", line, got, want)
		}
	}
	if got := makefileCC65Target("\tcl65 -t vic20 -o hello.prg hello.c\n"); got != "vic20" {
		t.Errorf("makefileCC65Target = %q, want vic20", got)
	}
}

func TestWriteSNESChecksum(t *testing.T) {
	rom := filepath.Join(t.TempDir(), "test.sfc")
	data := make([]byte, 0x8000)
	for i := range data {
		data[i] = byte(i)
	}
	if err := os.WriteFile(rom, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSNESChecksum(rom); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rom)
	if err != nil {
		t.Fatal(err)
	}
	complement := uint16(data[0x7FDC]) | uint16(data[0x7FDD])<<8
	checksum := uint16(data[0x7FDE]) | uint16(data[0x7FDF])<<8
	if complement^checksum != 0xFFFF {
		t.Errorf("checksum %04X and complement %04X do not match", checksum, complement)
	}
	var sum uint16
	for _, b := range data {
		sum += uint16(b)
	}
	if sum != checksum {
		t.Errorf("checksum is %04X, but the ROM sums to %04X", checksum, sum)
	}
}
