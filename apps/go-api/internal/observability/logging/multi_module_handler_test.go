package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
)

// readLog ouvre logs/{module}.log et retourne son contenu en string.
func readLog(t *testing.T, dir, module string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, module+".log"))
	if err != nil {
		t.Fatalf("read %s.log: %v", module, err)
	}
	return string(data)
}

// TestMultiModuleHandler_WritesToConsoleAndFile : un log Info doit apparaître
// à la fois dans le buffer console ET dans logs/{module}.log.
func TestMultiModuleHandler_WritesToConsoleAndFile(t *testing.T) {
	dir := t.TempDir()
	var consoleBuf bytes.Buffer
	console := slog.NewTextHandler(&consoleBuf, &slog.HandlerOptions{Level: slog.LevelInfo})

	mh, err := NewMultiModuleHandler(console, dir, slog.LevelInfo, DefaultRotationPolicy())
	if err != nil {
		t.Fatalf("NewMultiModuleHandler: %v", err)
	}
	defer mh.Close()

	logger := slog.New(mh)
	// Forcer le module via attribut explicit (pas dépendant du PC d'appel).
	logger.With("module", "sync").Info("test message", "k", "v")

	// Console
	if !strings.Contains(consoleBuf.String(), "test message") {
		t.Errorf("console missing 'test message': %q", consoleBuf.String())
	}
	// File
	fileContent := readLog(t, dir, "sync")
	if !strings.Contains(fileContent, "test message") {
		t.Errorf("file missing 'test message': %q", fileContent)
	}
	// Le fichier doit être en JSON
	var parsed map[string]any
	for _, line := range strings.Split(strings.TrimSpace(fileContent), "\n") {
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Errorf("fichier non-JSON: %v\nline: %s", err, line)
		}
		if parsed["msg"] != "test message" {
			t.Errorf("msg = %v, want 'test message'", parsed["msg"])
		}
		if parsed["k"] != "v" {
			t.Errorf("attribut k absent ou différent: %v", parsed["k"])
		}
	}
}

// TestMultiModuleHandler_ModuleAttributeRouting : un log avec module=foo
// va dans logs/foo.log, un autre avec module=bar va dans logs/bar.log.
func TestMultiModuleHandler_ModuleAttributeRouting(t *testing.T) {
	dir := t.TempDir()
	console := slog.NewTextHandler(&bytes.Buffer{}, nil)
	mh, _ := NewMultiModuleHandler(console, dir, slog.LevelInfo, DefaultRotationPolicy())
	defer mh.Close()
	logger := slog.New(mh)

	logger.With("module", "sync").Info("sync log")
	logger.With("module", "provider").Info("provider log")
	logger.With("module", "pool").Info("pool log")

	syncContent := readLog(t, dir, "sync")
	providerContent := readLog(t, dir, "provider")
	poolContent := readLog(t, dir, "pool")

	if !strings.Contains(syncContent, "sync log") {
		t.Error("sync.log missing sync message")
	}
	if strings.Contains(syncContent, "provider log") {
		t.Error("sync.log contains provider message (leak)")
	}
	if !strings.Contains(providerContent, "provider log") {
		t.Error("provider.log missing provider message")
	}
	if !strings.Contains(poolContent, "pool log") {
		t.Error("pool.log missing pool message")
	}
}

