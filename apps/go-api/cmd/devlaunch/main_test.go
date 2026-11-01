package main

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Variables du processus d'aide : le binaire de test se rejoue lui-même comme lanceur (A),
// copie (B) ou faux serveur, selon ses arguments (patron de os/exec).
const (
	helperEnv      = "DEVLAUNCH_TEST_HELPER"
	startedFileEnv = "DEVLAUNCH_TEST_STARTED"
	eofFileEnv     = "DEVLAUNCH_TEST_EOF"
)

func TestMain(m *testing.M) {
	if os.Getenv(helperEnv) != "1" {
		os.Exit(m.Run())
	}
	if len(os.Args) > 1 && os.Args[1] == stdinStopFlag {
		os.Exit(fakeServer())
	}
	os.Exit(run(os.Args[1:]))
}

// fakeServer note son démarrage, lit stdin jusqu'à EOF puis note "eof" : la preuve que le
// serveur a survécu à la mort du lanceur et vu la fermeture du tube.
func fakeServer() int {
	if err := os.WriteFile(os.Getenv(startedFileEnv), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		return 1
	}
	buf := make([]byte, 64)
	for {
		if _, err := os.Stdin.Read(buf); err != nil {
			break
		}
	}
	if err := os.WriteFile(os.Getenv(eofFileEnv), []byte("eof"), 0o600); err != nil {
		return 1
	}
	return 0
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitFile(t *testing.T, path string, within time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

// killTree tue le lanceur comme air : arbre entier sous Windows (TASKKILL /F /T), processus
// seul ailleurs.
func killTree(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if runtime.GOOS == "windows" {
		out, err := exec.Command("TASKKILL", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).CombinedOutput()
		if err != nil {
			t.Fatalf("TASKKILL : %v\n%s", err, out)
		}
	} else if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
}

func helperEnvFor(dir string, port int) []string {
	return append(os.Environ(),
		helperEnv+"=1",
		startedFileEnv+"="+filepath.Join(dir, "started"),
		eofFileEnv+"="+filepath.Join(dir, "eof"),
		"LEVELUP_API_PORT="+strconv.Itoa(port),
	)
}

// Tuer le lanceur (arbre compris) ne tue pas le serveur : il voit EOF sur son stdin et sort seul.
func TestLaunch_TuerLeLanceurFermeLeStdinDuServeurSansLeTuer(t *testing.T) {
	dir := t.TempDir()
	a := exec.Command(os.Args[0], os.Args[0])
	a.Env = helperEnvFor(dir, freePort(t))
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Process.Kill() })

	if !waitFile(t, filepath.Join(dir, "started"), 10*time.Second) {
		t.Fatal("le faux serveur n'a pas démarré")
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(dir, "eof")); err == nil {
		t.Fatal("le serveur a vu EOF alors que le lanceur est vivant")
	}

	killTree(t, a)
	if !waitFile(t, filepath.Join(dir, "eof"), 10*time.Second) {
		t.Fatal("le serveur n'a pas vu EOF après la mort du lanceur (tué avec lui, ou stdin resté ouvert)")
	}
}

func TestLaunch_ServeurIntrouvableSortEnErreur(t *testing.T) {
	dir := t.TempDir()
	a := exec.Command(os.Args[0], filepath.Join(dir, "absent.exe"))
	a.Env = helperEnvFor(dir, freePort(t))
	out, err := a.CombinedOutput()
	if err == nil {
		t.Fatal("sortie à 0 alors que le serveur n'existe pas")
	}
	if !strings.Contains(string(out), "lancement du serveur") {
		t.Errorf("message sans cause lisible : %q", out)
	}
}

func TestRun_SansArgumentAfficheLUsage(t *testing.T) {
	if code := run(nil); code != 2 {
		t.Errorf("code %d, attendu 2", code)
	}
}

func TestWaitPortFree(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	go func() { // accepte pour que le sondage voie un port vivant
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()

	if err := waitPortFree(port, 150*time.Millisecond, 20*time.Millisecond); err == nil {
		t.Error("port occupé jusqu'au bout : une erreur de délai était attendue")
	}

	released := make(chan error, 1)
	go func() { released <- waitPortFree(port, 5*time.Second, 20*time.Millisecond) }()
	time.Sleep(100 * time.Millisecond)
	_ = ln.Close()
	select {
	case err := <-released:
		if err != nil {
			t.Errorf("port libéré mais erreur : %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waitPortFree n'a pas vu le port se libérer")
	}
}

func TestApiPort(t *testing.T) {
	for in, want := range map[string]int{"": 8000, "abc": 8000, "0": 8000, "70000": 8000, "8123": 8123} {
		t.Setenv("LEVELUP_API_PORT", in)
		if got := apiPort(); got != want {
			t.Errorf("LEVELUP_API_PORT=%q : %d, attendu %d", in, got, want)
		}
	}
}
