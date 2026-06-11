// Package ops — logtail_test.go : lecture par la fin, filtres, parse
// best-effort, anti-traversal (fichiers temporaires — pas de CGO requis).
package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeLogFile écrit des lignes JSON slog horodatées minute par minute
// (la ligne i est à base+i minutes — la DERNIÈRE ligne est la plus récente).
func writeLogFile(t *testing.T, dir, module string, lines []string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, module+".log"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
}

func jsonLine(ts time.Time, level, msg string, extra string) string {
	base := fmt.Sprintf(`{"time":"%s","level":"%s","msg":"%s"`, ts.Format(time.RFC3339Nano), level, msg)
	if extra != "" {
		base += "," + extra
	}
	return base + "}"
}

func TestLogTail_OrderNewestFirstAndLimit(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, jsonLine(base.Add(time.Duration(i)*time.Minute), "INFO", fmt.Sprintf("msg-%d", i), ""))
	}
	writeLogFile(t, dir, "sync", lines)

	res, err := TailModuleLog(dir, "sync", LogTailOptions{N: 3})
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(res.Entries) != 3 {
		t.Fatalf("len = %d (attendu 3)", len(res.Entries))
	}
	if res.Entries[0].Msg != "msg-9" || res.Entries[2].Msg != "msg-7" {
		t.Fatalf("ordre inattendu : %s … %s (attendu msg-9 → msg-7)", res.Entries[0].Msg, res.Entries[2].Msg)
	}
	if res.Truncated {
		t.Error("Truncated ne doit pas être posé quand N est atteint")
	}
}

func TestLogTail_LevelThresholdAndContains(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	writeLogFile(t, dir, "auth", []string{
		jsonLine(base, "DEBUG", "verbose detail", ""),
		jsonLine(base.Add(time.Minute), "INFO", "token refresh ok", `"request_id":"req-42"`),
		jsonLine(base.Add(2*time.Minute), "WARN", "legacy source used", ""),
		jsonLine(base.Add(3*time.Minute), "ERROR", "oauth revoked for JGtm", `"err":"AADSTS70000"`),
	})

	warnUp, err := TailModuleLog(dir, "auth", LogTailOptions{Level: "warn"})
	if err != nil || len(warnUp.Entries) != 2 {
		t.Fatalf("level warn : %d entrées, err=%v (attendu 2)", len(warnUp.Entries), err)
	}
	if warnUp.Entries[0].Level != "error" || warnUp.Entries[1].Level != "warn" {
		t.Errorf("niveaux inattendus : %s, %s", warnUp.Entries[0].Level, warnUp.Entries[1].Level)
	}

	// contains insensible à la casse sur err + request_id.
	byErr, _ := TailModuleLog(dir, "auth", LogTailOptions{Contains: "aadsts"})
	if len(byErr.Entries) != 1 || byErr.Entries[0].Msg != "oauth revoked for JGtm" {
		t.Fatalf("contains err : %+v", byErr.Entries)
	}
	byReq, _ := TailModuleLog(dir, "auth", LogTailOptions{Contains: "REQ-42"})
	if len(byReq.Entries) != 1 {
		t.Fatalf("contains request_id : %d entrées (attendu 1)", len(byReq.Entries))
	}
}

func TestLogTail_SinceEarlyExit(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	var lines []string
	for i := 0; i < 6; i++ {
		lines = append(lines, jsonLine(base.Add(time.Duration(i)*time.Minute), "INFO", fmt.Sprintf("m%d", i), ""))
	}
	writeLogFile(t, dir, "scheduler", lines)

	since := base.Add(4 * time.Minute)
	res, err := TailModuleLog(dir, "scheduler", LogTailOptions{Since: since})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Entries) != 2 { // m5, m4
		t.Fatalf("since : %d entrées (attendu 2) %+v", len(res.Entries), res.Entries)
	}
}

func TestLogTail_MalformedLineBecomesRaw(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	writeLogFile(t, dir, "duckdb", []string{
		jsonLine(base, "INFO", "ok line", `"custom_field":42`),
		"panic: runtime error — pas du JSON",
	})

	res, err := TailModuleLog(dir, "duckdb", LogTailOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Entries) != 2 {
		t.Fatalf("len = %d (attendu 2 — la ligne brute ne doit pas faire échouer)", len(res.Entries))
	}
	if res.Entries[0].Level != "unknown" || !strings.Contains(res.Entries[0].Raw, "panic") {
		t.Errorf("ligne brute inattendue : %+v", res.Entries[0])
	}
	if res.Entries[1].Fields["custom_field"] == nil {
		t.Errorf("attrs restants attendus dans Fields : %+v", res.Entries[1].Fields)
	}
}

// TestLogTail_ChunkBoundary : un fichier plus grand qu'un chunk se lit
// correctement (lignes reconstruites à cheval sur les frontières de chunks).
func TestLogTail_ChunkBoundary(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	padding := strings.Repeat("x", 700)
	var lines []string
	for i := 0; i < 800; i++ { // ~570 KiB > 2 chunks de 256 KiB
		lines = append(lines, jsonLine(base.Add(time.Duration(i)*time.Second), "INFO",
			fmt.Sprintf("line-%04d %s", i, padding), ""))
	}
	writeLogFile(t, dir, "http", lines)

	res, err := TailModuleLog(dir, "http", LogTailOptions{N: 600})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Entries) != 600 {
		t.Fatalf("len = %d (attendu 600)", len(res.Entries))
	}
	for i, e := range res.Entries {
		want := fmt.Sprintf("line-%04d", 799-i)
		if !strings.HasPrefix(e.Msg, want) {
			t.Fatalf("entrée %d : msg %q (attendu préfixe %q) — ligne corrompue à la frontière de chunk", i, e.Msg[:20], want)
		}
	}
}

func TestLogTail_ModuleValidation(t *testing.T) {
	dir := t.TempDir()
	for _, bad := range []string{"../etc/passwd", "a b", "UPPER", "", "x/y"} {
		if _, err := TailModuleLog(dir, bad, LogTailOptions{}); err == nil {
			t.Errorf("module %q : erreur attendue (anti-traversal)", bad)
		}
	}
}

func TestListLogModules(t *testing.T) {
	dir := t.TempDir()
	writeLogFile(t, dir, "sync", []string{jsonLine(time.Now(), "INFO", "a", "")})
	writeLogFile(t, dir, "auth", []string{jsonLine(time.Now(), "INFO", "bb", ""), jsonLine(time.Now(), "INFO", "cc", "")})
	// Fichier de rotation lumberjack (timestampé, majuscules/points) → exclu.
	if err := os.WriteFile(filepath.Join(dir, "sync-2026-06-11T10-00-00.000.log"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	mods, err := ListLogModules(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 2 {
		t.Fatalf("len = %d (attendu 2 — rotation et .txt exclus) %+v", len(mods), mods)
	}
	if mods[0].Module != "auth" { // plus gros d'abord
		t.Errorf("tri par taille attendu, got %+v", mods)
	}
}
