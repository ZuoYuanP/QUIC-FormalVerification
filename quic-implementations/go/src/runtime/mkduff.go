// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// runtime·duffzero is a Duff's device for zeroing memory.
// The compiler jumps to computed addresses within
// the routine to zero chunks of memory.
// Do not change duffzero without also
// changing the uses in cmd/compile/internal/*/*.go.

// runtime·duffcopy is a Duff's device for copying memory.
// The compiler jumps to computed addresses within
// the routine to copy chunks of memory.
// Source and destination must not overlap.
// Do not change duffcopy without also
// changing the uses in cmd/compile/internal/*/*.go.

// See the zero* and copy* generators below
// for architecture-specific comments.

// mkduff generates duff_*.s.
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
)

func main() {
	gen("amd64", notags, zeroAMD64, copyAMD64)
	gen("386", notags, zero386, copy386)
	gen("arm", notags, zeroARM, copyARM)
	gen("arm64", notags, zeroARM64, copyARM64)
	gen("ppc64x", tagsPPC64x, zeroPPC64x, copyPPC64x)
	gen("mips64x", tagsMIPS64x, zeroMIPS64x, copyMIPS64x)
}

func gen(arch string, tags, zero, copy func(io.Writer)) {
	var buf bytes.Buffer

	fmt.Fprintln(&buf, "// Code generated by mkduff.go; DO NOT EDIT.")
	fmt.Fprintln(&buf, "// Run go generate from src/runtime to update.")
	fmt.Fprintln(&buf, "// See mkduff.go for comments.")
	tags(&buf)
	fmt.Fprintln(&buf, "#include \"textflag.h\"")
	fmt.Fprintln(&buf)
	zero(&buf)
	fmt.Fprintln(&buf)
	copy(&buf)

	if err := ioutil.WriteFile("duff_"+arch+".s", buf.Bytes(), 0644); err != nil {
		log.Fatalln(err)
	}
}

func notags(w io.Writer) { fmt.Fprintln(w) }

func zeroAMD64(w io.Writer) {
	// X0: zero
	// DI: ptr to memory to be zeroed
	// DI is updated as a side effect.
	fmt.Fprintln(w, "TEXT runtime·duffzero(SB), NOSPLIT, $0-0")
	for i := 0; i < 16; i++ {
		fmt.Fprintln(w, "\tMOVUPS\tX0,(DI)")
		fmt.Fprintln(w, "\tMOVUPS\tX0,16(DI)")
		fmt.Fprintln(w, "\tMOVUPS\tX0,32(DI)")
		fmt.Fprintln(w, "\tMOVUPS\tX0,48(DI)")
		fmt.Fprintln(w, "\tLEAQ\t64(DI),DI") // We use lea instead of add, to avoid clobbering flags
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "\tRET")
}

func copyAMD64(w io.Writer) {
	// SI: ptr to source memory
	// DI: ptr to destination memory
	// SI and DI are updated as a side effect.
	//
	// This is equivalent to a sequence of MOVSQ but
	// for some reason that is 3.5x slower than this code.
	// The STOSQ in duffzero seem fine, though.
	fmt.Fprintln(w, "TEXT runtime·duffcopy(SB), NOSPLIT, $0-0")
	for i := 0; i < 64; i++ {
		fmt.Fprintln(w, "\tMOVUPS\t(SI), X0")
		fmt.Fprintln(w, "\tADDQ\t$16, SI")
		fmt.Fprintln(w, "\tMOVUPS\tX0, (DI)")
		fmt.Fprintln(w, "\tADDQ\t$16, DI")
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "\tRET")
}

func zero386(w io.Writer) {
	// AX: zero
	// DI: ptr to memory to be zeroed
	// DI is updated as a side effect.
	fmt.Fprintln(w, "TEXT runtime·duffzero(SB), NOSPLIT, $0-0")
	for i := 0; i < 128; i++ {
		fmt.Fprintln(w, "\tSTOSL")
	}
	fmt.Fprintln(w, "\tRET")
}

func copy386(w io.Writer) {
	// SI: ptr to source memory
	// DI: ptr to destination memory
	// SI and DI are updated as a side effect.
	//
	// This is equivalent to a sequence of MOVSL but
	// for some reason MOVSL is really slow.
	fmt.Fprintln(w, "TEXT runtime·duffcopy(SB), NOSPLIT, $0-0")
	for i := 0; i < 128; i++ {
		fmt.Fprintln(w, "\tMOVL\t(SI), CX")
		fmt.Fprintln(w, "\tADDL\t$4, SI")
		fmt.Fprintln(w, "\tMOVL\tCX, (DI)")
		fmt.Fprintln(w, "\tADDL\t$4, DI")
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "\tRET")
}