// TestMultiModuleHandler_EventIDPropagation : un log emit via ctx avec event_id
// doit avoir event_id présent dans le fichier (ContextHandler doit être chaîné).
func TestMultiModuleHandler_EventIDPropagation(t *testing.T) {
	dir := t.TempDir()
	var consoleBuf bytes.Buffer
	rawConsole := slog.NewJSONHandler(&consoleBuf, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Chaîne typique de prod : ContextHandler → MultiModuleHandler.
	// On simule ça en wrappant le mh dans la chaîne.
	mh, _ := NewMultiModuleHandler(rawConsole, dir, slog.LevelInfo, DefaultRotationPolicy())
	defer mh.Close()

	// On instancie un context handler "inverse" : il est usually au-dessus
	// mais pour simplifier le test, on injecte event_id directement dans
	// le record via slog.With(slog.String("event_id", ...)).
	ctx := ctxkeys.WithEventID(context.Background(), "sync.RunDelta:abc123")
	_ = ctx // pour montrer l'usage typique

	// Simuler : un caller ajoute event_id via With (équivalent à ce que fait
	// ContextHandler en prod).
	logger := slog.New(mh).With(
		"module", "sync",
		"event_id", "sync.RunDelta:abc123",
	)
	logger.Info("event log")

	syncContent := readLog(t, dir, "sync")
	if !strings.Contains(syncContent, "sync.RunDelta:abc123") {
		t.Errorf("event_id absent du fichier: %s", syncContent)
	}
	if !strings.Contains(consoleBuf.String(), "sync.RunDelta:abc123") {
		t.Errorf("event_id absent de console: %s", consoleBuf.String())
	}
}

// TestMultiModuleHandler_FallbackGeneral : un log sans attribut module
// (et avec PC=0 simulé) tombe sur "general.log".
func TestMultiModuleHandler_FallbackGeneral(t *testing.T) {
	dir := t.TempDir()
	console := slog.NewTextHandler(&bytes.Buffer{}, nil)
	mh, _ := NewMultiModuleHandler(console, dir, slog.LevelInfo, DefaultRotationPolicy())
	defer mh.Close()

	// Construire un record manuellement avec PC=0 pour forcer le fallback.
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "fallback test", 0)
	_ = mh.Handle(context.Background(), rec)

	generalContent := readLog(t, dir, "general")
	if !strings.Contains(generalContent, "fallback test") {
		t.Errorf("general.log missing: %s", generalContent)
	}
}

// TestMultiModuleHandler_LevelFiltering : un log Debug ne va pas en fichier
// si FileLevel=Info, mais va en console si console=Debug.
func TestMultiModuleHandler_LevelFiltering(t *testing.T) {
	dir := t.TempDir()
	var consoleBuf bytes.Buffer
	console := slog.NewTextHandler(&consoleBuf, &slog.HandlerOptions{Level: slog.LevelDebug})

	mh, _ := NewMultiModuleHandler(console, dir, slog.LevelInfo, DefaultRotationPolicy())
	defer mh.Close()

	logger := slog.New(mh)
	logger.With("module", "sync").Debug("debug msg")

	if !strings.Contains(consoleBuf.String(), "debug msg") {
		t.Error("console missing debug (console level=Debug)")
	}
	// File doit être absent OU vide
	path := filepath.Join(dir, "sync.log")
	if _, err := os.Stat(path); err == nil {
		// Fichier existe : il doit être vide ou ne pas contenir "debug msg"
		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), "debug msg") {
			t.Errorf("file contient debug alors que FileLevel=Info: %s", data)
		}
	}
}

// TestMultiModuleHandler_ConsoleQuieterThanFile : console au niveau WARN et fichiers
// au niveau INFO. Les INFO doivent partir dans le fichier mais PAS dans la console.
// Régression : avant le fix, MultiModuleHandler.Handle appelait console.Handle()
// sans re-vérifier console.Enabled, donc le filtre console était bypassé dès que
// le fichier acceptait le record → INFO leak dans le terminal en prod.
func TestMultiModuleHandler_ConsoleQuieterThanFile(t *testing.T) {
	dir := t.TempDir()
	var consoleBuf bytes.Buffer
	console := slog.NewTextHandler(&consoleBuf, &slog.HandlerOptions{Level: slog.LevelWarn})

	mh, err := NewMultiModuleHandler(console, dir, slog.LevelInfo, DefaultRotationPolicy())
	if err != nil {
		t.Fatalf("NewMultiModuleHandler: %v", err)
	}
	defer mh.Close()

	logger := slog.New(mh)
	logger.With("module", "sync").Info("info-should-be-file-only")
	logger.With("module", "sync").Warn("warn-should-be-everywhere")

	consoleOut := consoleBuf.String()
	if strings.Contains(consoleOut, "info-should-be-file-only") {
		t.Errorf("console a reçu INFO alors que son niveau est WARN. Console output:\n%s", consoleOut)
	}
	if !strings.Contains(consoleOut, "warn-should-be-everywhere") {
		t.Errorf("console manque le WARN. Console output:\n%s", consoleOut)
	}

	fileData, err := os.ReadFile(filepath.Join(dir, "sync.log"))
	if err != nil {
		t.Fatalf("read sync.log: %v", err)
	}
	if !strings.Contains(string(fileData), "info-should-be-file-only") {
		t.Errorf("fichier manque l'INFO alors que FileLevel=Info. File output:\n%s", fileData)
	}
	if !strings.Contains(string(fileData), "warn-should-be-everywhere") {
		t.Errorf("fichier manque le WARN. File output:\n%s", fileData)
	}
}

