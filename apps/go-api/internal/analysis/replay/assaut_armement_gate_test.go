package replay

// assaut_armement_gate_test.go — LE GATE DU PORTAGE DE L'ARMEMENT : la chaîne de PRODUCTION
// (anneau ti=12 -> segments -> armements pleins et tenues de désarmement -> déduplication de
// paire -> confrontation locale à mèche MESURÉE -> publication) confrontée au relevé A0.3
// FIGÉ (les mêmes 28 explosions que le protocole du 2026-09-01).
//
// # LES CRITÈRES, RÉÉCRITS LE 2026-09-04 AVEC LA LEVÉE DE LA GARDE DE NOM
//
//	(a) sur `35b75a31` (Neutral Bomb) et `1c01e34f` (Husky Raid), la chaîne rend un calque
//	    NON VIDE et non supprimé ;
//	(b) TÉMOINS INCHANGÉS AU CHIFFRE PRÈS : chaque explosion du relevé y est précédée d'AU
//	    MOINS UN `bomb_armed`, daté à 4 930 ± 600 ms avant elle — la mèche courte, mesurée
//	    avant la lecture pausable. Chaque délai est publié au journal. Aucune tenue de
//	    désarmement n'est attendue sur ces films (17/17 des poses y ont explosé), donc le
//	    délai BRUT y vaut le délai corrigé : c'est le contrôle le plus dur possible du
//	    portage — s'il bougeait d'un armement, la lecture aurait changé de sens ;
//	(c) ONE BOMB PUBLIE MAINTENANT. Sur `9f57c612`, `c75f33b8` et `df8fcbef`, le calque est
//	    non supprimé, TOUTES les explosions du relevé sont couvertes, et la mèche MESURÉE du
//	    film est LONGUE (au-delà de la mèche courte et de sa tolérance) — la lecture
//	    « mèche pausable » du 2026-09-01, en production depuis le 2026-09-04. La garde de NOM
//	    qui écartait cette variante (`replaybuild.isArmableBombVariant`) N'EXISTE PLUS ; son
//	    ratchet vit dans `replaybuild.TestAucuneGardeParNomDeVariante`.
//
// CE QUI PROTÈGE DÉSORMAIS est la seule GARDE 2 : la confrontation locale, tout-ou-rien par
// film. Le critère (c) juge qu'elle DIT OUI là où la lecture tient ; les tests purs
// (`bomb_armings_test.go`) jugent qu'elle dit NON sur une explosion orpheline et sur des
// mèches qui se contredisent.
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
//	go test ./internal/analysis/replay/ -run AssautArmementGate -v -timeout 60m

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// agFenetreMS est la tolérance du critère (b) : 4 930 ± 600 ms, la demi-fenêtre sous laquelle
// la mèche COURTE a été mesurée (écart-type ~80 ms, CV 0,016 — la marge est large). Elle n'est
// plus une constante de production : la production MESURE la mèche du film au lieu de la
// supposer, et c'est ce gate qui vérifie que la valeur mesurée sur les témoins est bien celle
// d'avant.
const agFenetreMS = 600

// agUneBombe : les trois films de la variante One Bomb du corpus (même découpage que
// `filmdec.ti12UneBombe`, antérieur à toute mesure de ce lot), et CE QUE LA GARDE 2 EN DIT —
// FIGÉ SUR LA MESURE DU 2026-09-04, jamais sur une attente.
//
// UN SEUL DES TROIS PUBLIE, ET C'EST LE RÉSULTAT, PAS UN MANQUE. La garde 2 mord sur les deux
// autres, chacun par une branche différente — la couverture pour l'un, la dispersion pour
// l'autre. Le tout-ou-rien retient alors le film ENTIER : c'est exactement sa raison d'être,
// et le journal du gate imprime, explosion par explosion, le délai corrigé qui l'a déclenché.
//
// ET DANS LES DEUX CAS, LE FAIT QUI LA DÉCLENCHE EST UNE EXPLOSION DÉJÀ CONNUE POUR ANORMALE :
// `c75f33b8` @395 724 et `df8fcbef` @778 033 sont EXACTEMENT les deux entrées d'`a5SansPorteur`
// (mesure du 2026-08-31, ANTÉRIEURE à ce lot : les explosions dont AUCUN slot de joueur ne
// porte le point de mode — il n'existe que sur le slot d'ÉQUIPE). L'anneau ne les explique pas
// davantage que le statborg : la première n'a aucun armement dans la fenêtre de sens, la
// seconde rend 27 845 ms là où les trois autres du même film s'accordent à ~16 000. La garde 2
// n'invente donc pas un défaut, elle retrouve celui que la partition avait déjà relevé — et
// elle refuse de publier un film qu'elle n'explique qu'aux trois quarts.
type agUneBombeCas struct {
	id                    string
	publie                bool
	couvertes, explosions int
	raison                string
}

