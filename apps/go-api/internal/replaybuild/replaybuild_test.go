package replaybuild

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/testutil"
)

// TestArtifactUpToDate — la clé de reprise du backfill : seule la version de schéma
// COURANTE vaut « à jour ». Absent, illisible ou antérieur = à re-cuire.
func TestArtifactUpToDate(t *testing.T) {
	dir := t.TempDir()
	cas := map[string]struct {
		contenu string
		attendu bool
	}{
		"version courante":   {fmt.Sprintf(`{"schemaVersion":%d,"matchId":"m"}`, replay.SchemaVersion), true},
		"version anterieure": {`{"schemaVersion":2,"matchId":"m"}`, false},
		"json illisible":     {`{pas du json`, false},
		"sans version":       {`{"matchId":"m"}`, false},
	}
	for nom, c := range cas {
		p := filepath.Join(dir, nom+".json")
		if err := os.WriteFile(p, []byte(c.contenu), 0o644); err != nil {
			t.Fatalf("écriture fixture %s: %v", nom, err)
		}
		if got := ArtifactUpToDate(p); got != c.attendu {
			t.Errorf("%s : ArtifactUpToDate = %v, attendu %v", nom, got, c.attendu)
		}
	}
	if ArtifactUpToDate(filepath.Join(dir, "absent.json")) {
		t.Error("artefact absent : attendu « à re-cuire », obtenu « à jour »")
	}
}

// TestArtifactHasMatchFacts — le prédicat qui distingue un artefact COMPLET d'un artefact
// APPAUVRI, là où la version de schéma ne les distingue pas.
//
// Le signal est `scoreTimeline.players`, et il est adossé au contrat du document lui-même
// (`document_score.go:147-148` : « Players porte les joueurs dont le slot d'entité a été apparié
// à une ligne de match. Vide quand l'appelant n'a pas fourni les lignes »). Mesuré sur deux
// témoins le 2026-08-24 : 8 avec faits, 0 sans, sur 7344d24f comme sur 530820e5.
func TestArtifactHasMatchFacts(t *testing.T) {
	dir := t.TempDir()
	cas := map[string]struct {
		contenu string
		attendu bool
	}{
		"avec joueurs de score":  {`{"schemaVersion":18,"scoreTimeline":{"players":[{"xuid":"25332748"}]}}`, true},
		"joueurs vides":          {`{"schemaVersion":18,"scoreTimeline":{"players":[]}}`, false},
		"courbe sans joueurs":    {`{"schemaVersion":18,"scoreTimeline":{"teams":[{"team":0}]}}`, false},
		"aucune courbe de score": {`{"schemaVersion":18,"matchId":"m"}`, false},
		"json illisible":         {`{pas du json`, false},
	}
	for nom, c := range cas {
		p := filepath.Join(dir, nom+".json")
		if err := os.WriteFile(p, []byte(c.contenu), 0o644); err != nil {
			t.Fatalf("écriture fixture %s: %v", nom, err)
		}
		if got := ArtifactHasMatchFacts(p); got != c.attendu {
			t.Errorf("%s : ArtifactHasMatchFacts = %v, attendu %v", nom, got, c.attendu)
		}
	}
	if ArtifactHasMatchFacts(filepath.Join(dir, "absent.json")) {
		t.Error("artefact absent : attendu « sans faits », obtenu « avec faits »")
	}
}

// TestResolveMapEntry_SurLeCatalogueLivre — le builder résout les candidats DANS L'ORDRE
// sur le catalogue de bornes VERSIONNÉ, et rend l'échec voulu quand aucun ne résout.
// Oracle réel : Cliffhanger -> module ridgeline (la référence du POC).
//
// AUCUN SKIP : la racine vient de testutil.RepoRoot() (déduite de l'arbre source), et tout
// ce que NewBuilder lit est versionné — map_quant_bounds.json, weapon_names.toml,
// replay_labels.toml (git ls-files, 2026-08-19). Leur absence est une installation cassée,
// pas une dispense.
func TestResolveMapEntry_SurLeCatalogueLivre(t *testing.T) {
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	b, err := NewBuilder(repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	entry, err := b.ResolveMapEntry([]string{"", "Cliffhanger"})
	if err != nil {
		t.Fatalf("ResolveMapEntry(Cliffhanger): %v", err)
	}
	if entry.Module != "ridgeline" {
		t.Errorf("module de Cliffhanger = %q, attendu ridgeline", entry.Module)
	}
	if _, err := b.ResolveMapEntry([]string{"CarteInexistante-v75"}); !errors.Is(err, ErrMapNotInCatalog) {
		t.Errorf("carte inconnue : attendu ErrMapNotInCatalog, obtenu %v", err)
	}
	if _, err := b.ResolveMapEntry(nil); !errors.Is(err, ErrMapNotInCatalog) {
		t.Errorf("aucun candidat : attendu ErrMapNotInCatalog, obtenu %v", err)
	}
}
