package logging

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// countArchives compte les fichiers `{base}.N` présents dans dir.
func countArchives(t *testing.T, dir, base string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir %s: %v", dir, err)
	}
	n := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), base+".") {
			n++
		}
	}
	return n
}

// TestRotatingWriter_RotatesAtMaxSize : dépasser MaxSizeBytes crée `.1` et
// repart d'un fichier courant neuf.
func TestRotatingWriter_RotatesAtMaxSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sync.log")
	w, err := newRotatingWriter(path, RotationPolicy{MaxSizeBytes: 100, MaxBackups: 3})
	if err != nil {
		t.Fatalf("newRotatingWriter: %v", err)
	}
	defer w.Close()

	line := bytes.Repeat([]byte("a"), 40)
	for i := 0; i < 3; i++ { // 120 octets > 100 → rotation au 3e write
		if _, err := w.Write(line); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	archive := filepath.Join(dir, "sync.log.1")
	info, err := os.Stat(archive)
	if err != nil {
		t.Fatalf("archive sync.log.1 absente: %v", err)
	}
	if info.Size() != 80 {
		t.Errorf("archive = %d octets, attendu 80 (les 2 premiers writes)", info.Size())
	}
	cur, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat fichier courant: %v", err)
	}
	if cur.Size() != 40 {
		t.Errorf("fichier courant = %d octets, attendu 40 (write post-rotation)", cur.Size())
	}
}

// TestRotatingWriter_KeepsAtMostMaxBackups : la rétention est stricte — au-delà
// de MaxBackups archives, la plus ancienne est supprimée (borne disque).
func TestRotatingWriter_KeepsAtMostMaxBackups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provider.log")
	const maxBackups = 2
	w, err := newRotatingWriter(path, RotationPolicy{MaxSizeBytes: 50, MaxBackups: maxBackups})
	if err != nil {
		t.Fatalf("newRotatingWriter: %v", err)
	}
	defer w.Close()

	// 10 writes de 40 octets → 9 rotations, très au-delà de la rétention.
	for i := 0; i < 10; i++ {
		if _, err := w.Write(bytes.Repeat([]byte("b"), 40)); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	if got := countArchives(t, dir, "provider.log"); got != maxBackups {
		t.Errorf("%d archives conservées, attendu %d", got, maxBackups)
	}
	if _, err := os.Stat(filepath.Join(dir, "provider.log.3")); !os.IsNotExist(err) {
		t.Error("provider.log.3 ne doit jamais exister avec MaxBackups=2")
	}
}

// TestRotatingWriter_ContinuesWritingAfterRotation : l'écriture reste continue
// et le contenu reste lisible dans le fichier courant après plusieurs rotations.
func TestRotatingWriter_ContinuesWritingAfterRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.log")
	w, err := newRotatingWriter(path, RotationPolicy{MaxSizeBytes: 60, MaxBackups: 1})
	if err != nil {
		t.Fatalf("newRotatingWriter: %v", err)
	}
	defer w.Close()

	for i := 0; i < 20; i++ {
		if _, err := w.Write([]byte("ligne de log numero XX\n")); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}
	if _, err := w.Write([]byte("DERNIERE LIGNE\n")); err != nil {
		t.Fatalf("write final: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fichier courant: %v", err)
	}
	if !strings.Contains(string(data), "DERNIERE LIGNE") {
		t.Errorf("fichier courant sans la dernière ligne: %q", string(data))
	}
	// Aucun fichier (courant ou archive) ne dépasse le plafond + une ligne.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		info, statErr := e.Info()
		if statErr != nil {
			continue
		}
		if info.Size() > 60+int64(len("ligne de log numero XX\n")) {
			t.Errorf("%s = %d octets, dépasse le plafond de rotation", e.Name(), info.Size())
		}
	}
}

// TestRotatingWriter_DisabledWhenMaxSizeZero : MaxSizeBytes=0 conserve le
// comportement historique (append illimité, aucune archive).
func TestRotatingWriter_DisabledWhenMaxSizeZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "general.log")
	w, err := newRotatingWriter(path, RotationPolicy{MaxSizeBytes: 0, MaxBackups: 3})
	if err != nil {
		t.Fatalf("newRotatingWriter: %v", err)
	}
	defer w.Close()

	for i := 0; i < 50; i++ {
		if _, err := w.Write(bytes.Repeat([]byte("c"), 100)); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}
	if got := countArchives(t, dir, "general.log"); got != 0 {
		t.Errorf("%d archives créées alors que la rotation est désactivée", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != 5000 {
		t.Errorf("fichier = %d octets, attendu 5000 (aucune rotation)", info.Size())
	}
}

// TestRotatingWriter_ReopenKeepsExistingSize : à la ré-ouverture d'un fichier
// déjà volumineux (redémarrage du serveur), la taille de départ est celle du
// disque — sinon un process relancé écrirait 100 Mo de plus avant de roter.
func TestRotatingWriter_ReopenKeepsExistingSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "duckdb.log")
	if err := os.WriteFile(path, bytes.Repeat([]byte("d"), 90), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	w, err := newRotatingWriter(path, RotationPolicy{MaxSizeBytes: 100, MaxBackups: 1})
	if err != nil {
		t.Fatalf("newRotatingWriter: %v", err)
	}
	defer w.Close()

	if _, err := w.Write(bytes.Repeat([]byte("e"), 20)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "duckdb.log.1")); err != nil {
		t.Fatalf("rotation attendue dès le 1er write (90+20 > 100): %v", err)
	}
}

