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
//	    attribués vaut `ArmingsAttributed`, celui-ci se ventile exactement en (par lâcher +
//	    par repli), et la ventilation complète (attribués + sans porteur + sans pont +
//	    ambigus) retombe sur le nombre d'armements DATÉS.
//	(d) PUBLICATION : le test imprime, par film, `armements dates / attribues par lacher /
//	    attribues par repli / non attribues`, la RÈGLE qui a nommé chaque acteur, la
//	    distribution SIGNÉE de (lâcher − armement) qui calibre la fenêtre de jointure, et les
//	    périodes qui COUVRENT chaque armement — la matière du repli.
//
// # ONE BOMB PUBLIE, DEPUIS LE 2026-09-04 (E2-ter)
//
// Cet en-tête a porté l'inverse, et c'est la mesure qui l'a périmé : sous la lecture SIMPLE
// (mèche fixe de 4,93 s) la confrontation locale retenait le calque entier de One Bomb (CV
// 0,725). La lecture « MÈCHE PAUSABLE » est PASSÉE EN PRODUCTION avec le schéma 39, et
// `9f57c612` publie désormais **5 armements** — 65 137 / 279 103 / 335 193 / 388 080 /
// 445 839 ms —, 4/4 explosions couvertes, mèche mesurée 16 183 ms (CV 0,010), 3 armeurs nommés
// (2 par lâcher, 1 par repli). Le statborg corrobore : pour l'explosion de 298 489 il nomme le
// joueur que la règle du lâcher a nommé sur l'armement de 279 103.
//
// `9f57c612` est donc au gate pour les critères (a), (c) et (d) comme les autres. Ce qui reste
// vrai de l'ancien en-tête, et qui vaut d'être écrit ici : DEUX des trois films One Bomb du
// corpus (`c75f33b8`, `df8fcbef`) restent RETENUS par la garde 2 — une explosion sans armement
// pour l'un, un délai corrigé aberrant pour l'autre. Ce sont exactement les deux instants
// d'`a5SansPorteur` ; relâcher la garde pour les récupérer publierait un calque expliqué aux
// trois quarts. Ni l'un ni l'autre n'est au gate.
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
	{"9f57c612", true},  // One Bomb — 4 explosions, 5 armements publiés (cf. l'en-tête)
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
// `bpExtraire`), armements (via `agExtraire`), puis LA JOINTURE PAR LE SITE DE CÂBLAGE DE LA
// PRODUCTION.
//
// LE GATE N'ÉCRIT PLUS NI LE RECALAGE NI LES TÉMOINS DE LECTURE, et c'est le correctif du
// constat I-2 de la revue de branche. Il recopiait `premierPaquetDuFilmUS/1000 −
// deathOffsetMS` et dérivait `ArmingsRead` de `!armCov.Suppressed` — une copie de plus de la
// formule, et un prédicat DIVERGENT de la production, qui exige aussi `Scanned` (un calque non
// balayé n'est pas un calque publié). Le gate assemble donc maintenant le DOCUMENT MINIMAL que
// `attachBombStats` consomme et appelle `attachBombStats` : les trois témoins de lecture et le
// recalage sortent du code livré, jamais d'une seconde écriture.
//
// `f.offset` reste calculé ici, mais POUR L'AFFICHAGE SEUL (les deux diagnostics du critère (d),
// qui posent les armements sur l'horloge du match) : il n'entre dans AUCUNE mesure ni dans aucun
// verdict. Sa valeur est imprimée — les quatre films témoins d'origin.go la mesurent à 16-81 ms,
// les cinq du gate à 33-114 ms, et un ordre de grandeur au-dessus dirait qu'un des deux termes a
// changé de référentiel. Il est ÉPINGLÉ à la production par
// `TestBombStatsCablageRecalageHorloge` (bomb_stats_wiring_test.go), qui tourne en CI et rougit
// si `attachBombStats` cesse d'appliquer cette dérivation.
func baMesurer(t *testing.T, cache, id string) baFilm {
	t.Helper()
	periodes, _, _, own := bpExtraire(t, cache, id)
	armings, armCov, _ := agExtraire(t, cache, id)
	filmClockUS, err := ScanFilmClockOrigin(filepath.Join(cache, "film_chunks", id))
	if err != nil {
		t.Fatalf("%s : horloge du film illisible : %v", id, err)
	}
	offset := int(int64(filmClockUS)/1000 - own.DeathOffsetMS)
	t.Logf("%s : recalage film -> match = %d ms (premier paquet %d ms, deathOffset %d ms) "+
		"— AFFICHAGE SEUL, la mesure passe par attachBombStats",
		id, offset, int64(filmClockUS)/1000, own.DeathOffsetMS)
	doc := ReplayDocument{MatchID: id, BombArmings: armings,
		Coverage: &Coverage{BombArmings: armCov}}
	attachBombStats(&doc, Options{FilmClockOriginUS: filmClockUS,
		Bomb: BombInput{CarryScanned: true}}, own,
		HeldObjectCarry{Periods: periodes})
	if doc.BombStats == nil {
		t.Fatalf("%s : attachBombStats n'a pose aucune statistique", id)
	}
	return baFilm{id: id, periodes: periodes, armings: armings, offset: offset,
		stats: *doc.BombStats, evts: doc.BombEvents}
}

