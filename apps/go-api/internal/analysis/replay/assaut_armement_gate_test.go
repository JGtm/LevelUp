package replay

// assaut_armement_gate_test.go — LE GATE DU PORTAGE DE L'ARMEMENT : la chaîne de PRODUCTION
// (anneau ti=12 -> montées contiguës -> déduplication de paire -> confrontation locale ->
// publication) confrontée au relevé A0.3 FIGÉ (les mêmes 28 explosions que le protocole du
// 2026-09-01), sur les trois films désignés du portage.
//
// # LES CRITÈRES, ÉCRITS AVANT LE PREMIER RUN
//
//	(a) sur `35b75a31` (Neutral Bomb) et `1c01e34f` (Husky Raid), la chaîne rend un calque
//	    NON VIDE et non supprimé ;
//	(b) chaque explosion du relevé y est précédée d'AU MOINS UN `bomb_armed`, daté à
//	    4 930 ± 600 ms avant elle — chaque délai mesuré est publié au journal ;
//	(c) sur `9f57c612` (One Bomb), ZÉRO événement publié — dégradation propre :
//	    c1. la garde de NOM ne pose jamais `Scanned` (`replaybuild.isArmableBombVariant`,
//	        testée dans son paquet : TestIsArmableBombVariant) ;
//	    c2. défense en profondeur : même `Scanned` FORCÉ, la confrontation locale retient le
//	        calque entier (`Suppressed`) — aucune explosion One Bomb n'a d'armement à la mèche.
//
// L'ORACLE est `a5Explosions` (assaut_a5_explosions_test.go) : le relevé A0.3, commité le
// 2026-08-27, antérieur à la percée ti=12 — il ne doit rien au canal qu'il juge. L'HORLOGE de
// confrontation est une grille synthétique (originMS=0) : le gate juge les DÉLAIS en
// millisecondes du manifeste, pas la conversion en frames (couverte par les tests unitaires).
//
// RÉGIME : garde `ASSAUT_CACHE`. Aucune base, aucun réseau, sentinelle mémoire armée, UN SEUL
// décodage à la fois (`filmdec.LockProcessDecode`). Jamais `cmd/replay-build` : l'extraction
// est en processus.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run AssautArmementGate -v -timeout 30m

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// agFenetreMS est la tolérance du critère (b) : 4 930 ± 600 ms. C'est la fenêtre de la
// confrontation locale de production (bombFuseWindowMS) — le gate vérifie qu'elles concordent
// pour que son verdict porte sur la constante réellement livrée.
const agFenetreMS = 600

// TestAssautArmementGate applique les critères (a), (b) et (c2).
func TestAssautArmementGate(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	if agFenetreMS != bombFuseWindowMS {
		t.Fatalf("fenetre du gate (%d ms) != fenetre de production (%d ms) — accorder avant de juger",
			agFenetreMS, bombFuseWindowMS)
	}
	defer amArmeSentinelle(t, "TestAssautArmementGate")()
	release := filmdec.LockProcessDecode()
	defer release()

	// (a) + (b) — les deux films où le canal est prouvé.
	for _, id := range []string{"35b75a31", "1c01e34f"} {
		armings, cov := agExtraire(t, cache, id)
		if cov.Suppressed || len(armings) == 0 {
			t.Errorf("%s : calque vide ou supprime (montees %d, armements %d, explosions %d, "+
				"couvertes %d) — critere (a) NON tenu", id, cov.Rises, cov.Armed,
				cov.Detonations, cov.DetonationsCovered)
			continue
		}
		t.Logf("%s : %d lectures, %d montees, %d armements (%d fondus de paire), %d publies",
			id, cov.Reads, cov.Rises, cov.Armed, cov.PairMerged, cov.Published)
		agVerifierDelais(t, id, armings, a5Explosions[id])
	}

	// (c2) — One Bomb, garde de nom court-circuitée : la confrontation locale doit retenir.
	armings, cov := agExtraire(t, cache, "9f57c612")
	if !cov.Suppressed || len(armings) != 0 {
		t.Errorf("9f57c612 (One Bomb, Scanned force) : %d evenement(s) publie(s), suppressed=%v "+
			"— critere (c2) NON tenu (montees %d, explosions %d, couvertes %d)",
			len(armings), cov.Suppressed, cov.Rises, cov.Detonations, cov.DetonationsCovered)
	} else {
		t.Logf("9f57c612 : calque retenu a la source (montees %d, explosions %d, couvertes %d) "+
			"— degradation propre", cov.Rises, cov.Detonations, cov.DetonationsCovered)
	}
}