var agUneBombe = []agUneBombeCas{
	{"9f57c612", true, 4, 4,
		"les 4 explosions ont leur armement, meche mesuree ~16 183 ms (CV 0,010)"},
	{"c75f33b8", false, 2, 3,
		"l'explosion 395 724 (a5SansPorteur) n'a AUCUN armement dans la fenetre de sens ; les " +
			"deux autres rendent 16 318 et 15 900 ms — garde 2 par la COUVERTURE"},
	{"df8fcbef", false, 4, 4,
		"les 4 sont couvertes, mais l'explosion 778 033 (a5SansPorteur) rend 27 845 ms contre " +
			"15 965 / 15 929 / 16 785 pour les trois autres : CV 0,331 > 0,20 — garde 2 par la " +
			"DISPERSION"},
}

// TestAssautArmementGate applique les critères (a), (b) et (c).
func TestAssautArmementGate(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautArmementGate")()
	release := filmdec.LockProcessDecode()
	defer release()

	// (a) + (b) — LES TÉMOINS. Toute dérive ici est une régression, pas un effet de bord.
	for _, id := range []string{"35b75a31", "1c01e34f"} {
		armings, cov, v := agExtraire(t, cache, id)
		if cov.Suppressed || len(armings) == 0 {
			t.Errorf("%s : calque vide ou supprime (segments %d, armements %d, explosions %d, "+
				"couvertes %d) — critere (a) NON tenu", id, cov.Rises, cov.Armed,
				cov.Detonations, cov.DetonationsCovered)
			continue
		}
		t.Logf("%s : %d lectures, %d segments, %d armements (%d fondus de paire), %d publies, "+
			"meche MESUREE %d ms (CV %.3f)",
			id, cov.Reads, cov.Rises, cov.Armed, cov.PairMerged, cov.Published, v.FuseMS, v.CV)
		if d := v.FuseMS - BombFuseMS; d < -agFenetreMS || d > agFenetreMS {
			t.Errorf("%s : meche mesuree %d ms hors de %d +/- %d — le temoin a DERIVE (critere (b))",
				id, v.FuseMS, BombFuseMS, agFenetreMS)
		}
		agVerifierDelais(t, id, armings, a5Explosions[id])
	}

	// (c) — ONE BOMB, la variante que la garde de nom écartait.
	for _, f := range agUneBombe {
		armings, cov, v := agExtraire(t, cache, f.id)
		t.Logf("%s (One Bomb) : %d segments, %d armements, %d publies, %d/%d explosions "+
			"couvertes, meche MESUREE %d ms (CV %.3f, retenu %v, meche incoherente %v) — %s",
			f.id, cov.Rises, cov.Armed, cov.Published, cov.DetonationsCovered, cov.Detonations,
			v.FuseMS, v.CV, cov.Suppressed, v.Inconsistent, f.raison)
		agJugerUneBombe(t, f, armings, cov, v)
	}
}

// agJugerUneBombe applique le critère (c) à UN film One Bomb, contre le verdict FIGÉ de la
// table : celui qui publie doit publier (couverture pleine, mèche LONGUE), ceux que la garde 2
// retient doivent rester retenus AVEC LA MÊME COUVERTURE. Les deux sens comptent — un film qui
// se mettrait à publier sans qu'on sache pourquoi est une dérive autant qu'un film qui
// cesserait de le faire.
func agJugerUneBombe(t *testing.T, f agUneBombeCas, armings []BombArming,
	cov *BombArmingsCoverage, v bombFuseVerdict) {
	t.Helper()
	id := f.id
	if cov.Detonations != f.explosions || cov.DetonationsCovered != f.couvertes {
		t.Errorf("%s (One Bomb) : %d/%d explosions couvertes, fige %d/%d — critere (c) NON tenu",
			id, cov.DetonationsCovered, cov.Detonations, f.couvertes, f.explosions)
	}
	if publie := !cov.Suppressed && len(armings) > 0; publie != f.publie {
		t.Errorf("%s (One Bomb) : publie=%v, fige %v (retenu %v, incoherente %v, CV %.3f) — "+
			"critere (c) NON tenu", id, publie, f.publie, cov.Suppressed, v.Inconsistent, v.CV)
		return
	}
	if !f.publie {
		return
	}
	if v.FuseMS <= BombFuseMS+agFenetreMS {
		t.Errorf("%s (One Bomb) : meche mesuree %d ms — attendue LONGUE (au-dela de %d ms), "+
			"sinon la lecture pausable n'est pas celle qui a servi",
			id, v.FuseMS, BombFuseMS+agFenetreMS)
	}
	for _, det := range a5Explosions[id] {
		meilleur, trouve := 0, false
		for _, a := range armings {
			if d := det - a.TimeMS; d > 0 && (!trouve || d < meilleur) {
				meilleur, trouve = d, true
			}
		}
		if !trouve {
			t.Logf("%s : explosion %7d ms — aucun armement PUBLIE avant elle", id, det)
			continue
		}
		t.Logf("%s : explosion %7d ms <- bomb_armed a %7d ms, delai BRUT %6d ms (meche mesuree "+
			"%d ms — l'ecart est la somme des tenues de desarmement)",
			id, det, det-meilleur, meilleur, v.FuseMS)
	}
}

