package replay

// assaut_bomb_arms_gate_test.go — LE GATE DE LA JOINTURE D'ARMEMENT sur films réels : la
// chaîne complète (anneau ti=12 -> armements datés | canal des armes tenues -> périodes de
// portage | recalage d'horloge -> `BuildBombStats`) confrontée aux résultats DÉJÀ ARBITRÉS du
// chantier bombe.
//
// # LES CRITÈRES, ÉCRITS AVANT LE PREMIER RUN
//
//	(a) NON-RÉGRESSION DU PORTAGE. Sur `35b75a31` (Neutral Bomb, 3 explosions) et `9f57c612`
//	    (One Bomb, 4 explosions), chaque explosion garde son poseur et AUCUN DÉSACCORD NEUF
//	    n'apparaît — le juge est `bpJugeExplosion`, celui du gate de portage, à l'identique.
//	    Le dénominateur du « 100 % » y est « les explosions où le statborg NOMME le détonateur
//	    ET où le porteur est ponté » : une période au slot non ponté est HORS dénominateur
//	    (`coverage.noBridge`), pas un désaccord.
//	(b) LES QUATRE DÉSACCORDS CONNUS RESTENT LES QUATRE. Sur `1c01e34f`, `3d58eb37` et
//	    `69b16f5d` — les trois films qui les portent (B2/B3, 2026-09-01) — la même règle B2
//	    rend EXACTEMENT 4 désaccords. Ni un de plus (régression), ni un de moins : B3 ne les a
//	    pas tranchés en faveur du statborg, et une disparition silencieuse serait une dérive
//	    de la chronologie, pas un progrès.
//	(c) CONTRÔLE DE COHÉRENCE DE LA JOINTURE, sur les cinq films : la somme des `bomb_arms`
//	    attribués vaut `ArmingsAttributed`, et la ventilation
//	    (attribués + sans lâcher + sans pont) retombe sur le nombre d'armements DATÉS.
//	(d) PUBLICATION : le test imprime, par film, `armements dates / attribues / non attribues`,
//	    plus la distribution SIGNÉE de (lâcher − armement) qui calibre la fenêtre de jointure.
//
// # CE QUE LE GATE NE JUGE PAS, ET POURQUOI
//
// One Bomb ne publie AUCUN armement : la confrontation locale de `buildBombArmings` retient le
// calque entier (mèche fixe de 4,93 s réfutée sur cette variante, CV 0,725 — la mèche pausable
// de 16,2 s est une lecture de RECHERCHE, elle n'est pas en production sur cette branche).
// `9f57c612` est donc au gate pour le critère (a) — la non-régression du portage — et son
// compte d'armements attendu est ZÉRO. Ce n'est pas un manque du lot : c'est la dégradation
// propre déjà gatée par `TestAssautArmementGate` (critère c2).
//
// RÉGIME : garde `ASSAUT_CACHE`. Aucune base, aucun réseau, sentinelle mémoire armée, UN SEUL
// décodage à la fois (`filmdec.LockProcessDecode`). Jamais `cmd/replay-build`.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run AssautBombArmsGate -v -timeout 60m

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// baGateFilms : les cinq films du gate. `portage` marque les deux où AUCUN désaccord n'est
// toléré (critère a) ; les trois autres portent les quatre désaccords connus (critère b).
var baGateFilms = []struct {
	id      string
	portage bool
}{
	{"35b75a31", true},  // Neutral Bomb — 3 explosions, armements publiés
	{"9f57c612", true},  // One Bomb — 4 explosions, ZÉRO armement publié (cf. l'en-tête)
	{"1c01e34f", false}, // Husky Raid
	{"3d58eb37", false},
	{"69b16f5d", false},
}

// baDesaccordsConnus est le compte FIGÉ des désaccords B2 sur les trois films non-portage.
const baDesaccordsConnus = 4

// baDiagFenetreMS borne le diagnostic imprimé autour de chaque armement : il montre les
// lâchers voisins BIEN AU-DELÀ de la fenêtre de jointure, pour qu'un mauvais dimensionnement
// de celle-ci se VOIE au journal au lieu de se deviner.
const baDiagFenetreMS = 5000

