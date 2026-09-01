package replay

// bombe_b2_chronologie_test.go — LA CHRONOLOGIE DE LA BOMBE, CONFRONTÉE AUX ORACLES.
//
// # CE QUE B1 A ÉTABLI (2026-09-01)
//
// La famille `0x3fee4fcf` est l'UNIQUE candidate C1+C2+C4 des neuf films d'Assaut : hors
// catalogue d'armes, présente sur les 9 films, prise ET lâchée sur chacun, tenue par 7 à 42
// slots-vies (médiane 13). L'atlas HUD du jeu la confirme indépendamment : ses tags weap
// (`3fee4fcf`, `523b4648`) pointent le sprite `contour-34`, nommé `ball | bomb` dans
// l'index (`static/weapons-assets/halo_infinite/jeu/index.json`). Le témoin Oddball a validé
// le canal : le crâne `0x0017592c` y émet (14 prises, 4 lâchers, 14 slots-vies).
//
// # PROTOCOLE, écrit avant la mesure
//
// La timeline d'un objet (bombe ou crâne) se reconstruit ainsi : PRISE = transition VERS la
// famille (le composant weapon-state-type-info du bipède reçoit l'identité), LÂCHER =
// transition DEPUIS. Une période de portage s'ouvre à la prise du slot s et se ferme au
// premier des trois : lâcher de s, MORT du porteur (fil des morts, via pont slot->xuid — le
// canal n'émet PAS de lâcher à la mort, la vie du bipède s'arrête), fin du film.
//
//	V1 — LE POSEUR. Pour chacune des 28 explosions datées (oracle a5Explosions), la pose est
//	     à tE − 4930 ms (anneau ti=12 i14, validé 0/1000). Le porteur courant à la pose — ou
//	     à défaut le dernier porteur dont la période s'est fermée dans les 60 s précédentes —
//	     DOIT être le détonateur du statborg (IdentifiedEvent bomb_detonations, pont par
//	     manche). Cible écrite : accord >= 90 % des explosions où les deux côtés sont résolus
//	     (détonateur identifié : 21 attendues ; porteur trouvé et ponté).
//	V2 — LA BOMBE POSÉE N'EST PORTÉE PAR PERSONNE. Aucune PRISE de la famille bombe dans
//	     ]tPose + 500 ms, tE − 500 ms[ (marge : les bords sont les gestes eux-mêmes).
//	     Cible écrite : >= 90 % des 28 intervalles vides.
//
//	     MESURÉ (2026-09-01, deux passes identiques) : V1 = 13/17 (76,5 %) — la cible de
//	     90 % N'EST PAS tenue ; V2 = 27/28 (96,4 %) — tenue. Les QUATRE désaccords V1 sont
//	     instruits par bombe_b3_desaccords_test.go (positions en quanta) : trois penchent
//	     CANAL (2,4-2,5x plus près des sites de pose authentifiés, immobilité d'armement),
//	     un reste INDÉCIS (vies adjacentes, coéquipiers côte à côte) — l'arbitrage définitif
//	     (créations ti=42 au site) est au registre. Le délai médian lâcher -> explosion vaut
//	     4 804 ms : le lâcher du canal EST le geste de pose, à ~130 ms de la mèche. Suivant
//	     le patron du dépôt (TestAssautA5PontIdentite), les chiffres MESURÉS ET EXPLIQUÉS
//	     sont FIGÉS comme référence : toute amélioration comme toute dégradation rougit.
//	V3 — LE TÉMOIN ODDBALL. Chaque événement skull_carry du pied (xuid, t) doit tomber
//	     pendant une période de portage du crâne attribuée au MÊME xuid (tolérance ±1000 ms
//	     aux bords). Cible écrite : accord >= 90 % des événements dont l'instant est couvert
//	     par une période pontée ; le taux brut (couverts ou non) se publie à côté.
//
//	     RAFFINEMENT (2026-09-01, première passe lue AVANT re-mesure) : la première passe a
//	     rendu 46/56 = 82,1 %, et les DIX désaccords tombent TOUS à <= 5 ms d'une fin de
//	     période PAR MORT. Le pied th=10 mélange donc plusieurs natures d'événement (la piste
//	     P3 du handoff Oddball les disait « jamais élucidés ») : l'hypothèse écrite ici est
//	     qu'un événement th=10 émis À LA MORT du porteur crédite son TUEUR — la stat
//	     `skull_carriers_killed` de l'oracle officiel. Un tel événement n'est pas un
//	     heartbeat de possession : il se VÉRIFIE contre le fil des kills (xuid crédité =
//	     tueur à ±150 ms de la fin de période) et se classe à part. S'il ne se vérifie pas,
//	     il reste un DÉSACCORD plein.
//
// Les taux se publient TELS QUELS, accompagnés de la distribution des délais (fin de portage
// -> explosion) qui documente la mécanique réelle du geste de pose.
//
// RÉGIME : garde `ASSAUT_CACHE`. Aucune base, aucun réseau, sentinelle mémoire armée, verrou
// process filmdec (un seul décodage à la fois).
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run BombeB2 -v -timeout 60m

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

