package testfixtures

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WALEntry est une vue brute (json.RawMessage) sur un batch WAL persiste.
//
// On ne decode pas vers persist.MatchBatch ici pour eviter un cycle d'import
// (testfixtures importe par tous les packages, y compris persist). Les tests
// downstream font le json.Unmarshal vers leur type concret.
type WALEntry struct {
	BatchID string
	Path    string
	RawJSON []byte
}

// WALAvailable retourne true si data/wal/ contient au moins 1 batch.
func WALAvailable() bool {
	entries, err := os.ReadDir(WALDir())
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			return true
		}
	}
	return false
}

// LoadAllWAL retourne tous les batches WAL presents dans data/wal/.
//
// Si data/wal/ est vide ou absent, appelle t.Skip avec instructions.
// Ordre : trie par nom de fichier (= batch_id, pseudo-ordre temporel).
func LoadAllWAL(t *testing.T) []WALEntry {
	t.Helper()
	dir := WALDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("WAL dir absent (%s): %v — generer via un cycle de sync", dir, err)
	}
	out := make([]WALEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("LoadAllWAL: read %s: %v", path, err)
		}
		out = append(out, WALEntry{
			BatchID: strings.TrimSuffix(e.Name(), ".json"),
			Path:    path,
			RawJSON: raw,
		})
	}
	if len(out) == 0 {
		t.Skipf("WAL dir vide (%s) — generer via un cycle de sync", dir)
	}
	return out
}

// LoadWALByPlayer charge tous les batches WAL d'un joueur donne (filtre sur
// le champ "player" du JSON). Skip si aucun match.
func LoadWALByPlayer(t *testing.T, gamertag string) []WALEntry {
	t.Helper()
	all := LoadAllWAL(t)
	out := make([]WALEntry, 0, len(all))
	for _, entry := range all {
		var probe struct {
			Player string `json:"player"`
		}
		if err := json.Unmarshal(entry.RawJSON, &probe); err != nil {
			continue
		}
		if probe.Player == gamertag {
			out = append(out, entry)
		}
	}
	if len(out) == 0 {
		t.Skipf("aucun batch WAL pour %s dans %s", gamertag, WALDir())
	}
	return out
}

// LoadWALByMatchID charge le 1er batch WAL pour un match_id donne. Skip si absent.
//
// Permet de cibler un match specifique pour reproduire un bug particulier.
func LoadWALByMatchID(t *testing.T, matchID string) WALEntry {
	t.Helper()
	all := LoadAllWAL(t)
	for _, entry := range all {
		var probe struct {
			Shared struct {
				Match struct {
					MatchID string `json:"MatchID"`
				} `json:"match"`
			} `json:"shared"`
		}
		if err := json.Unmarshal(entry.RawJSON, &probe); err != nil {
			continue
		}
		if probe.Shared.Match.MatchID == matchID {
			return entry
		}
	}
	t.Skipf("aucun batch WAL pour match_id=%s", matchID)
	return WALEntry{}
}

// WALCountByPlayer retourne le compteur de batches WAL par gamertag, utile
// pour les tests qui veulent stats globales sans charger tout en memoire.
func WALCountByPlayer(t *testing.T) map[string]int {
	t.Helper()
	all := LoadAllWAL(t)
	counts := make(map[string]int)
	for _, entry := range all {
		var probe struct {
			Player string `json:"player"`
		}
		if err := json.Unmarshal(entry.RawJSON, &probe); err != nil {
			continue
		}
		counts[probe.Player]++
	}
	return counts
}

// AssertWALStructure verifie qu'un batch WAL a la structure minimale attendue
// (batch_id, title_slug, player, xuid, shared.match). Utilise par les tests
// de round-trip pour valider l'invariant avant unmarshal vers le type concret.
func AssertWALStructure(t *testing.T, entry WALEntry) {
	t.Helper()
	var probe struct {
		BatchID   string `json:"batch_id"`
		TitleSlug string `json:"title_slug"`
		Player    string `json:"player"`
		XUID      string `json:"xuid"`
		Shared    struct {
			Match map[string]any `json:"match"`
		} `json:"shared"`
	}
	if err := json.Unmarshal(entry.RawJSON, &probe); err != nil {
		t.Fatalf("AssertWALStructure(%s): unmarshal: %v", entry.BatchID, err)
	}
	if probe.BatchID == "" {
		t.Errorf("AssertWALStructure(%s): batch_id vide", entry.BatchID)
	}
	if probe.TitleSlug == "" {
		t.Errorf("AssertWALStructure(%s): title_slug vide", entry.BatchID)
	}
	if probe.Player == "" {
		t.Errorf("AssertWALStructure(%s): player vide", entry.BatchID)
	}
	if probe.Shared.Match == nil {
		t.Errorf("AssertWALStructure(%s): shared.match nil", entry.BatchID)
	}
}
