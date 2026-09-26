package replaybuild

// zones_table_absente_test.go — UNE TABLE D OBJECTIFS ABSENTE EST LE CAS NOMINAL, JOURNALISE EN
// DEBUG (lot J2.11, constat CONV-1, 2026-09-26).
//
// `mappings.LoadObjectiveRolesFromFile` enveloppe l erreur de lecture (`read %s: %w`) :
// `os.IsNotExist`, qui ne deroule pas l enveloppe, ne la reconnaissait jamais, et un titre sans
// table tombait dans la branche « table illisible » — un WARN par Builder pour un cas nominal.

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"levelup/go-api/internal/domain/title"
)

func TestObjectiveRoles_TableAbsenteEstJournaliseeEnDebug(t *testing.T) {
	var buf bytes.Buffer
	ancien := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(ancien) })

	b := &Builder{repoRoot: t.TempDir(), titleSlug: title.DefaultSlug}
	if set := b.objectiveRoles(); set != nil {
		t.Fatalf("table absente : %+v, attendu nil", set)
	}
	var niveaux []string
	for _, ligne := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		var rec struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
		}
		if err := json.Unmarshal(ligne, &rec); err != nil {
			t.Fatalf("journal illisible %q : %v", ligne, err)
		}
		niveaux = append(niveaux, rec.Level+" "+rec.Msg)
	}
	if len(niveaux) != 1 || niveaux[0] !=
		"DEBUG replaybuild: titre sans table d'objectifs — rejeu sans etat de zone" {
		t.Fatalf("journal %q, attendu la seule ligne DEBUG « titre sans table d'objectifs »", niveaux)
	}
}
