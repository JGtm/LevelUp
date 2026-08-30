//go:build integration

// Package killcollector — positions_integration_test.go : LE GATE DE LA CAPTURE, sur un film REEL.
//
// Meme fixture, meme skip, meme raison que collector_test.go (TestKillSourceCollecteFilmReelEt
// RelitParLaVue) : ⚠ LES FILMS NE SONT PAS VERSIONNES (107 Mo). Sans KILLSOURCE_FIXTURES, ce
// test se SKIPPE — la commande exacte est dans collector_test.go.
//
// CE QUE CE FICHIER NE PEUT PAS SAVOIR, ET COMMENT IL S EN ACCOMMODE : cette session n a acces
// a aucune donnee de production (pas de data/ dans ce worktree — CLAUDE.md du chantier), donc
// elle ignore quelle carte le film de reference (9b191a7f) a ete joue sur, et si cette carte est
// meme au catalogue de bornes (elle peut etre une carte Forge, hors des 79 cartes natives). Le
// test offre donc TOUS les noms du catalogue REEL (config versionne, pas data/) comme candidats
// — Lookup fait une correspondance EXACTE par cle normalisee, donc une carte absente reste
// absente meme avec la liste complete en candidats — et traite « 0 ligne de position » comme un
// SKIP documente plutot qu un echec : ce cas est aussi legitime qu un match sans film.
package killcollector

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/haloclient"
)

// realMapQuantCatalog charge le catalogue de bornes REEL du depot — DONNEE DE REFERENCE
// VERSIONNEE (data/titles/halo_infinite/reference/map_quant_bounds.json, ~22 Ko, commitee), pas
// une sortie de sync/backfill : elle est disponible meme dans un worktree sans data/ de travail.
// Chemin resolu par PathResolver (CLAUDE.md : jamais de filepath.Join(..., "data", ...) a la main).
func realMapQuantCatalog(t *testing.T) *filmdec.MapQuantCatalog {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	pr := titlePkg.NewPathResolver(repoRoot)
	cat, err := filmdec.LoadMapQuantCatalog(pr.MapQuantBoundsPath(titlePkg.DefaultSlug))
	if err != nil {
		t.Skipf("catalogue de bornes indisponible (%v) — positions non testables sans lui", err)
	}
	return cat
}

// allCatalogNames : TOUS les noms du catalogue REEL, comme candidats (cf. en-tete du fichier).
func allCatalogNames(cat *filmdec.MapQuantCatalog) []string {
	names := make([]string, 0, len(cat.Maps))
	for name := range cat.Maps {
		names = append(names, name)
	}
	return names
}

// staticMapNames : port.ReplayMapNameRepo qui rend TOUJOURS la meme liste, sans base — le test
// n a pas besoin de savoir QUEL match on lui demande, il n a qu UN film de fixture.
type staticMapNames struct{ names []string }

func (s staticMapNames) MapKeysForMatch(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{Names: s.names}, nil
}

// TestKillSourcePositionsFilmReelEtRelitParLaVue — meme film et meme gate que
// TestKillSourceCollecteFilmReelEtRelitParLaVue (collector_test.go), positions en plus : le pont
// disque, les quatre lectures hors ligne et le pont slot->xuid s emboitent sur du binaire REEL,
// pas seulement sur des chunks synthetiques.
func TestKillSourcePositionsFilmReelEtRelitParLaVue(t *testing.T) {
	const film = "9b191a7f"
	chunks := chargerFilmDeFixture(t, film)
	cat := realMapQuantCatalog(t)

	db := openSharedTestDB(t)
	client := &fakeFilmClient{chunks: map[string][]haloclient.FilmChunk{film: chunks}}
	caps := games.CapabilityMap{
		games.CapFilmKillSource:    games.CapSupported,
		games.CapFilmKillPositions: games.CapSupported,
	}
	col := NewKillSourceCollector(client, fakeRoster{}, sharedWriter(db), caps, 0).
		WithPositionCapture(staticMapNames{names: allCatalogNames(cat)}, cat)

	if _, _, err := col.CollectMatch(context.Background(), film); err != nil {
		t.Fatalf("CollectMatch: %v", err)
	}

	var morts, positions int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = ?)`,
		film, film).Scan(&morts, &positions); err != nil {
		t.Fatalf("select: %v", err)
	}
	t.Logf("film %s : %d morts, %d lignes de position", film, morts, positions)
	if positions == 0 {
		t.Skip("0 ligne de position — la carte de ce film n'est probablement pas au catalogue " +
			"de bornes (carte Forge, ou hors des 79 cartes natives) : cas normal, pas une regression")
	}
	if positions > morts {
		t.Errorf("%d lignes de position pour %d morts : ne peut jamais depasser (au plus une "+
			"ligne par mort resolue)", positions, morts)
	}

	var sansKiller int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest
		WHERE match_id = ? AND (killer_xuid IS NULL OR killer_xuid = '')`, film).Scan(&sansKiller); err != nil {
		t.Fatalf("select killer_xuid: %v", err)
	}
	if sansKiller != 0 {
		t.Errorf("%d ligne(s) sans killer_xuid — la cle fonctionnelle doit toujours etre renseignee", sansKiller)
	}

	var auMoinsUnePosition int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = ?
		AND (killer_x IS NOT NULL OR victim_x IS NOT NULL)`, film).Scan(&auMoinsUnePosition); err != nil {
		t.Fatalf("select coordonnees: %v", err)
	}
	if auMoinsUnePosition != positions {
		t.Errorf("%d ligne(s) sans AUCUNE coordonnee (ni tueur ni victime) — "+
			"BuildKillPositions ne devait en ecrire aucune", positions-auMoinsUnePosition)
	}

	// Idempotence append-only (ADR 0026) : une 2e passe ne double pas la vue, la table garde les
	// deux (meme propriete que collector_test.go pour match_kill_events).
	if _, _, err := col.CollectMatch(context.Background(), film); err != nil {
		t.Fatalf("2e passe: %v", err)
	}
	var positions2, table2 int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM kill_positions WHERE match_id = ?)`, film, film).
		Scan(&positions2, &table2); err != nil {
		t.Fatalf("select 2e passe: %v", err)
	}
	if positions2 != positions {
		t.Errorf("apres 2 passes, la vue sert %d lignes au lieu de %d — le dedoublonnage par cle "+
			"(match_id, killer_xuid, time_ms) ne tient plus", positions2, positions)
	}
	if table2 <= positions {
		t.Errorf("table = %d lignes apres 2 passes, attendu > %d (append-only : la 1ere passe "+
			"reste physiquement presente)", table2, positions)
	}
}
