//go:build integration

// player_read_cache_degraded_test.go — chargements DÉGRADÉS du cache des lectures joueur,
// sur les VRAIS chargeurs (lot L9-go, 2026-09-23, revue adversariale B, P0 ; P2 de la revue
// D). Une panne de metadata ou une requête annulée pendant les traductions best-effort
// rendent des lignes incomplètes SANS erreur : elles ne doivent jamais entrer en cache
// (avant le correctif : servies 60 s à toutes les requêtes du joueur).
package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// panneMetadata simule une panne de lecture des traductions : la colonne `name` de
// asset_translations disparaît le temps de la panne (toute lecture qui la cite échoue,
// ce n'est pas une table absente), puis revient avec reparer.
func panneMetadata(t *testing.T, meta *DB) (reparer func()) {
	t.Helper()
	ctx := context.Background()
	if _, err := meta.Exec(ctx, `ALTER TABLE asset_translations RENAME COLUMN name TO name_en_panne`); err != nil {
		t.Fatalf("panne metadata : %v", err)
	}
	return func() {
		if _, err := meta.Exec(ctx, `ALTER TABLE asset_translations RENAME COLUMN name_en_panne TO name`); err != nil {
			t.Fatalf("réparation metadata : %v", err)
		}
	}
}

// seedTraductionsH5 met m1 dans l'état « voie Halo 5 » (noms NULL, ids remplis) et
// seede ses libellés : les noms des lignes de filtres viennent alors TOUS de metadata.
func seedTraductionsH5(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	nullifyRegistryNamesForH5(t, pdb, "map-h5", "playlist-h5", "variant-h5")
	seedFilterAssetTranslations(t, pdb.Metadata, [][4]string{
		{"map-h5", "map", "en-US", "Truth"},
		{"map-h5", "map", "fr-FR", "Vérité"},
		{"playlist-h5", "playlist", "en-US", "Team Arena"},
		{"playlist-h5", "playlist", "fr-FR", "Arène en équipe"},
		{"variant-h5", "game_variant", "en-US", "Slayer"},
		{"variant-h5", "game_variant", "fr-FR", "Assassin"},
	})
}

// lignesNonTraduites : la ligne de m1 n'a pas ses trois libellés FR.
func lignesNonTraduites(rows []domain.FilterMatchRow) bool {
	if len(rows) != 1 {
		return false
	}
	m := rows[0]
	return m.MapNameFR == nil || *m.MapNameFR != "Vérité" ||
		m.GameVariantNameFR == nil || *m.GameVariantNameFR != "Assassin" ||
		m.PlaylistName == nil || *m.PlaylistName != "Arène en équipe"
}

// filtresCachesIsoles : le VRAI FiltersRepo derrière un cache neuf (pas le cache
// process-wide : les tests restent indépendants).
func filtresCachesIsoles(pdb *PlayerDB) *CachedFiltersRepo {
	inner := NewFiltersRepo(pdb)
	return &CachedFiltersRepo{FiltersRepo: inner, rows: inner,
		cache: newPlayerReadCache("filter_rows_cache", cloneFilterRows), scope: scopeOf(pdb)}
}

// TestCachedFiltersRepo_PanneMetadataNonMiseEnCache : pendant une panne de metadata, les
// lignes de filtres arrivent sans leurs noms, sans erreur (best-effort) ; réparée, la
// requête suivante les relit traduites au lieu de servir le cache.
func TestCachedFiltersRepo_PanneMetadataNonMiseEnCache(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedTraductionsH5(t, pdb)
	repo := filtresCachesIsoles(pdb)
	ctx := context.Background()

	reparer := panneMetadata(t, pdb.Metadata)
	rows, err := repo.LoadMatchesForFilters(ctx)
	if err != nil || !lignesNonTraduites(rows) {
		t.Fatalf("pendant la panne : err=%v, %d ligne(s), non traduites=%v — want des lignes sans libellés, sans erreur",
			err, len(rows), lignesNonTraduites(rows))
	}
	reparer()
	rows, err = repo.LoadMatchesForFilters(ctx)
	if err != nil || len(rows) != 1 {
		t.Fatalf("après réparation : err=%v, %d ligne(s)", err, len(rows))
	}
	assertFilterName(t, "MapNameFR", rows[0].MapNameFR, "Vérité")
	assertFilterName(t, "GameVariantNameFR", rows[0].GameVariantNameFR, "Assassin")
	assertFilterName(t, "PlaylistName", rows[0].PlaylistName, "Arène en équipe")
}