// TestAssautBombArmsGate applique les critères (a) à (d).
func TestAssautBombArmsGate(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautBombArmsGate")()
	release := filmdec.LockProcessDecode()
	defer release()

	desaccords := 0
	for _, f := range baGateFilms {
		mesure := baMesurer(t, cache, f.id)
		baPublier(t, mesure)
		baVerifierCoherence(t, mesure)

		detonateurs := b2Detonateurs(t, cache, f.id)
		if f.portage {
			for _, tE := range a5Explosions[f.id] {
				bpJugeExplosion(t, f.id, tE, mesure.periodes, detonateurs)
			}
			continue
		}
		desaccords += baCompterDesaccords(t, f.id, mesure.periodes, detonateurs)
	}
	if desaccords != baDesaccordsConnus {
		t.Errorf("desaccords B2 sur les trois films instruits : %d, reference figee %d — "+
			"la chronologie ou un pont a bouge (critere (b) NON tenu)",
			desaccords, baDesaccordsConnus)
	} else {
		t.Logf("critere (b) tenu : %d desaccords B2 sur les trois films instruits, inchanges",
			desaccords)
	}
}

// baFilm porte TOUT ce que le gate a mesuré sur un film : les deux canaux, le recalage, et ce
// que la jointure en a fait. Un seul agrégat plutôt que sept paramètres traînés de fonction en
// fonction (seuil CLAUDE.md n° 5).
type baFilm struct {
	id       string
	periodes []HeldObjectPeriod
	armings  []BombArming
	offset   int
	stats    BombMatchStats
	evts     []BombEvent
}

// baMesurer déroule la chaîne complète sur UN film : portage (chaîne de production, via
// `bpExtraire`), armements (via `agExtraire`), recalage d'horloge, puis la jointure.
//
// LE RECALAGE est calculé EXACTEMENT comme l'appelant de production devra le faire :
// `premierPaquetDuFilmUS/1000 − deathOffsetMS` (dérivation complète en tête de bomb_arms.go).
// Sa valeur est imprimée : les quatre films témoins d'origin.go la mesurent à 16-81 ms, et un
// ordre de grandeur au-dessus dirait qu'un des deux termes a changé de référentiel.
//
// `ArmingsRead` suit `Suppressed` : un calque retenu à la source (One Bomb) n'est pas un
// armement à zéro, c'est une absence de lecture — et la couverture doit le dire ainsi.
func baMesurer(t *testing.T, cache, id string) baFilm {
	t.Helper()
	periodes, _, _, own := bpExtraire(t, cache, id)
	armings, armCov := agExtraire(t, cache, id)
	filmClockUS, err := ScanFilmClockOrigin(filepath.Join(cache, "film_chunks", id))
	if err != nil {
		t.Fatalf("%s : horloge du film illisible : %v", id, err)
	}
	offset := int(int64(filmClockUS)/1000 - own.DeathOffsetMS)
	t.Logf("%s : recalage film -> match = %d ms (premier paquet %d ms, deathOffset %d ms)",
		id, offset, int64(filmClockUS)/1000, own.DeathOffsetMS)
	f := baFilm{id: id, periodes: periodes, armings: armings, offset: offset}
	f.stats, f.evts = BuildBombStats(BombStatsInput{
		ArmingsRead: !armCov.Suppressed, Armings: armings,
		CarryRead: true, Carry: HeldObjectCarry{Periods: periodes},
		FilmToMatchOffsetMS: offset,
	})
	return f
}

