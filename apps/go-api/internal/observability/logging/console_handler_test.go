package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// newConsoleHandler crée un handler aux options par défaut sur un buffer.
// Helper interne pour réduire le boilerplate des tests.
func newConsoleHandler(buf *bytes.Buffer, opts ConsoleHandlerOptions) *slog.Logger {
	h := NewConsoleHandler(buf, opts)
	return slog.New(h)
}

// TestConsoleHandler_FormatCompact : un Info simple doit produire
// `HH:MM:SS [INFO]  msg k=v\n` (alignement [INFO] = 7 chars padded).
func TestConsoleHandler_FormatCompact(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})
	logger.Info("sync.postSync: pipeline démarré", "matches_inserted", 3)

	line := strings.TrimRight(buf.String(), "\n")
	// Time HH:MM:SS = 8 chars + ' '
	if len(line) < 9 || line[2] != ':' || line[5] != ':' {
		t.Errorf("format temps incorrect: %q", line)
	}
	if !strings.Contains(line, "[INFO] ") {
		t.Errorf("label [INFO] absent: %q", line)
	}
	if !strings.Contains(line, "sync.postSync: pipeline démarré") {
		t.Errorf("message absent: %q", line)
	}
	if !strings.Contains(line, "matches_inserted=3") {
		t.Errorf("attr key=val absent: %q", line)
	}
}

// TestConsoleHandler_LevelPadding : ERROR/WARN/INFO/DEBUG occupent 7 chars
// pour aligner verticalement la colonne message.
func TestConsoleHandler_LevelPadding(t *testing.T) {
	cases := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelError, "[ERROR]"},
		{slog.LevelWarn, "[WARN] "},
		{slog.LevelInfo, "[INFO] "},
		{slog.LevelDebug, "[DEBUG]"},
	}
	for _, c := range cases {
		got := formatLevel(c.level, false)
		if got != c.want {
			t.Errorf("formatLevel(%v) = %q, want %q (len=%d, want 7)", c.level, got, c.want, len(got))
		}
		if len(got) != 7 {
			t.Errorf("formatLevel(%v) len = %d, want 7 (alignement vertical)", c.level, len(got))
		}
	}
}

// TestConsoleHandler_LevelFiltering : Debug filtré si Level=Info.
func TestConsoleHandler_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})
	logger.Debug("ne doit pas apparaître")
	logger.Info("doit apparaître")

	out := buf.String()
	if strings.Contains(out, "ne doit pas") {
		t.Errorf("debug a fui sur Info-level handler: %q", out)
	}
	if !strings.Contains(out, "doit apparaître") {
		t.Errorf("info absent: %q", out)
	}
}

// TestConsoleHandler_SkipAttrsDefault : event_id/request_id/source.* masqués
// par défaut sur console (préservés ailleurs).
func TestConsoleHandler_SkipAttrsDefault(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})
	logger.Info("test",
		"event_id", "sync.RunDelta:abc123",
		"request_id", "req-42",
		"source.function", "main.go:120",
		"keep", "visible",
	)

	out := buf.String()
	if strings.Contains(out, "event_id") {
		t.Errorf("event_id non masqué: %q", out)
	}
	if strings.Contains(out, "request_id") {
		t.Errorf("request_id non masqué: %q", out)
	}
	if strings.Contains(out, "source.function") {
		t.Errorf("source.function non masqué: %q", out)
	}
	if !strings.Contains(out, "keep=visible") {
		t.Errorf("attr visible absent: %q", out)
	}
}

// TestConsoleHandler_SkipAttrsCustom : override de la liste skip via options.
func TestConsoleHandler_SkipAttrsCustom(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{
		Level:     slog.LevelInfo,
		SkipAttrs: []string{"secret"},
	})
	logger.Info("test",
		"event_id", "should-show-now",
		"secret", "should-hide",
	)

	out := buf.String()
	if !strings.Contains(out, "event_id=should-show-now") {
		t.Errorf("event_id devrait apparaître (skip custom n'inclut pas event_id): %q", out)
	}
	if strings.Contains(out, "secret") {
		t.Errorf("secret non masqué: %q", out)
	}
}

// TestConsoleHandler_MaxWidthTruncation : ligne dépassant MaxWidth tronquée à
// MaxWidth runes avec suffixe `…`.
func TestConsoleHandler_MaxWidthTruncation(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{
		Level:    slog.LevelInfo,
		MaxWidth: 50,
	})
	longMsg := strings.Repeat("X", 100)
	logger.Info(longMsg)

	line := strings.TrimRight(buf.String(), "\n")
	// Comptage runes (pas bytes) pour respecter UTF-8.
	if lineWidth(line) != 50 {
		t.Errorf("ligne tronquée à %d runes, want 50: %q", lineWidth(line), line)
	}
	if !strings.HasSuffix(line, "…") {
		t.Errorf("suffixe ellipsis absent: %q", line)
	}
}