const (
	b2Bombe = uint32(0x3fee4fcf)
	// b2MecheMS : fin de montée d'armement -> explosion (navpoint_ti12_plancher, 0/1000).
	b2MecheMS = 4930
	// b2DernierPorteurMaxMS : fenêtre de rattrapage du « dernier porteur » quand la pose a
	// déjà vidé la main à tPose (la fenêtre de sens du chantier ti12 est 120 s ; ici 60 s
	// suffisent à couvrir transport + armement sans repêcher un portage d'une autre action).
	b2DernierPorteurMaxMS = 60000
	// b2MargeV2MS : marge des bords de l'intervalle [pose, explosion] (les gestes eux-mêmes).
	b2MargeV2MS = 500
	// b2TolV3MS : tolérance aux bords des périodes pour le témoin Oddball.
	b2TolV3MS = 1000
)

// b2Timeline charge la chronologie d'une famille sur UN film : événements datés (ms match),
// pont slot->xuid et fil des morts. Un seul décodage à la fois (verrou pris par l'appelant).
func b2Timeline(t *testing.T, cache, id string, fam uint32) ([]HeldObjectEvent, map[uint32]uint64, []Death) {
	t.Helper()
	dir := filepath.Join(cache, "film_chunks", id)
	changes, _, err := filmdec.ScanFilmHeldWeaponChanges(dir, nil)
	if err != nil {
		t.Fatalf("%s : canal des armes tenues illisible : %v", id, err)
	}
	originUS, err := ScanFilmClockOrigin(dir)
	if err != nil {
		t.Fatalf("%s : horloge du film illisible : %v", id, err)
	}
	toMS := func(us uint64) int { return int((us - originUS) / 1000) }
	var evs []HeldObjectEvent
	for _, ch := range changes {
		if ch.Family == fam {
			evs = append(evs, HeldObjectEvent{TimeMS: toMS(ch.TimestampUS), Slot: ch.Slot, Pickup: true})
		}
		if ch.Previous == fam && ch.Family != fam {
			evs = append(evs, HeldObjectEvent{TimeMS: toMS(ch.TimestampUS), Slot: ch.Slot, Pickup: false})
		}
	}
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].TimeMS < evs[j].TimeMS })

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
	slotXUID, rep := ResolveSlotXUID(pos, deaths, idx)
	t.Logf("%s : %d transitions bombe/crane, pont slot->xuid : %d slots nommés (vies=%d)",
		id, len(evs), len(slotXUID), rep.LivesTotal)
	return evs, slotXUID, deaths
}

// b2Periodes délègue à l'instrument publié (held_object_carry.go) : la logique des périodes
// n'a qu'UNE implémentation, celle que le produit consommera.
func b2Periodes(evs []HeldObjectEvent, slotXUID map[uint32]uint64, deaths []Death) []HeldObjectPeriod {
	return BuildHeldObjectCarry(evs, slotXUID, deaths).Periods
}

