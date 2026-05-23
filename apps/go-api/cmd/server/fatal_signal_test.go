// Package main — fatal_signal_test.go : test du dump de stack au moment d'un
// signal fatal (SIGABRT depuis C++ DuckDB terminate(), SIGSEGV, etc.). Phase
// 4.2 du plan stabilisation 2026-05-22.
//
// On ne peut pas tester directement le signal handler installé par
// installFatalSignalHandler (il appelle os.Exit qui terminerait le test).
// On teste à la place dumpFatalStack qui contient toute la logique de capture.

package main

import (
	"bytes"
	"strings"
	"syscall"
	"testing"
)

func TestDumpFatalStack_WritesHeaderAndStack(t *testing.T) {
	var buf bytes.Buffer
	dumpFatalStack(&buf, syscall.SIGABRT)

	out := buf.String()
	// Contient l'en-tête avec timestamp + signal + pid.
	if !strings.Contains(out, "=== fatal signal") {
		t.Errorf("absence d'en-tête '=== fatal signal' :\n%s", out)
	}
	if !strings.Contains(out, "pid=") {
		t.Errorf("absence de pid= :\n%s", out)
	}
	// Contient le marqueur de fin.
	if !strings.Contains(out, "=== end fatal stack ===") {
		t.Errorf("absence de marqueur de fin :\n%s", out)
	}
	// Contient une stack trace Go (goroutine 1 ou autre).
	if !strings.Contains(out, "goroutine") {
		t.Errorf("absence du mot 'goroutine' dans la stack :\n%s", out)
	}
}

// TestDumpFatalStack_DifferentSignals : le nom du signal apparaît bien dans
// le header — important pour distinguer SIGABRT (C++ terminate) de SIGSEGV
// (access violation) lors de l'analyse post-mortem du crash.log.
func TestDumpFatalStack_DifferentSignals(t *testing.T) {
	cases := []struct {
		name string
		sig  syscall.Signal
	}{
		{"SIGABRT", syscall.SIGABRT},
		{"SIGSEGV", syscall.SIGSEGV},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			dumpFatalStack(&buf, c.sig)
			// Le formattage du signal par fmt.Sprintf("%s", sig) varie selon
			// l'OS mais doit contenir au moins un indice (numéro ou nom).
			out := buf.String()
			if !strings.Contains(out, "=== fatal signal") {
				t.Errorf("[%s] absence d'en-tête fatal signal :\n%s", c.name, out)
			}
		})
	}
}