// TestConsoleHandler_MaxWidthZero_NoTruncation : MaxWidth=0 désactive le
// tronquage.
func TestConsoleHandler_MaxWidthZero_NoTruncation(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{
		Level:    slog.LevelInfo,
		MaxWidth: 0,
	})
	longMsg := strings.Repeat("X", 500)
	logger.Info(longMsg)

	line := strings.TrimRight(buf.String(), "\n")
	if !strings.Contains(line, longMsg) {
		t.Errorf("MaxWidth=0 doit préserver intégralement: line len=%d", len(line))
	}
	if strings.Contains(line, "…") {
		t.Errorf("MaxWidth=0 ne devrait pas tronquer: %q", line)
	}
}

// TestConsoleHandler_QuotedValueWithSpace : valeur d'attribut contenant un
// espace doit être quotée pour préserver le parsing key=val.
func TestConsoleHandler_QuotedValueWithSpace(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})
	logger.Info("test", "path", "C:\\Program Files\\app")

	out := buf.String()
	if !strings.Contains(out, `path="C:\Program Files\app"`) {
		t.Errorf("valeur avec espace non quotée: %q", out)
	}
}

// TestConsoleHandler_QuotedValueWithEquals : valeur contenant `=` quotée.
func TestConsoleHandler_QuotedValueWithEquals(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})
	logger.Info("test", "url", "host=foo")

	out := buf.String()
	if !strings.Contains(out, `url="host=foo"`) {
		t.Errorf("valeur avec = non quotée: %q", out)
	}
}

// TestConsoleHandler_NumericTypes : int/float/bool/duration formatés sans
// quoting (formats compacts natifs).
func TestConsoleHandler_NumericTypes(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})
	logger.Info("test",
		"count", 42,
		"ratio", 0.75,
		"ok", true,
		"dur", 250*time.Millisecond,
	)

	out := buf.String()
	wants := []string{"count=42", "ratio=0.75", "ok=true", "dur=250ms"}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("attr %q absent ou mal formaté: %q", w, out)
		}
	}
}

// TestConsoleHandler_WithAttrs : attrs accumulés via With propagés à chaque log.
func TestConsoleHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})
	scoped := logger.With("module", "sync", "gamertag", "Madina97294")
	scoped.Info("first")
	scoped.Info("second")

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("attendu 2 lignes, got %d: %q", len(lines), out)
	}
	for i, line := range lines {
		if !strings.Contains(line, "module=sync") {
			t.Errorf("ligne %d sans module=sync: %q", i, line)
		}
		if !strings.Contains(line, "gamertag=Madina97294") {
			t.Errorf("ligne %d sans gamertag: %q", i, line)
		}
	}
}

// TestConsoleHandler_WithAttrs_SkipApplies : attrs skip aussi via WithAttrs.
func TestConsoleHandler_WithAttrs_SkipApplies(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})
	logger.With("event_id", "x:y", "keep", "visible").Info("test")

	out := buf.String()
	if strings.Contains(out, "event_id") {
		t.Errorf("event_id via WithAttrs non skippé: %q", out)
	}
	if !strings.Contains(out, "keep=visible") {
		t.Errorf("keep via WithAttrs absent: %q", out)
	}
}

// TestConsoleHandler_ColorANSI : avec Color=true, le label est encadré de
// codes ANSI.
func TestConsoleHandler_ColorANSI(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{
		Level: slog.LevelInfo,
		Color: true,
	})
	logger.Error("oops")

	out := buf.String()
	if !strings.Contains(out, "\x1b[31m[ERROR]\x1b[0m") {
		t.Errorf("code ANSI rouge absent pour ERROR: %q", out)
	}
}

// TestConsoleHandler_ColorOff : sans Color, pas de codes ANSI.
func TestConsoleHandler_ColorOff(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{
		Level: slog.LevelInfo,
		Color: false,
	})
	logger.Error("oops")

	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("codes ANSI présents alors que Color=false: %q", buf.String())
	}
}

// TestConsoleHandler_EmitsNewline : chaque record produit exactement 1 ligne.
func TestConsoleHandler_EmitsNewline(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})
	logger.Info("a")
	logger.Info("b")
	logger.Info("c")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Errorf("attendu 3 lignes, got %d: %q", len(lines), buf.String())
	}
}

// TestConsoleHandler_TruncateRespectUTF8 : tronquage compte les runes, pas
// les bytes (caractères accentués / unicode préservés correctement).
func TestConsoleHandler_TruncateRespectUTF8(t *testing.T) {
	// `é` = 2 bytes UTF-8, 1 rune. Si on tronquait par bytes, on couperait
	// un rune au milieu et obtiendrait un UTF-8 invalide.
	msg := strings.Repeat("é", 200) // 200 runes, 400 bytes
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{
		Level:    slog.LevelInfo,
		MaxWidth: 80,
	})
	logger.Info(msg)

	line := strings.TrimRight(buf.String(), "\n")
	if lineWidth(line) != 80 {
		t.Errorf("ligne tronquée à %d runes, want 80", lineWidth(line))
	}
	// L'ellipsis doit être bien encodée (3 bytes).
	if !strings.HasSuffix(line, "…") {
		t.Errorf("suffixe ellipsis manquant: %q", line)
	}
}