// agExtraire déroule la chaîne de production sur UN film : le balayage que `BuildFromFilm`
// ferait (mêmes fonctions), puis `buildBombArmings` avec l'oracle figé pour explosions.
func agExtraire(t *testing.T, cache, id string) ([]BombArming, *BombArmingsCoverage, bombFuseVerdict) {
	t.Helper()
	src, ok, err := filmcache.Open(cache, id)
	if err != nil || !ok {
		t.Fatalf("film %s absent du cache (%s) : %v — le gate exige ses films", id, cache, err)
	}
	clock := map[int]int{}
	for _, c := range src.Meta() {
		clock[c.Index] = c.StartMS
	}
	film, err := filmsource.LoadDir(filepath.Join(cache, "film_chunks", id), nil)
	if err != nil {
		t.Fatalf("chunks du film %s illisibles : %v", id, err)
	}
	reads := decodeFilmBombReads(filmdec.NewFilmContext(film), id, BombInput{Scanned: true, ChunkStartMS: clock})
	agDiagnostiquerSegments(t, id, reads, a5ExplosionTimes(id))
	// Grille synthétique : originMS=0, pas 100 ms, axe assez long pour tout le film — le gate
	// juge les délais en ms, la conversion en frames est couverte par les tests unitaires.
	return buildBombArmings(reads, a5ExplosionTimes(id), scoreClock{intervalMS: 100, frames: 1 << 20})
}

// agDiagnostiquerSegments publie CHAQUE armement dédupliqué avec ses quanta, et CHAQUE tenue
// de désarmement avec sa pente : c'est la matière brute de la lecture pausable, montrée avant
// tout verdict. Le marqueur `<- EXPLOSION` dit quels armements une explosion suit de près.
func agDiagnostiquerSegments(t *testing.T, id string, reads []filmdec.NavpointRadialRead, explosions []int) {
	t.Helper()
	cov := &BombArmingsCoverage{}
	full, pauses := classifyBombSegments(filmdec.NavpointSegments(reads), cov)
	for _, r := range dedupPairedSegments(full, cov) {
		lien := ""
		for _, det := range explosions {
			if d := det - int(r.EndMS); d > 0 && d <= BombFuseSenseWindowMS {
				lien = fmt.Sprintf(" <- EXPLOSION a +%d ms", d)
			}
		}
		t.Logf("%s : armement %7d..%7d ms, q %3d -> %3d (min %3d, max %3d), %2d ech.%s",
			id, r.StartMS, r.EndMS, r.QStart, r.QEnd, r.QMin, r.QMax, r.Samples, lien)
	}
	for slot, ps := range pauses {
		for _, p := range ps {
			durS := float64(p.EndMS-p.StartMS) / 1000
			t.Logf("%s : tenue de desarmement slot %d, %7d..%7d ms (%.1f s), q %3d -> %3d, "+
				"pente %.1f q/s", id, slot, p.StartMS, p.EndMS, durS, p.QStart, p.QEnd,
				float64(int(p.QStart)-int(p.QEnd))/durS)
		}
	}
	// LE DÉLAI CORRIGÉ, EXPLOSION PAR EXPLOSION — c'est la matière exacte sur laquelle la
	// garde 2 statue : une explosion sans armement dans la fenêtre de sens la fait mordre par
	// la couverture, un délai qui s'écarte des autres la fait mordre par la dispersion.
	for _, det := range explosions {
		d, ok := bombCorrectedDelay(full, pauses, int32(det))
		if !ok {
			t.Logf("%s : explosion %7d ms — AUCUN armement dans la fenetre de sens (garde 2)",
				id, det)
			continue
		}
		t.Logf("%s : explosion %7d ms — delai CORRIGE %6d ms", id, det, d)
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
		t.Logf("%s : explosion %7d ms <- bomb_armed a %7d ms, delai %4d ms (meche courte %d ms)",
			id, det, det-meilleur, meilleur, BombFuseMS)
	}
}