// TestCachedPlayerMatchesRepo_PanneMetadataNonMiseEnCache : même garde sur l'historique
// canonique enrichi (PlayerMatchesAdapter puis EnrichCanonicalAssetTranslations).
func TestCachedPlayerMatchesRepo_PanneMetadataNonMiseEnCache(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedFilterAssetTranslations(t, pdb.Metadata, [][4]string{
		{"playlist-ranked-slayer", "playlist", "en-US", "Ranked Slayer"},
		{"playlist-ranked-slayer", "playlist", "fr-FR", "Assassin classé"},
	})
	adapter := NewPlayerMatchesAdapter(NewPlayerMatchesRepo(pdb), pdb.TitleSlug, pdb.Gamertag)
	repo := &CachedPlayerMatchesRepo{inner: adapter,
		cache: newPlayerReadCache("player_matches_cache", clonePlayerMatchRows), scope: scopeOf(pdb)}
	ctx := context.Background()
	libelleFR := func(etape string, rows []canonical.PlayerMatchRow) string {
		t.Helper()
		if len(rows) != 1 || rows[0].Summary.Playlist == nil {
			t.Fatalf("%s : %d ligne(s) ou playlist absente", etape, len(rows))
		}
		return rows[0].Summary.Playlist.Labels["fr"]
	}
	reference, err := adapter.LoadPlayerMatches(ctx, "", "", port.PlayerMatchFilters{})
	if err != nil {
		t.Fatalf("référence : %v", err)
	}
	attendu := libelleFR("référence", reference)
	if attendu == "" {
		t.Fatal("référence sans libellé FR : le test ne prouverait rien")
	}

	reparer := panneMetadata(t, pdb.Metadata)
	pendant, err := repo.LoadPlayerMatches(ctx, "", "", port.PlayerMatchFilters{})
	if err != nil {
		t.Fatalf("pendant la panne : %v (best-effort : pas d'erreur attendue)", err)
	}
	if got := libelleFR("pendant", pendant); got == attendu {
		t.Fatalf("pendant la panne : libellé FR %q déjà résolu — la panne n'a pas porté", got)
	}
	reparer()
	apres, err := repo.LoadPlayerMatches(ctx, "", "", port.PlayerMatchFilters{})
	if err != nil {
		t.Fatalf("après réparation : %v", err)
	}
	if got := libelleFR("après", apres); got != attendu {
		t.Errorf("après réparation : libellé FR %q, want %q — l'historique dégradé a été servi du cache", got, attendu)
	}
}

// dureeDeChargement : durée moyenne (5 chargements, sans cache) des lignes de filtres.
func dureeDeChargement(pdb *PlayerDB) time.Duration {
	const mesures = 5
	var total time.Duration
	for i := 0; i < mesures; i++ {
		debut := time.Now()
		_, _ = NewFiltersRepo(pdb).LoadMatchesForFilters(context.Background())
		total += time.Since(debut)
	}
	return total / mesures
}

// TestCachedFiltersRepo_AnnulationsRepartiesNEmpoisonnentPasLeCache : reprise durable du
// test de la revue B (TestRevB_FilterRowsCachePoisonedByCancelledRequest : 25
// empoisonnements sur 400 annulations avant le correctif). Des requêtes annulées à des
// instants répartis sur la fenêtre des traductions — entre la durée d'un chargement SANS
// metadata (lignes partagées et enrichissement joueur seuls) et celle d'un chargement
// complet. Quand l'annulation tombe pendant les traductions, la requête reçoit ses lignes
// non traduites SANS erreur (best-effort) ; la requête vivante suivante ne doit jamais les
// recevoir du cache.
func TestCachedFiltersRepo_AnnulationsRepartiesNEmpoisonnentPasLeCache(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedTraductionsH5(t, pdb)
	if rows, err := NewFiltersRepo(pdb).LoadMatchesForFilters(context.Background()); err != nil || lignesNonTraduites(rows) {
		t.Fatalf("témoin : err=%v, lignes traduites attendues", err)
	}
	sansMetadata := *pdb
	sansMetadata.Metadata = nil
	avantTraductions, complet := dureeDeChargement(&sansMetadata), dureeDeChargement(pdb)
	debut, fin := avantTraductions*8/10, complet*12/10

	const essais = 150
	var erreurs, completes, degradesServis, empoisonnements int
	for i := 0; i < essais; i++ {
		repo := filtresCachesIsoles(pdb)
		ctx, cancel := context.WithCancel(context.Background())
		timer := time.AfterFunc(debut+(fin-debut)*time.Duration(i)/essais, cancel)
		rows, err := repo.LoadMatchesForFilters(ctx)
		timer.Stop()
		cancel()
		switch {
		case err != nil:
			erreurs++
			continue
		case !lignesNonTraduites(rows):
			completes++
			continue
		}
		degradesServis++
		if live, err := repo.LoadMatchesForFilters(context.Background()); err == nil && lignesNonTraduites(live) {
			empoisonnements++
		}
	}
	t.Logf("fenêtre des annulations [%v, %v] (sans metadata %v, complet %v) ; %d annulations : %d erreurs, "+
		"%d complètes, %d dégradées servies à la requête annulée, %d empoisonnements",
		debut, fin, avantTraductions, complet, essais, erreurs, completes, degradesServis, empoisonnements)
	if empoisonnements > 0 {
		t.Errorf("cache empoisonné par une requête annulée : %d cas sur %d chargements dégradés", empoisonnements, degradesServis)
	}
}