// agExtraire déroule la chaîne de production sur UN film : le balayage que `BuildFromFilm`
// ferait (mêmes fonctions), puis `buildBombArmings` avec l'oracle figé pour explosions.
func agExtraire(t *testing.T, cache, id string) ([]BombArming, *BombArmingsCoverage) {
	t.Helper()
	src, ok, err := filmcache.Open(cache, id)
	if err != nil || !ok {
		t.Fatalf("film %s absent du cache (%s) : %v — le gate exige les trois films", id, cache, err)
	}
	clock := map[int]int{}
	for _, c := range src.Meta() {
		clock[c.Index] = c.StartMS
	}
	film, err := filmsource.LoadDir(filepath.Join(cache, "film_chunks", id), nil)
	if err != nil {
		t.Fatalf("chunks du film %s illisibles : %v", id, err)
	}
	reads := decodeFilmBombReads(film, id, BombInput{Scanned: true, ChunkStartMS: clock})
	agDiagnostiquerMontees(t, id, reads, a5ExplosionTimes(id))
	// Grille synthétique : originMS=0, pas 100 ms, axe assez long pour tout le film — le gate
	// juge les délais en ms, la conversion en frames est couverte par les tests unitaires.
	return buildBombArmings(reads, a5ExplosionTimes(id), scoreClock{intervalMS: 100, frames: 1 << 20})
}

// agDiagnostiquerMontees publie CHAQUE montée dédupliquée avec ses quanta : c'est la mesure qui
// départage une montée COMPLÈTE (le plein de l'anneau -> bombe armée) d'une montée AVORTÉE (un
// hold relâché). Le marqueur `<- EXPLOSION` dit lesquelles la mèche confirme.
func agDiagnostiquerMontees(t *testing.T, id string, reads []filmdec.NavpointRadialRead, explosions []int) {
	t.Helper()
	cov := &BombArmingsCoverage{}
	for _, r := range dedupPairedRises(filmdec.NavpointContiguousRises(reads), cov) {
		lien := ""
		for _, det := range explosions {
			if d := det - int(r.EndMS); d >= BombFuseMS-agFenetreMS && d <= BombFuseMS+agFenetreMS {
				lien = " <- EXPLOSION"
			}
		}
		t.Logf("%s : montee %7d..%7d ms, q %3d -> %3d, %2d ech.%s",
			id, r.StartMS, r.EndMS, r.QStart, r.QEnd, r.Samples, lien)
	}
}

// a5ExplosionTimes rend les instants du relevé pour un film, copiés (l'oracle reste figé).
func a5ExplosionTimes(id string) []int {
	return append([]int(nil), a5Explosions[id]...)
}

// agVerifierDelais applique le critère (b) : chaque explosion du relevé a AU MOINS un
// `bomb_armed` à 4 930 ± 600 ms avant elle. Chaque délai est publié au journal.
func agVerifierDelais(t *testing.T, id string, armings []BombArming, explosions []int) {
	t.Helper()
	for _, det := range explosions {
		meilleur, trouve := 0, false
		for _, a := range armings {
			d := det - a.TimeMS
			if d < BombFuseMS-agFenetreMS || d > BombFuseMS+agFenetreMS {
				continue
			}
			meilleur, trouve = d, true
		}
		if !trouve {
			t.Errorf("%s : explosion a %d ms SANS bomb_armed dans [%d, %d] ms avant elle — "+
				"critere (b) NON tenu", id, det, BombFuseMS-agFenetreMS, BombFuseMS+agFenetreMS)
			continue
		}
		t.Logf("%s : explosion %7d ms <- bomb_armed a %7d ms, delai %4d ms (meche %d ms)",
			id, det, det-meilleur, meilleur, BombFuseMS)
	}
}