// TestRotatingWriter_OversizedRecordNeverLoops : un record plus gros que le
// plafond ne doit pas déclencher une rotation à chaque ligne (purge en boucle
// des archives). Il est écrit tel quel, puis la ligne suivante rote.
func TestRotatingWriter_OversizedRecordNeverLoops(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "handlers.log")
	w, err := newRotatingWriter(path, RotationPolicy{MaxSizeBytes: 50, MaxBackups: 2})
	if err != nil {
		t.Fatalf("newRotatingWriter: %v", err)
	}
	defer w.Close()

	big := bytes.Repeat([]byte("f"), 500)
	for i := 0; i < 3; i++ {
		if _, err := w.Write(big); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}
	// 3 writes → 2 rotations (le 1er write part sur fichier vide, jamais roté).
	if got := countArchives(t, dir, "handlers.log"); got != 2 {
		t.Errorf("%d archives, attendu 2", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != 500 {
		t.Errorf("fichier courant = %d octets, attendu 500 (record complet préservé)", info.Size())
	}
}

// TestMultiModuleHandler_RotatesCategoryFile : bout-en-bout — la rotation
// s'applique bien au writer des CATÉGORIES (pas seulement à un writer isolé).
func TestMultiModuleHandler_RotatesCategoryFile(t *testing.T) {
	dir := t.TempDir()
	console := slog.NewTextHandler(&bytes.Buffer{}, nil)
	mh, err := NewMultiModuleHandler(console, dir, slog.LevelInfo,
		RotationPolicy{MaxSizeBytes: 2048, MaxBackups: 2})
	if err != nil {
		t.Fatalf("NewMultiModuleHandler: %v", err)
	}
	defer mh.Close()

	logger := slog.New(mh).With("module", "sync")
	for i := 0; i < 200; i++ {
		logger.Info("message de remplissage pour forcer la rotation par taille",
			"i", i, "payload", strings.Repeat("x", 64))
	}

	if got := countArchives(t, dir, "sync.log"); got == 0 {
		t.Fatal("aucune archive sync.log.N — la rotation ne s'applique pas aux catégories")
	} else if got > 2 {
		t.Errorf("%d archives, MaxBackups=2 non respecté", got)
	}
	// L'écriture continue après rotation : le fichier courant reçoit les derniers logs.
	if content := readLog(t, dir, "sync"); !strings.Contains(content, "remplissage") {
		t.Errorf("fichier courant vide après rotation: %q", content)
	}
}

// TestDefaultRotationPolicy_Sizing verrouille le dimensionnement livré
// (100 Mo × 3 archives = ~400 Mo par catégorie au pire).
func TestDefaultRotationPolicy_Sizing(t *testing.T) {
	p := DefaultRotationPolicy()
	if p.MaxSizeBytes != 100*1024*1024 {
		t.Errorf("MaxSizeBytes = %d, attendu 100 Mo", p.MaxSizeBytes)
	}
	if p.MaxBackups != 3 {
		t.Errorf("MaxBackups = %d, attendu 3", p.MaxBackups)
	}
	if !p.enabled() {
		t.Error("la politique par défaut doit être active")
	}
}

// TestLoadConfig_RotationFromEnv : les bornes sont pilotables en ops sans
// redéploiement de code.
func TestLoadConfig_RotationFromEnv(t *testing.T) {
	t.Setenv("LEVELUP_LOGS_MAX_SIZE_MB", "5")
	t.Setenv("LEVELUP_LOGS_MAX_BACKUPS", "1")
	cfg := LoadConfig("")
	if cfg.Rotation.MaxSizeBytes != 5*1024*1024 {
		t.Errorf("MaxSizeBytes = %d, attendu 5 Mo", cfg.Rotation.MaxSizeBytes)
	}
	if cfg.Rotation.MaxBackups != 1 {
		t.Errorf("MaxBackups = %d, attendu 1", cfg.Rotation.MaxBackups)
	}

	t.Setenv("LEVELUP_LOGS_MAX_SIZE_MB", "0")
	if LoadConfig("").Rotation.enabled() {
		t.Error("0 Mo doit désactiver la rotation")
	}
}