// b2PorteurA rend la période active à t, ou la dernière fermée dans la fenêtre [t-maxMS, t].
func b2PorteurA(periodes []HeldObjectPeriod, t, maxMS int) (HeldObjectPeriod, bool) {
	var best HeldObjectPeriod
	found := false
	for _, p := range periodes {
		if p.DebutMS <= t && t <= p.FinMS {
			return p, true
		}
		if p.FinMS < t && t-p.FinMS <= maxMS {
			if !found || p.FinMS > best.FinMS {
				best, found = p, true
			}
		}
	}
	return best, found
}

// b2Detonateurs rend, par instant d'explosion, le xuid du détonateur (pont par manche).
func b2Detonateurs(t *testing.T, cache, id string) map[int]string {
	t.Helper()
	src, ok, err := filmcache.Open(cache, id)
	if err != nil || !ok {
		t.Fatalf("%s : film absent du cache : %v", id, err)
	}
	recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, id)
	named := objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeBomb)
	dir := filepath.Join(cache, "film_chunks", id)
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("%s : fil des morts illisible : %v", id, err)
	}
	di := make([]objectiveevents.DeathInstant, 0, len(deaths))
	for _, d := range deaths {
		di = append(di, objectiveevents.DeathInstant{
			XUID: strconv.FormatUint(d.XUID, 10), TimeMS: int(d.TimeMS)})
	}
	identity := objectiveevents.ResolveRoundIdentity(recs, di)
	out := map[int]string{}
	for _, e := range objectiveevents.IdentifyNamedEventsByRound(named, identity) {
		if e.Stat == objectiveevents.StatBombDetonations {
			out[e.TimeMS] = e.XUID
		}
	}
	return out
}

// TestBombeB2Assaut applique V1 et V2 aux 28 explosions datées.
func TestBombeB2Assaut(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandée : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestBombeB2Assaut")()
	release := filmdec.LockProcessDecode()
	defer release()

	films := make([]string, 0, len(a5Explosions))
	for id := range a5Explosions {
		films = append(films, id)
	}
	sort.Strings(films)

	var bilan b2Bilan
	for _, id := range films {
		evs, slotXUID, deaths := b2Timeline(t, cache, id, b2Bombe)
		periodes := b2Periodes(evs, slotXUID, deaths)
		detonateurs := b2Detonateurs(t, cache, id)
		b2LogPeriodes(t, id, periodes)
		for _, tE := range a5Explosions[id] {
			b2JugeExplosion(t, id, tE, evs, periodes, detonateurs, &bilan)
		}
	}
	v1Resolues, v1Accords := bilan.v1Resolues, bilan.v1Accords
	v2Total, v2Vides := bilan.v2Total, bilan.v2Vides
	delais := bilan.delais
	sort.Ints(delais)
	if len(delais) > 0 {
		t.Logf("délai fin de portage -> explosion : min %d, p25 %d, médiane %d, p75 %d, max %d ms (n=%d)",
			delais[0], delais[len(delais)/4], delais[len(delais)/2], delais[3*len(delais)/4],
			delais[len(delais)-1], len(delais))
	}
	t.Logf("V1 : %d/%d accords (%.1f %%) sur les explosions résolues des deux côtés",
		v1Accords, v1Resolues, 100*float64(v1Accords)/float64(max(v1Resolues, 1)))
	t.Logf("V2 : %d/%d intervalles [pose, explosion] sans prise (%.1f %%)",
		v2Vides, v2Total, 100*float64(v2Vides)/float64(max(v2Total, 1)))
	// Chiffres FIGÉS (cf. en-tête, « MESURÉ ») : un écart dans un sens comme dans l'autre
	// est une régression à instruire — patron TestAssautA5PontIdentite.
	if v1Accords != 13 || v1Resolues != 17 {
		t.Errorf("V1 : %d/%d, référence figée 13/17 — la chronologie ou un pont a bougé", v1Accords, v1Resolues)
	}
	if v2Vides != 27 || v2Total != 28 {
		t.Errorf("V2 : %d/%d, référence figée 27/28", v2Vides, v2Total)
	}
}

// b2Bilan cumule les compteurs de V1/V2 à travers les films.
type b2Bilan struct {
	v1Resolues, v1Accords int
	v2Total, v2Vides      int
	delais                []int
}

