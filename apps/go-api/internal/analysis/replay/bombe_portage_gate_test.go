package replay

// bombe_portage_gate_test.go — LE GATE DU PORTAGE DE LA BOMBE (schéma 30) : la chaîne de
// PRODUCTION (canal des armes tenues -> bombHeldEventsOf -> BuildHeldObjectCarry ->
// buildBombCarries) confrontée au relevé A0.3 FIGÉ et au détonateur du statborg, sur les
// deux films désignés par la mission.
//
// # LES CRITÈRES, ÉCRITS AVANT LE PREMIER RUN
//
//	(a) sur `35b75a31` (Neutral Bomb, 3 explosions) et `9f57c612` (One Bomb, 4 explosions,
//	    manches 0..3), la chaîne rend un calque NON VIDE et une couverture ÉQUILIBRÉE —
//	    One Bomb DOIT publier : la garde du portage couvre toute la famille bomb ;
//	(b) CHAQUE explosion du relevé a AU MOINS une période de portage qui l'explique : une
//	    période active à tPose = tE − 4 930 ms, ou fermée dans les 60 s avant (la fenêtre de
//	    rattrapage du protocole B2 — transport + armement ; en One Bomb la mèche réelle est
//	    plus longue, le rattrapage la couvre). Le juge est la CHRONOLOGIE RECONSTRUITE
//	    (périodes pontées OU NON, la règle exacte de B2/`b2PorteurA`) : une période au slot
//	    non ponté EST un portage vu par le canal — la publication ne peut pas le NOMMER,
//	    et c'est ce que `coverage.noBridge` dit au lecteur ;
//	(c) quand le statborg NOMME le détonateur ET que le porteur est ponté (le dénominateur
//	    de B2, « les explosions où les deux côtés sont résolus »), porteur = poseur à 100 %
//	    SUR CES DEUX FILMS — les quatre désaccords du corpus B2 vivent sur
//	    `1c01e34f`/`3d58eb37`/`69b16f5d`, aucun ici : un désaccord neuf est une régression.
//
// PREMIER RUN (2026-09-01), et ce qu'il a corrigé : la première écriture jugeait (b) et (c)
// sur les seuls portages PUBLIÉS. Sur chacun des deux films, UNE période au slot non ponté
// (`noBridge` = 1) précédait une explosion ; le juge sautait alors à un porteur antérieur
// ponté et fabriquait un faux désaccord (787051 de 35b75a31), ou ne trouvait rien (83322 de
// 9f57c612). B2 jugeait déjà sur la chronologie brute et classait ces cas hors dénominateur.
//
// L'ORACLE est `a5Explosions` (relevé A0.3, commité le 2026-08-27, antérieur à la percée du
// canal) et le détonateur `b2Detonateurs` (pont statborg par manche, indépendant du pont de
// bipède jugé). L'HORLOGE du gate est la grille identité (origin 0, step 1000 µs -> frame ==
// ms match) : le gate juge les DÉLAIS en millisecondes, la conversion en frames réelle est
// couverte par les tests unitaires. La CONVERSION de production (TimestampUS − deathOffsetMS
// du pont) diffère du zéro du manifeste de 16-81 ms mesurés — trois ordres de grandeur sous
// la fenêtre jugée.
//
// RÉGIME : garde `ASSAUT_CACHE`. Aucune base, aucun réseau, sentinelle mémoire armée, UN
// SEUL décodage à la fois (`filmdec.LockProcessDecode`). Jamais `cmd/replay-build` :
// l'extraction est en processus.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run BombePortageGate -v -timeout 30m

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// bpFilms : les deux films du gate, et le compte d'explosions attendu du relevé.
var bpFilms = []struct {
	id         string
	explosions int
}{
	{"35b75a31", 3}, // Neutral Bomb, 1 manche
	{"9f57c612", 4}, // One Bomb, manches 0..3 — le portage DOIT y être publié
}

// TestBombePortageGate applique les critères (a), (b) et (c).
func TestBombePortageGate(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestBombePortageGate")()
	release := filmdec.LockProcessDecode()
	defer release()

	for _, f := range bpFilms {
		periodes, carries, cov, _ := bpExtraire(t, cache, f.id)
		detonateurs := b2Detonateurs(t, cache, f.id)
		bpLog(t, f.id, periodes, cov)
		// (a) — calque non vide, couverture équilibrée.
		if len(carries) == 0 {
			t.Errorf("%s : calque VIDE (transitions %d, periodes %d, sansPont %d) — critere (a) NON tenu",
				f.id, cov.Events, cov.Periods, cov.NoBridge)
			continue
		}
		if !cov.Balanced() {
			t.Errorf("%s : couverture desequilibree %+v — critere (a) NON tenu", f.id, cov)
		}
		if len(a5Explosions[f.id]) != f.explosions {
			t.Errorf("%s : releve A0.3 a bouge (%d explosions, attendu %d)",
				f.id, len(a5Explosions[f.id]), f.explosions)
		}
		// (b) + (c) — chaque explosion a son porteur ; porteur = poseur quand les deux côtés
		// sont résolus (la règle B2, sur la chronologie brute).
		for _, tE := range a5Explosions[f.id] {
			bpJugeExplosion(t, f.id, tE, periodes, detonateurs)
		}
	}
}