// baPublier imprime le bilan demandé par le critère (d) : armements datés / attribués / non
// attribués, la ventilation des non attribués, les lignes joueur, et la distribution signée
// des lâchers voisins de chaque armement.
func baPublier(t *testing.T, f baFilm) {
	t.Helper()
	cov := f.stats.Coverage
	t.Logf("%s : armements dates %d / attribues %d / non attribues %d "+
		"(sans lacher %d, slot non ponte %d)",
		f.id, cov.Armings, cov.ArmingsAttributed,
		cov.ArmingsNoDrop+cov.ArmingsNoBridge, cov.ArmingsNoDrop, cov.ArmingsNoBridge)
	for _, p := range f.stats.Players {
		if p.Arms != nil && *p.Arms > 0 {
			t.Logf("  %s : xuid %s -> bomb_arms %d", f.id, p.XUID, *p.Arms)
		}
	}
	for _, e := range f.evts {
		if e.Type != BombEventArmed {
			continue
		}
		acteur := e.XUID
		if acteur == "" {
			acteur = "SANS ACTEUR"
		}
		t.Logf("  %s : bomb_armed %7d ms (film) -> %s", f.id, e.TimeMS, acteur)
	}
	baDistributionLachers(t, f)
}

// baDistributionLachers imprime, pour chaque armement, les lâchers voisins avec leur écart
// SIGNÉ (lâcher − armement, horloge du match). Diagnostic BRUT : il n'applique pas la règle de
// jointure, il montre la matière sur laquelle elle s'applique.
func baDistributionLachers(t *testing.T, f baFilm) {
	t.Helper()
	var ecarts []int
	for _, a := range f.armings {
		armMatchMS := a.TimeMS + f.offset
		for _, p := range f.periodes {
			if p.Ouverte || p.FinParMort {
				continue
			}
			d := p.FinMS - armMatchMS
			if d < -baDiagFenetreMS || d > baDiagFenetreMS {
				continue
			}
			t.Logf("  %s : armement %7d ms (film) <- lacher slot %d xuid %d a %+6d ms",
				f.id, a.TimeMS, p.Slot, p.XUID, d)
			ecarts = append(ecarts, d)
		}
	}
	if len(ecarts) == 0 {
		return
	}
	sort.Ints(ecarts)
	t.Logf("%s : ecart lacher - armement : min %+d, mediane %+d, max %+d ms (n=%d, fenetre "+
		"de jointure +/-%d ms)",
		f.id, ecarts[0], ecarts[len(ecarts)/2], ecarts[len(ecarts)-1], len(ecarts),
		bombArmDropWindowMS)
}

// baVerifierCoherence applique le critère (c) : la somme des `bomb_arms` publiés vaut le
// nombre d'armements attribués, et la ventilation retombe sur les armements datés.
func baVerifierCoherence(t *testing.T, f baFilm) {
	t.Helper()
	cov := f.stats.Coverage
	somme := 0
	for _, p := range f.stats.Players {
		if p.Arms != nil {
			somme += *p.Arms
		}
	}
	if somme != cov.ArmingsAttributed {
		t.Errorf("%s : somme des bomb_arms = %d, armements attribues = %d — critere (c) NON tenu",
			f.id, somme, cov.ArmingsAttributed)
	}
	if somme > cov.Armings {
		t.Errorf("%s : somme des bomb_arms = %d > %d armements DATES — critere (c) NON tenu",
			f.id, somme, cov.Armings)
	}
	if v := cov.ArmingsAttributed + cov.ArmingsNoDrop + cov.ArmingsNoBridge; v != cov.Armings {
		t.Errorf("%s : ventilation %d != %d armements dates — critere (c) NON tenu",
			f.id, v, cov.Armings)
	}
}

// baCompterDesaccords applique la règle B2 (porteur à la pose contre détonateur du statborg)
// et rend le nombre de désaccords du film. C'est la MÊME règle et les MÊMES helpers que
// `TestBombeB2Assaut` : ce gate n'en écrit pas une seconde version.
func baCompterDesaccords(t *testing.T, id string, periodes []HeldObjectPeriod,
	detonateurs map[int]string,
) int {
	t.Helper()
	n := 0
	for _, tE := range a5Explosions[id] {
		xDet, okDet := detonateurs[tE]
		p, okP := b2PorteurA(periodes, tE-b3MecheMS(id), b2DernierPorteurMaxMS)
		if !okDet || !okP || p.XUID == 0 {
			continue
		}
		if strconv.FormatUint(p.XUID, 10) != xDet {
			n++
			t.Logf("%s : explosion %d — DESACCORD connu (porteur %d, detonateur %s)",
				id, tE, p.XUID, xDet)
		}
	}
	return n
}
