//go:build integration

package replayartifacts

// padtiers_integration_test.go — LA PORTE DE CAPABILITY DES NIVEAUX D'ARMES GOUVERNE REELLEMENT
// L'ECRITURE, et la projection au fil de l'eau ECRIT.
//
// # POURQUOI CE FICHIER, ALORS QU UN GREP GARDE DEJA LA PORTE
//
// Constat de revue (2026-09-14) : le garde-rail de `padtiers_test.go` est un GREP sur le source.
// Il prouve que l appel existe ; il ne prouve rien de ce qu il fait. Une mutation « porte
// toujours fermee » (`return false, false`) laisse ce grep vert, toute la suite verte, tags
// compris — et la table ne se remplit plus jamais.
//
// Ces tests-ci passent par `persisterNiveauxDArmes` avec une VRAIE base migree et un VRAI
// artefact, et comptent les lignes ecrites. Ils rougissent sur la mutation.

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// artefactAvecSocles pose un artefact PROJETABLE : schema courant, deux humains au roster, deux
// socles et deux prises nommees.
func artefactAvecSocles(t *testing.T, repoRoot, matchID string) {
	t.Helper()
	a, b := "a1", "b1"
	doc := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		MatchID:       matchID,
		Roster: []replay.RosterEntry{
			{XUID: "a1", Name: "Alpha"}, {XUID: "b1", Name: "Bravo"},
		},
		WeaponPads: []replay.WeaponPad{
			{X: 0, Y: 0, Z: 0, Weapon: "0xB619D84A"},
			{X: 9, Y: 0, Z: 0, Weapon: "0x9D6AAED2"},
		},
		PadPickups: []replay.PadPickup{
			{Pad: 0, TLow: 5, THigh: 15, XUID: &a},
			{Pad: 1, TLow: 20, THigh: 30, XUID: &b},
		},
	}
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// lignesDeNiveauxEnBase compte les lignes servies par la VUE `_latest` (ADR 0026).
func lignesDeNiveauxEnBase(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_pad_pickups_by_tier_latest`).Scan(&n); err != nil {
		t.Fatalf("comptage des niveaux: %v", err)
	}
	return n
}

// monterLeLot prepare base + racine + artefact et rend les Deps du fil de l eau.
func monterLeLot(t *testing.T, matchID string) (*sql.DB, Deps, []artefactLu) {
	t.Helper()
	db := baseRegistre(t)
	repoRoot := racineAvecConfigDuTitre(t, titlePkg.DefaultSlug)
	inscrireAuRegistre(t, db, matchID, temoinT0(), 0)
	artefactAvecSocles(t, repoRoot, matchID)

	d := Deps{
		RepoRoot: repoRoot, TitleSlug: titlePkg.DefaultSlug, Gamertag: "testeur",
		WithRead:      func(_ context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil },
	}
	return db, d, lireArtefacts(context.Background(), d, []ArtefactRange{{
		MatchID: matchID,
		Path:    titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, matchID),
	}})
}

// TestPersisterNiveauxDArmes_EcritQuandLaCapabiliteEstDeclaree — le cas NOMINAL, et la mutation
// « porte toujours fermee » le fait rougir.
func TestPersisterNiveauxDArmes_EcritQuandLaCapabiliteEstDeclaree(t *testing.T) {
	db, d, lus := monterLeLot(t, "niveaux1")
	if len(lus) != 1 {
		t.Fatalf("%d artefact(s) lu(s), attendu 1", len(lus))
	}
	persisterNiveauxDArmes(context.Background(), d, &bilanDerivations{}, lus)

	if n := lignesDeNiveauxEnBase(t, db); n == 0 {
		t.Fatal("AUCUNE ligne ecrite alors que le titre declare film.weapon_tiers : la porte " +
			"de capability ou l ecriture est debranchee, et la table ne se remplirait jamais")
	}
	// Les deux joueurs ont une ligne, et le niveau vient de la CARTE : la carte du temoin n est
	// pas dans la reference copiee, donc les deux prises sont NON CLASSEES — c est une mesure,
	// pas une panne, et c est ce que `pads_confirmed = 0` dit.
	var confirmes, total int
	if err := db.QueryRow(
		`SELECT pads_confirmed, pads_total FROM match_pad_pickups_by_tier_latest LIMIT 1`).
		Scan(&confirmes, &total); err != nil {
		t.Fatalf("lecture des valeurs de match: %v", err)
	}
	if total != 2 {
		t.Errorf("pads_total = %d, attendu 2 (les deux socles du film)", total)
	}
}

// TestPersisterNiveauxDArmes_NEcritRienSansLaCapabilite — la porte gouverne, et c est la SEULE
// chose qui change entre les deux tests.
func TestPersisterNiveauxDArmes_NEcritRienSansLaCapabilite(t *testing.T) {
	db, d, lus := monterLeLot(t, "niveaux2")
	retirerCapabiliteNiveaux(t, d.RepoRoot)

	persisterNiveauxDArmes(context.Background(), d, &bilanDerivations{}, lus)

	if n := lignesDeNiveauxEnBase(t, db); n != 0 {
		t.Errorf("%d ligne(s) ecrite(s) sur un titre qui NE DECLARE PAS film.weapon_tiers : la "+
			"porte de capability ne gouverne rien", n)
	}
}

// TestPersisterNiveauxDArmes_IdentiteIllisibleNEcritRien — un lot dont le registre ne rend pas
// l identite n est PAS projete : un `pair_name` vide ecrirait un niveau « base » sur des Fiesta.
func TestPersisterNiveauxDArmes_IdentiteIllisibleNEcritRien(t *testing.T) {
	db, d, lus := monterLeLot(t, "niveaux3")
	// Le match disparait du registre : son identite devient illisible.
	if _, err := db.Exec(`DELETE FROM match_registry WHERE match_id = 'niveaux3'`); err != nil {
		t.Fatalf("suppression au registre: %v", err)
	}
	persisterNiveauxDArmes(context.Background(), d, &bilanDerivations{}, lus)

	if n := lignesDeNiveauxEnBase(t, db); n != 0 {
		t.Errorf("%d ligne(s) ecrite(s) sans identite de match : la projection a suppose un mode "+
			"regulier, ce qui ecrit un niveau de base sur des equipements tires au sort", n)
	}
}

// TestPersisterNiveauxDArmes_ReferenceAbsenteNePrivePasLesAutres — cette famille s abstient
// SEULE : elle ne pose aucun echec au bilan, donc les autres gardent leur marque.
func TestPersisterNiveauxDArmes_ReferenceAbsenteNePrivePasLesAutres(t *testing.T) {
	_, d, lus := monterLeLot(t, "niveaux4")
	ref := filepath.Join(d.RepoRoot, "data", "titles", titlePkg.DefaultSlug, "reference",
		"map_weapon_pads.json")
	if err := os.Remove(ref); err != nil && !os.IsNotExist(err) {
		t.Fatalf("retrait de la reference: %v", err)
	}
	b := &bilanDerivations{}
	persisterNiveauxDArmes(context.Background(), d, b, lus)

	if b.aEchoue("niveaux4") {
		t.Error("la famille a pose un echec au bilan alors que SA reference manque : les " +
			"quatre autres familles, deja ecrites, perdraient leur marque de derivation")

	}
}

// temoinT0 : un instant de reference stable pour le registre des tests.
func temoinT0() time.Time { return time.Now().UTC().Add(-2 * time.Hour) }

// retirerCapabiliteNiveaux retire `film.weapon_tiers` du capabilities.toml de la racine de
// test. C est la SEULE difference entre les deux temoins de la porte.
func retirerCapabiliteNiveaux(t *testing.T, repoRoot string) {
	t.Helper()
	p := filepath.Join(repoRoot, "config", "titles", titlePkg.DefaultSlug, "mappings", "capabilities.toml")
	blob, err := os.ReadFile(p) //nolint:gosec // racine de test
	if err != nil {
		t.Fatalf("lecture capabilities: %v", err)
	}
	var out []string
	for _, l := range strings.Split(string(blob), "\n") {
		if strings.Contains(l, string(games.CapFilmWeaponTiers)) {
			continue
		}
		out = append(out, l)
	}
	if err := os.WriteFile(p, []byte(strings.Join(out, "\n")), 0o600); err != nil {
		t.Fatalf("ecriture capabilities: %v", err)
	}
}