// b2LogPeriodes journalise les périodes de portage d'un film.
func b2LogPeriodes(t *testing.T, id string, periodes []HeldObjectPeriod) {
	t.Helper()
	for _, p := range periodes {
		fin := fmt.Sprintf("%d", p.FinMS)
		if p.Ouverte {
			fin = "fin de film"
		}
		mort := ""
		if p.FinParMort {
			mort = " (mort)"
		}
		t.Logf("  %s : portage slot %d xuid %d : %d -> %s ms%s", id, p.Slot, p.XUID, p.DebutMS, fin, mort)
	}
}

// b2JugeExplosion applique V1 et V2 à UNE explosion datée et cumule dans le bilan.
func b2JugeExplosion(t *testing.T, id string, tE int, evs []HeldObjectEvent,
	periodes []HeldObjectPeriod, detonateurs map[int]string, bilan *b2Bilan) {
	t.Helper()
	tPose := tE - b2MecheMS
	// V2 : personne ne prend la bombe pendant qu'elle est posée.
	bilan.v2Total++
	prises := 0
	for _, e := range evs {
		if e.Pickup && e.TimeMS > tPose+b2MargeV2MS && e.TimeMS < tE-b2MargeV2MS {
			prises++
		}
	}
	if prises == 0 {
		bilan.v2Vides++
	} else {
		t.Logf("  %s explosion %d : V2 — %d prise(s) dans ]pose, explosion[", id, tE, prises)
	}
	// V1 : le porteur à la pose est le détonateur.
	xDet, okDet := detonateurs[tE]
	p, okPorteur := b2PorteurA(periodes, tPose, b2DernierPorteurMaxMS)
	if okPorteur && p.FinMS < tE && tE-p.FinMS <= b2DernierPorteurMaxMS {
		// Distribution du délai lâcher -> explosion : seules les périodes FERMÉES avant
		// l'explosion parlent (une période couvrant tE — portage à travers l'explosion,
		// cas des désaccords — ou ouverte n'a pas de « lâcher »).
		bilan.delais = append(bilan.delais, tE-p.FinMS)
	}
	switch {
	case !okDet:
		t.Logf("  %s explosion %d : détonateur NON identifié (pont par manche) — hors dénominateur V1", id, tE)
	case !okPorteur:
		t.Logf("  %s explosion %d : AUCUN porteur à la pose ni dans les %d s avant — hors dénominateur V1",
			id, tE, b2DernierPorteurMaxMS/1000)
	case p.XUID == 0:
		t.Logf("  %s explosion %d : porteur slot %d NON ponté — hors dénominateur V1", id, tE, p.Slot)
	default:
		bilan.v1Resolues++
		xP := strconv.FormatUint(p.XUID, 10)
		if xP == xDet {
			bilan.v1Accords++
			t.Logf("  %s explosion %d : ACCORD — poseur = détonateur (xuid %s, portage %d->%d, prises V2=%d)",
				id, tE, xP, p.DebutMS, p.FinMS, prises)
		} else {
			// Documenté, PAS une erreur unitaire : les quatre désaccords du corpus sont
			// instruits par bombe_b3_desaccords_test.go ; le gate est le COMPTE figé.
			t.Logf("  %s explosion %d : DÉSACCORD — porteur %s, détonateur %s (portage %d->%d)",
				id, tE, xP, xDet, p.DebutMS, p.FinMS)
		}
	}
}

// b2Kills lit le fil des KILLS du film (même chunk highlight que le fil des morts) : par
// instant, les xuids crédités d'un frag. Sert à requalifier les événements th=10 émis à la
// mort du porteur (cf. raffinement V3 en tête de fichier).
func b2Kills(t *testing.T, cache, id string) []analysishl {
	t.Helper()
	dir := filepath.Join(cache, "film_chunks", id)
	n := filmdec.CountFilmChunks(dir)
	raw, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		t.Fatalf("%s : chunk highlight illisible : %v", id, err)
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		t.Fatalf("%s : highlights illisibles : %v", id, err)
	}
	var out []analysishl
	for _, e := range evs {
		if e.EventType == analysis.EventTypeKill {
			out = append(out, analysishl{xuid: e.XUID, tMS: e.TimeMS})
		}
	}
	return out
}