// bpExtraire déroule la chaîne de production sur UN film : les MÊMES fonctions que
// `BuildFromFilm`/`attachBombCarries`, sur la grille identité (frame == ms match).
//
// Le prédicat de spawn du balayage est nil : il ne sert qu'à qualifier les ré-annonces
// (`Restated`), que `bombHeldEventsOf` ne consulte pas — la bombe n'est jamais une arme de
// spawn. Le gate de présence est nil, comme dans les tests unitaires : les pistes publiées
// n'existent pas hors assemblage complet, et ce que le gate juge est la chronologie.
//
// L'`OwnerReport` est rendu en quatrième valeur parce que son `DeathOffsetMS` EST la moitié du
// recalage d'horloge de la jointure d'armement (cf. bomb_arms.go) : le gate de `bomb_arms`
// réutilise cette extraction plutôt que d'en payer une seconde sur le même film.
func bpExtraire(t *testing.T, cache, id string) (
	[]HeldObjectPeriod, []BombCarry, *BombCarriesCoverage, OwnerReport,
) {
	t.Helper()
	dir := filepath.Join(cache, "film_chunks", id)
	changes, _, err := filmdec.ScanFilmHeldWeaponChanges(dir, nil)
	if err != nil {
		t.Fatalf("%s : canal des armes tenues illisible : %v", id, err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("%s : fil des morts illisible : %v", id, err)
	}
	roster := map[uint64]bool{}
	for _, d := range deaths {
		roster[d.XUID] = true
	}
	xuids := make([]uint64, 0, len(roster))
	for x := range roster {
		xuids = append(xuids, x)
	}
	sort.Slice(xuids, func(i, j int) bool { return xuids[i] < xuids[j] })
	opt := filmdec.DefaultScanFilmOptions()
	opt.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("%s : positions bipeds illisibles : %v", id, err)
	}
	idx, err := ScanFilmPlayerIndices(dir, xuids)
	if err != nil {
		t.Logf("%s : index de joueur illisible (%v) — pont par le seul fil des morts", id, err)
	}
	slotXUID, own := ResolveSlotXUID(pos, deaths, idx)
	events := bombHeldEventsOf(changes, own.DeathOffsetMS)
	carry := BuildHeldObjectCarry(events, slotXUID, deaths)
	carries, cov := buildBombCarries(carry, matchClock{origin: 0, step: 1000, frames: 1 << 20}, nil)
	return carry.Periods, carries, cov, own
}

// bpJugeExplosion applique (b) et (c) à UNE explosion du relevé, sur la chronologie BRUTE
// (le juge de B2, `b2PorteurA` — les périodes non pontées comptent pour (b), et sortent du
// dénominateur de (c) comme dans la mesure).
func bpJugeExplosion(t *testing.T, id string, tE int, periodes []HeldObjectPeriod, detonateurs map[int]string) {
	t.Helper()
	tPose := tE - b2MecheMS
	p, ok := b2PorteurA(periodes, tPose, b2DernierPorteurMaxMS)
	if !ok {
		t.Errorf("%s : explosion a %d ms SANS periode de portage active a la pose ni dans les %d s "+
			"avant — critere (b) NON tenu", id, tE, b2DernierPorteurMaxMS/1000)
		return
	}
	xDet, okDet := detonateurs[tE]
	switch {
	case !okDet:
		t.Logf("%s : explosion %7d ms <- portage slot %d [%d, %d] ; detonateur NON nomme par le "+
			"statborg — hors denominateur (c)", id, tE, p.Slot, p.DebutMS, p.FinMS)
	case p.XUID == 0:
		t.Logf("%s : explosion %7d ms <- portage slot %d [%d, %d] NON ponte (coverage.noBridge) "+
			"— hors denominateur (c)", id, tE, p.Slot, p.DebutMS, p.FinMS)
	case strconv.FormatUint(p.XUID, 10) != xDet:
		t.Errorf("%s : explosion a %d ms — porteur %d, detonateur %s (portage [%d, %d]) : "+
			"critere (c) NON tenu (aucun desaccord attendu sur ce film)",
			id, tE, p.XUID, xDet, p.DebutMS, p.FinMS)
	default:
		t.Logf("%s : explosion %7d ms <- poseur = detonateur %s (portage [%d, %d], delai fin -> "+
			"explosion %d ms)", id, tE, xDet, p.DebutMS, p.FinMS, tE-p.FinMS)
	}
}

// bpLog journalise la chronologie brute et la couverture — les chiffres que le CR publiera.
func bpLog(t *testing.T, id string, periodes []HeldObjectPeriod, cov *BombCarriesCoverage) {
	t.Helper()
	t.Logf("%s : %d transitions, %d periodes, %d portages publies (%d fermes dont %d par mort, "+
		"%d ouverts), %d sansPont, %d horsFenetre",
		id, cov.Events, cov.Periods, cov.Carries, cov.Closed, cov.ByDeath, cov.Open,
		cov.NoBridge, cov.OutOfWindow)
	for _, p := range periodes {
		fin := strconv.Itoa(p.FinMS)
		if p.Ouverte {
			fin = "fin de film"
		}
		mort := ""
		if p.FinParMort {
			mort = " (mort)"
		}
		t.Logf("  %s : periode slot %d xuid %d [%d, %s]%s", id, p.Slot, p.XUID, p.DebutMS, fin, mort)
	}
}