// TestConsoleHandler_NewRecord_NoSource : avec un slog.Record bricolé (PC=0,
// AddSource false côté ConsoleHandler), le format reste stable.
func TestConsoleHandler_NewRecord_NoSource(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})
	rec := slog.NewRecord(time.Date(2026, 5, 20, 14, 30, 8, 0, time.UTC), slog.LevelInfo, "boot ok", 0)
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	line := strings.TrimRight(buf.String(), "\n")
	if !strings.HasPrefix(line, "14:30:08 [INFO] ") {
		t.Errorf("préfixe inattendu: %q", line)
	}
	if !strings.HasSuffix(line, "boot ok") {
		t.Errorf("suffixe inattendu: %q", line)
	}
}

// TestConsoleHandler_Enabled : Enabled respecte le Level configuré.
func TestConsoleHandler_Enabled(t *testing.T) {
	h := NewConsoleHandler(&bytes.Buffer{}, ConsoleHandlerOptions{Level: slog.LevelWarn})
	ctx := context.Background()
	if h.Enabled(ctx, slog.LevelDebug) {
		t.Error("Debug enabled alors que Level=Warn")
	}
	if h.Enabled(ctx, slog.LevelInfo) {
		t.Error("Info enabled alors que Level=Warn")
	}
	if !h.Enabled(ctx, slog.LevelWarn) {
		t.Error("Warn non enabled alors que Level=Warn")
	}
	if !h.Enabled(ctx, slog.LevelError) {
		t.Error("Error non enabled")
	}
}

// TestConsoleHandler_ConcurrentWrites : deux goroutines logguant en parallèle
// ne corrompent pas la sortie (lignes entières, pas d'interleaving rune-level).
func TestConsoleHandler_ConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	logger := newConsoleHandler(&buf, ConsoleHandlerOptions{Level: slog.LevelInfo})

	done := make(chan struct{}, 2)
	for range 2 {
		go func() {
			for range 50 {
				logger.Info("msg", "k", "v")
			}
			done <- struct{}{}
		}()
	}
	<-done
	<-done

	// Chaque ligne doit commencer par HH:MM:SS et finir par v.
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 100 {
		t.Errorf("attendu 100 lignes, got %d", len(lines))
	}
	for i, line := range lines {
		if len(line) < 9 || line[2] != ':' || line[5] != ':' {
			t.Errorf("ligne %d corrompue (concurrent write): %q", i, line)
			break
		}
	}
}

// TestParseConsoleFormat : la combinatoire LEVELUP_LOG_FORMAT × LEVELUP_LOG_JSON
// résout au bon format avec la priorité documentée.
func TestParseConsoleFormat(t *testing.T) {
	cases := []struct {
		name       string
		format     string
		jsonLegacy string
		want       string
	}{
		{"default", "", "", "compact"},
		{"explicit_compact", "compact", "", "compact"},
		{"explicit_text", "text", "", "text"},
		{"explicit_json", "json", "", "json"},
		{"legacy_json", "", "true", "json"},
		{"format_wins_over_legacy", "compact", "true", "compact"},
		{"unknown_falls_back", "weird", "", "compact"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("LEVELUP_LOG_FORMAT", c.format)
			t.Setenv("LEVELUP_LOG_JSON", c.jsonLegacy)
			got := parseConsoleFormat()
			if got != c.want {
				t.Errorf("parseConsoleFormat() = %q, want %q", got, c.want)
			}
		})
	}
}

// TestParseIntEnv : valeurs valides, défaut, négatif → 0.
func TestParseIntEnv(t *testing.T) {
	t.Setenv("LEVELUP_TEST_INT", "")
	if got := parseIntEnv("LEVELUP_TEST_INT", 200); got != 200 {
		t.Errorf("default = %d, want 200", got)
	}
	t.Setenv("LEVELUP_TEST_INT", "300")
	if got := parseIntEnv("LEVELUP_TEST_INT", 200); got != 300 {
		t.Errorf("parsed = %d, want 300", got)
	}
	t.Setenv("LEVELUP_TEST_INT", "-1")
	if got := parseIntEnv("LEVELUP_TEST_INT", 200); got != 0 {
		t.Errorf("négatif = %d, want 0 (clampé)", got)
	}
	t.Setenv("LEVELUP_TEST_INT", "abc")
	if got := parseIntEnv("LEVELUP_TEST_INT", 200); got != 200 {
		t.Errorf("invalide = %d, want 200 (fallback défaut)", got)
	}
}