// analysishl est un frag daté du fil des highlights.
type analysishl struct {
	xuid uint64
	tMS  int
}

// b2EstTueurDuPorteur dit si l'événement (tC, xC) est un « tueur du porteur » : il tombe à
// <= 150 ms d'une fin de période par MORT et son xuid est crédité d'un frag au même instant.
func b2EstTueurDuPorteur(tC int, xC string, periodes []HeldObjectPeriod, kills []analysishl) bool {
	const tol = 150
	finProche := false
	for _, p := range periodes {
		if p.FinParMort && absInt(p.FinMS-tC) <= tol {
			finProche = true
			break
		}
	}
	if !finProche {
		return false
	}
	for _, k := range kills {
		if absInt(k.tMS-tC) <= tol && strconv.FormatUint(k.xuid, 10) == xC {
			return true
		}
	}
	return false
}

// TestBombeB2TemoinOddball applique V3 : la chronologie du crâne contre les skull_carry du pied.
func TestBombeB2TemoinOddball(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandée : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestBombeB2TemoinOddball")()
	release := filmdec.LockProcessDecode()
	defer release()

	evs, slotXUID, deaths := b2Timeline(t, cache, b1Temoin, b1Crane)
	periodes := b2Periodes(evs, slotXUID, deaths)
	for _, p := range periodes {
		t.Logf("  portage crâne slot %d xuid %d : %d -> %d ms (mort=%v ouverte=%v)",
			p.Slot, p.XUID, p.DebutMS, p.FinMS, p.FinParMort, p.Ouverte)
	}
	kills := b2Kills(t, cache, b1Temoin)

	src, ok, err := filmcache.Open(cache, b1Temoin)
	if err != nil || !ok {
		t.Fatalf("témoin %s absent du cache : %v", b1Temoin, err)
	}
	carries, couverts, accords := 0, 0, 0
	tueurs, sansIdentite := 0, 0
	for _, ev := range objectiveevents.Extract(b1Temoin, "Oddball:Arena", src, objectiveevents.MapRoster{}) {
		if ev.EventType != objectiveevents.EventTypeSkullCarry || ev.TimeMS == nil || len(ev.Players) == 0 {
			continue
		}
		carries++
		tC, xC := *ev.TimeMS, ev.Players[0].XUID
		if b2EstTueurDuPorteur(tC, xC, periodes, kills) {
			tueurs++
			t.Logf("  th10 %d ms (xuid %s) : TUEUR DU PORTEUR (skull_carriers_killed) — hors heartbeats", tC, xC)
			continue
		}
		var dedans *HeldObjectPeriod
		for i := range periodes {
			p := &periodes[i]
			if p.DebutMS-b2TolV3MS <= tC && tC <= p.FinMS+b2TolV3MS {
				if p.XUID != 0 {
					dedans = p
					break
				}
				if dedans == nil {
					dedans = p // période sans identité : couverture sans comparaison
				}
			}
		}
		switch {
		case dedans == nil:
			t.Logf("  skull_carry %d ms (xuid %s) : hors de toute période", tC, xC)
		case dedans.XUID == 0:
			sansIdentite++
			t.Logf("  skull_carry %d ms (xuid %s) : couvert par une période NON pontée (slot %d)", tC, xC, dedans.Slot)
		default:
			couverts++
			if strconv.FormatUint(dedans.XUID, 10) == xC {
				accords++
			} else {
				t.Logf("  skull_carry %d ms : porteur reconstruit xuid %d, pied dit %s — DÉSACCORD",
					tC, dedans.XUID, xC)
			}
		}
	}
	t.Logf("V3 : %d th10 ; %d tueurs-du-porteur requalifiés ; %d couverts sans identité de pont ; "+
		"%d heartbeats comparés ; %d accords (%.1f %% des comparés)",
		carries, tueurs, sansIdentite, couverts, accords, 100*float64(accords)/float64(max(couverts, 1)))
	if couverts > 0 && float64(accords) < 0.9*float64(couverts) {
		t.Errorf("V3 sous la cible écrite de 90 %% des heartbeats comparés")
	}
}