// baPublier imprime le bilan demandé par le critère (d) : armements datés, attribués VENTILÉS
// PAR RÈGLE (lâcher / repli), non attribués avec leurs trois raisons, les lignes joueur, et la
// distribution signée des lâchers voisins de chaque armement.
//
// La ventilation par règle n'est pas cosmétique : un chiffre global de couverture qui monterait
// uniquement par le repli ne dirait PAS la même chose qu'un chiffre porté par des gestes
// observés, et c'est précisément ce qu'un lecteur doit pouvoir juger.
func baPublier(t *testing.T, f baFilm) {
	t.Helper()
	cov := f.stats.Coverage
	t.Logf("%s : armements dates %d / attribues par lacher %d / attribues par repli %d / "+
		"non attribues %d (sans porteur %d, slot non ponte %d, ambigus %d)",
		f.id, cov.Armings, cov.ArmingsByDrop, cov.ArmingsByActiveCarry,
		cov.ArmingsNoCarrier+cov.ArmingsNoBridge+cov.ArmingsAmbiguous,
		cov.ArmingsNoCarrier, cov.ArmingsNoBridge, cov.ArmingsAmbiguous)
	for _, p := range f.stats.Players {
		if p.Arms != nil && *p.Arms > 0 {
			t.Logf("  %s : xuid %s -> bomb_arms %d", f.id, p.XUID, *p.Arms)
		}
	}
	for _, e := range f.evts {
		if e.Type != BombEventArmed {
			continue
		}
		acteur, regle := e.XUID, e.ActorSource
		if acteur == "" {
			acteur, regle = "SANS ACTEUR", "-"
		}
		t.Logf("  %s : bomb_armed %7d ms (film) -> %s (regle %s)", f.id, e.TimeMS, acteur, regle)
	}
	baDistributionLachers(t, f)
	baCouverturesActives(t, f)
}

// baCouverturesActives imprime, pour chaque armement, les périodes FERMÉES qui COUVRENT
// l'instant armé — la matière du REPLI, montrée brute. Elle dit d'un coup d'œil pourquoi un
// armement est nommé par le repli (une seule couverture), pourquoi il reste ambigu (deux
// couvertures), ou pourquoi il reste anonyme (aucune).
func baCouverturesActives(t *testing.T, f baFilm) {
	t.Helper()
	for _, a := range f.armings {
		armMatchMS := a.TimeMS + f.offset
		n := 0
		for _, p := range f.periodes {
			if p.Ouverte || p.DebutMS > armMatchMS || p.FinMS < armMatchMS {
				continue
			}
			n++
			t.Logf("  %s : armement %7d ms (film) <- COUVERT par slot %d xuid %d "+
				"[%d, %d] (fin par mort : %t)",
				f.id, a.TimeMS, p.Slot, p.XUID, p.DebutMS, p.FinMS, p.FinParMort)
		}
		if n == 0 {
			t.Logf("  %s : armement %7d ms (film) <- AUCUNE periode fermee ne le couvre",
				f.id, a.TimeMS)
		}
	}
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
	if v := cov.ArmingsByDrop + cov.ArmingsByActiveCarry; v != cov.ArmingsAttributed {
		t.Errorf("%s : par regle %d (lacher %d + repli %d) != %d attribues — critere (c) NON tenu",
			f.id, v, cov.ArmingsByDrop, cov.ArmingsByActiveCarry, cov.ArmingsAttributed)
	}
	v := cov.ArmingsAttributed + cov.ArmingsNoCarrier + cov.ArmingsNoBridge + cov.ArmingsAmbiguous
	if v != cov.Armings {
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