// TestMultiModuleHandler_Close_Idempotent : Close peut être appelé plusieurs fois sans crash.
func TestMultiModuleHandler_Close_Idempotent(t *testing.T) {
	dir := t.TempDir()
	console := slog.NewTextHandler(&bytes.Buffer{}, nil)
	mh, _ := NewMultiModuleHandler(console, dir, slog.LevelInfo, DefaultRotationPolicy())
	slog.New(mh).With("module", "test").Info("trigger file creation")

	if err := mh.Close(); err != nil {
		t.Errorf("1st close: %v", err)
	}
	if err := mh.Close(); err != nil {
		t.Errorf("2nd close (idempotent attendu): %v", err)
	}
}

// TestSanitizeModuleName : noms de modules avec caractères dangereux normalisés.
func TestSanitizeModuleName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "general"},
		{"sync", "sync"},
		{"SYNC", "sync"},
		{"Sync.Engine", "sync_engine"},
		{"foo/bar", "foo_bar"},
		{"..", "__"},
		{"already-ok", "already-ok"},
		{"with_underscore", "with_underscore"},
	}
	for _, c := range cases {
		got := SanitizeModuleName(c.in)
		if got != c.want {
			t.Errorf("SanitizeModuleName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestWithEvent_RoundTrip : WithEvent met l'id dans le ctx, CurrentEvent le lit.
func TestWithEvent_RoundTrip(t *testing.T) {
	ctx := context.Background()
	ctx, id := WithEvent(ctx, "test.scenario")
	if id == "" {
		t.Fatal("WithEvent returned empty id")
	}
	if !strings.HasPrefix(id, "test.scenario:") {
		t.Errorf("id = %q, want prefix 'test.scenario:'", id)
	}
	if got := CurrentEvent(ctx); got != id {
		t.Errorf("CurrentEvent = %q, want %q", got, id)
	}
}

// TestWithEvent_UniqueIDs : 2 appels successifs produisent des ids différents.
func TestWithEvent_UniqueIDs(t *testing.T) {
	ctx := context.Background()
	_, id1 := WithEvent(ctx, "")
	_, id2 := WithEvent(ctx, "")
	if id1 == id2 {
		t.Errorf("expected unique ids, got %q twice", id1)
	}
}

// TestLoadConfig_Defaults : sans env vars, config sane.
func TestLoadConfig_Defaults(t *testing.T) {
	// Nettoyer les env vars pour éviter pollution
	t.Setenv("LEVELUP_LOGS_ENABLED", "")
	t.Setenv("LEVELUP_LOGS_DIR", "")
	t.Setenv("LEVELUP_LOGS_FILE_LEVEL", "")

	cfg := LoadConfig("/tmp/repo")
	if !cfg.Enabled {
		t.Error("Enabled default = false, want true")
	}
	if cfg.LogsDir == "" {
		t.Error("LogsDir empty avec repoRoot")
	}
	if !strings.HasSuffix(cfg.LogsDir, "logs") {
		t.Errorf("LogsDir = %q, want suffix 'logs'", cfg.LogsDir)
	}
	if cfg.FileLevel != slog.LevelInfo {
		t.Errorf("FileLevel = %v, want Info", cfg.FileLevel)
	}
}

// TestLoadConfig_Disabled : kill-switch via env.
func TestLoadConfig_Disabled(t *testing.T) {
	t.Setenv("LEVELUP_LOGS_ENABLED", "false")
	cfg := LoadConfig("/tmp")
	if cfg.Enabled {
		t.Error("Enabled = true malgré LEVELUP_LOGS_ENABLED=false")
	}
	if cfg.LogsDir != "" {
		t.Errorf("LogsDir = %q quand disabled, want empty", cfg.LogsDir)
	}
}