func zeroARM(w io.Writer) {
	// R0: zero
	// R1: ptr to memory to be zeroed
	// R1 is updated as a side effect.
	fmt.Fprintln(w, "TEXT runtime·duffzero(SB), NOSPLIT, $0-0")
	for i := 0; i < 128; i++ {
		fmt.Fprintln(w, "\tMOVW.P\tR0, 4(R1)")
	}
	fmt.Fprintln(w, "\tRET")
}

func copyARM(w io.Writer) {
	// R0: scratch space
	// R1: ptr to source memory
	// R2: ptr to destination memory
	// R1 and R2 are updated as a side effect
	fmt.Fprintln(w, "TEXT runtime·duffcopy(SB), NOSPLIT, $0-0")
	for i := 0; i < 128; i++ {
		fmt.Fprintln(w, "\tMOVW.P\t4(R1), R0")
		fmt.Fprintln(w, "\tMOVW.P\tR0, 4(R2)")
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "\tRET")
}

func zeroARM64(w io.Writer) {
	// ZR: always zero
	// R20: ptr to memory to be zeroed
	// On return, R20 points to the last zeroed dword.
	fmt.Fprintln(w, "TEXT runtime·duffzero(SB), NOSPLIT|NOFRAME, $0-0")
	for i := 0; i < 63; i++ {
		fmt.Fprintln(w, "\tSTP.P\t(ZR, ZR), 16(R20)")
	}
	fmt.Fprintln(w, "\tSTP\t(ZR, ZR), (R20)")
	fmt.Fprintln(w, "\tRET")
}

func copyARM64(w io.Writer) {
	// R20: ptr to source memory
	// R21: ptr to destination memory
	// R26, R27 (aka REGTMP): scratch space
	// R20 and R21 are updated as a side effect
	fmt.Fprintln(w, "TEXT runtime·duffcopy(SB), NOSPLIT|NOFRAME, $0-0")

	for i := 0; i < 64; i++ {
		fmt.Fprintln(w, "\tLDP.P\t16(R20), (R26, R27)")
		fmt.Fprintln(w, "\tSTP.P\t(R26, R27), 16(R21)")
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "\tRET")
}

func tagsPPC64x(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "// +build ppc64 ppc64le")
	fmt.Fprintln(w)
}

func zeroPPC64x(w io.Writer) {
	// R0: always zero
	// R3 (aka REGRT1): ptr to memory to be zeroed - 8
	// On return, R3 points to the last zeroed dword.
	fmt.Fprintln(w, "TEXT runtime·duffzero(SB), NOSPLIT|NOFRAME, $0-0")
	for i := 0; i < 128; i++ {
		fmt.Fprintln(w, "\tMOVDU\tR0, 8(R3)")
	}
	fmt.Fprintln(w, "\tRET")
}

func copyPPC64x(w io.Writer) {
	fmt.Fprintln(w, "// TODO: Implement runtime·duffcopy.")
}

func tagsMIPS64x(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "// +build mips64 mips64le")
	fmt.Fprintln(w)
}

func zeroMIPS64x(w io.Writer) {
	// R0: always zero
	// R1 (aka REGRT1): ptr to memory to be zeroed - 8
	// On return, R1 points to the last zeroed dword.
	fmt.Fprintln(w, "TEXT runtime·duffzero(SB), NOSPLIT|NOFRAME, $0-0")
	for i := 0; i < 128; i++ {
		fmt.Fprintln(w, "\tMOVV\tR0, 8(R1)")
		fmt.Fprintln(w, "\tADDV\t$8, R1")
	}
	fmt.Fprintln(w, "\tRET")
}

func copyMIPS64x(w io.Writer) {
	fmt.Fprintln(w, "TEXT runtime·duffcopy(SB), NOSPLIT|NOFRAME, $0-0")
	for i := 0; i < 128; i++ {
		fmt.Fprintln(w, "\tMOVV\t(R1), R23")
		fmt.Fprintln(w, "\tADDV\t$8, R1")
		fmt.Fprintln(w, "\tMOVV\tR23, (R2)")
		fmt.Fprintln(w, "\tADDV\t$8, R2")
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "\tRET")
}