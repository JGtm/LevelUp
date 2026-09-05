package filmdec

// vehicules_v13_deadstate_test.go — MESURE V13 : le dead-state `i11` des vehicules, lu par LA
// MARCHE et non par l'ancre. Instrument de recherche, LECTURE SEULE, garde par V13_FILMS.
//
// LE GATE EST ECRIT AVANT LA MESURE : `.ai/V7.5/GATE_V13_DEADSTATE_MARCHE_2026-09-05.md`.
// Resume des criteres, tous implementes ici :
//
//	G1a  chiffre ce que le FILTRE DE BANDE aurait perdu (dead-states ti=35 hors de la bande
//	     derivee des images-cles). Rapport, pas assertion : la marche range par ARCHETYPE, donc
//	     un dead-state hors bande est compte quand meme. Redefini apres la premiere mesure —
//	     ecrit d'abord comme une assertion, il a echoue en disant vrai (la bande EST incomplete,
//	     le film lie des entites par records NEW en cours de flux) ;
//	G1b  la marche doit trouver STRICTEMENT PLUS de dead-states `ti=35` que l'ancre
//	     (`ScanFilmBipedPositions` + recordMaskHook, le temoin du lot V10) — sinon elle
//	     n'apporte rien et la mesure est nulle ;
//	G2   sur un film ou AUCUNE entite `ti=40` n'est declaree, le compte vehicule doit etre 0 ;
//	G3   aucun compte ne se publie sans sa couverture (records propres / records marches, et
//	     slots atteints / slots declares), pour `ti=35` ET `ti=40`.
//
// REGLE DE LECTURE POSEE D'AVANCE : un compte faible avec une couverture faible ne conclut PAS a
// l'absence — il conclut « toujours sous-instrumente ».
//
// USAGE (depuis apps/go-api, cache Go isole, UN FILM PAR PROCESS de preference) :
//
//	CGO_ENABLED=0 V13_FILM_ROOT=<repo>/data/cache V13_FILMS=0d76e8f1 \
//	  go test ./internal/analysis/filmdec/ -run '^TestV13DeadStateMarche$' -v -timeout 120m

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// v13MaxChunks : bornage par defaut du nombre de chunks marches (cout, pas correction).
const v13MaxChunks = 16

// TestV13DeadStateMarche est LA mesure du lot.
func TestV13DeadStateMarche(t *testing.T) {
	films := v13ParseFilms(t)
	root := v13Root()

	release := LockProcessDecode()
	defer release()

	for _, short8 := range films {
		t.Run(short8, func(t *testing.T) { v13MeasureFilm(t, root, short8) })
	}
}

// v13MeasureFilm : la passe complete sur un film.
func v13MeasureFilm(t *testing.T, root, short8 string) {
	dir := v13Dir(root, short8)
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("%s : chunk_00 illisible : %v", short8, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("%s : registre illisible : %v", short8, err)
	}
	n := CountFilmChunks(dir)
	if m := v13MaxChunksEnv(); n > m {
		n = m
	}

	kfs, deltas := v13LoadPackets(t, dir, n)
	if len(deltas) == 0 {
		t.Fatalf("%s : aucun paquet delta", short8)
	}

	// --- Cadre de la boucle de records : RUNTIME, donc BALAYE et PUBLIE (jamais suppose).
	cfg := v13Calibrate(t, reg, kfs, deltas)

	bipSlots, bipLo, bipHi := v13DeclaredSlots(kfs, v13BipedTI)
	vehSlots, vehLo, vehHi := v13DeclaredSlots(kfs, VehicleTypeIndex)
	t.Logf("V13 %s — %d chunks · cadre retenu : idLow=%d amorce=%d extra=%v", short8, n,
		cfg.IDLowBits, cfg.PacketPreambleBits, cfg.HasExtraFields)
	t.Logf("  images-cles : %d slots ti=35 declares (bande %d..%d) · %d slots ti=40 declares (bande %d..%d)",
		len(bipSlots), bipLo, bipHi, len(vehSlots), vehLo, vehHi)

	// --- LA MARCHE.
	tl := newV13Timeline(reg, kfs)
	st := newV13Stats(reg)
	var deads []v13Dead
	for _, d := range deltas {
		w := tl.advanceTo(d.ts)
		st.pkTotal++
		start := cfg.PacketPreambleBits
		if v13HasEvents(d.pay) {
			st.pkEvents++
			s := v13Locate(d.pay, w, cfg)
			if s < 0 {
				continue // paquet a events non localise : pas de point de depart sur.
			}
			st.pkLocated++
			start = s
		}
		deads = v13HarvestPacket(d.pay, w, cfg, start, d.ts, st, deads)
	}
	sort.Slice(deads, func(i, j int) bool { return deads[i].ts < deads[j].ts })
	deads = v13Dedup(deads)

	// --- G3 : la couverture, publiee AVANT tout compte.
	v13ReportCoverage(t, st, bipSlots, vehSlots)

	// --- Les comptes, par archetype.
	byTI := map[uint32][]v13Dead{}
	for _, d := range deads {
		byTI[d.ti] = append(byTI[d.ti], d)
	}
	tis := make([]int, 0, len(byTI))
	for ti := range byTI {
		tis = append(tis, int(ti))
	}
	sort.Ints(tis)
	strict, tail := 0, 0
	for _, d := range deads {
		if d.tailDesync {
			tail++
		} else {
			strict++
		}
	}
	t.Logf("  DEAD-STATES (Mort=1, dedupliques) : %d au total — %d STRICTS (record entierement porte) "+
		"+ %d a QUEUE INCONNUE (rupture APRES le dead-state : bits du dead-state lus au bon endroit, "+
		"seule la fin du record est non modelisee)", len(deads), strict, tail)
	for _, ti := range tis {
		// Le taux dead-states / records propres du MEME archetype est le controle decisif : il dit
		// si « peu de records » empeche vraiment de voir une mort. Un archetype a faible rendement
		// qui rend quand meme beaucoup de dead-states demontre que le rendement n'est pas le verrou.
		u := uint32(ti)
		t.Logf("    archetype ti=%-3d : %3d dead-states · records propres %6d (%6d marches) · "+
			"taux %.4f · %s", ti, len(byTI[u]), st.recClean[u], st.recTotal[u],
			float64(len(byTI[u]))/float64(max(st.recClean[u], 1)), v13ArchSignature(reg, ti))
	}

	v13ReportMask(t, st)
	v13GateG1a(t, byTI[v13BipedTI], bipLo, bipHi)
	v13GateG1b(t, dir, short8, len(byTI[v13BipedTI]))
	v13GateG2(t, len(vehSlots), len(byTI[VehicleTypeIndex]))
	v13ReportPower(t, st, len(byTI[v13BipedTI]), len(byTI[VehicleTypeIndex]))
	v13ReportVehicles(t, byTI[VehicleTypeIndex], byTI[v13BipedTI], vehLo, vehHi)
}

// v13ReportCoverage publie les denominateurs du gate G3.
func v13ReportCoverage(t *testing.T, st *v13Stats, bipSlots, vehSlots map[int]bool) {
	t.Helper()
	allTot, allClean := 0, 0
	for ti, n := range st.recTotal {
		allTot += n
		allClean += st.recClean[ti]
	}
	t.Logf("  [G3 COUVERTURE] paquets delta marches %d · a events %d · localises %d (%.1f %% des paquets a events)",
		st.pkTotal, st.pkEvents, st.pkLocated, v13Pct(st.pkLocated, st.pkEvents))
	t.Logf("  [G3 COUVERTURE] TOUS archetypes : %d records marches · %d propres (%.1f %%) · %d archetypes distincts",
		allTot, allClean, v13Pct(allClean, allTot), len(st.recTotal))
	for _, ti := range []uint32{v13BipedTI, VehicleTypeIndex} {
		decl := bipSlots
		if ti == VehicleTypeIndex {
			decl = vehSlots
		}
		// NB : « slots atteints » peut DEPASSER « declares aux images-cles » — le film cree aussi
		// des entites par records NEW en cours de flux. Ce n'est donc pas un ratio de couverture,
		// mais deux comptes a lire ensemble ; la couverture, c'est la part de records PROPRES.
		t.Logf("  [G3 COUVERTURE] ti=%-3d records marches %6d · propres %6d (%.1f %%) · slots atteints %d (images-cles : %d declares)",
			ti, st.recTotal[ti], st.recClean[ti], v13Pct(st.recClean[ti], st.recTotal[ti]),
			len(st.slotsSeen[ti]), len(decl))
	}
}

// v13ReportMask est LE controle qui separe « le vehicule ne meurt pas dans le film » de « ses
// morts sont dans la fraction desynchronisee qu'on jette ». Il compte, par archetype, les records
// dont le MASQUE DECLARE le composant dead-state — que la traversee ait desynchronise ou non. Le
// masque se lit AVANT toute consommation de corps : il est insensible a une erreur de grammaire
// en aval. Si aucun record ti=40 ne declare jamais le composant, la desynchronisation n'explique
// rien et l'absence est REELLE.
func v13ReportMask(t *testing.T, st *v13Stats) {
	t.Helper()
	tis := make([]int, 0, len(st.recTotal))
	for ti := range st.recTotal {
		tis = append(tis, int(ti))
	}
	sort.Ints(tis)
	for _, ti := range tis {
		u := uint32(ti)
		if st.deadStateIndex(u) < 0 || st.recTotal[u] < 50 {
			continue // archetype sans composant dead-state, ou trop peu marche pour dire quoi que ce soit
		}
		t.Logf("  [MASQUE i-dead] ti=%-3d : %4d / %6d records DECLARENT le dead-state (%.2f %%) · "+
			"dont %d dans des records desynchronises", ti, st.maskDead[u], st.recTotal[u],
			v13Pct(st.maskDead[u], st.recTotal[u]), st.maskDeadDirty[u])
		if st.maskDeadDirty[u]*2 < st.maskDead[u] {
			continue // la majorite se lit : rien a diagnostiquer sur cet archetype
		}
		v13ReportDesync(t, st, u)
	}
}

// v13ReportDesync nomme le point de rupture : l'index ou la traversee s'arrete, avec le nom du
// composant que le registre place a cet index. C'est la cible du correctif.
func v13ReportDesync(t *testing.T, st *v13Stats, ti uint32) {
	t.Helper()
	a, ok := st.reg.Archetype(int(ti))
	if !ok {
		return
	}
	type kv struct{ at, n int }
	var hs []kv
	for at, n := range st.desyncAt[ti] {
		hs = append(hs, kv{at, n})
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i].n > hs[j].n })
	for i, h := range hs {
		if i >= 5 {
			t.Logf("      ... et %d autres positions de rupture", len(hs)-5)
			break
		}
		name := "index hors registre"
		if h.at >= 0 && h.at < len(a.Components) {
			name = a.Components[h.at]
		}
		t.Logf("      rupture a DesyncAt=%-3d dans %4d records · composant a cet index : %s",
			h.at, h.n, name)
	}
	// Le masque le plus frequent : la liste des composants presents, pour lire la sequence que la
	// traversee doit consommer avant d'atteindre le dead-state.
	var bm uint64
	bn := 0
	for m, n := range st.desyncMask[ti] {
		if n > bn {
			bm, bn = m, n
		}
	}
	var idx []string
	for i := 0; i < 64 && i < len(a.Components); i++ {
		if bm&(1<<uint(i)) != 0 {
			idx = append(idx, fmt.Sprintf("i%d:%s", i, a.Components[i]))
		}
	}
	t.Logf("      masque le plus frequent (%d records) : %s", bn, strings.Join(idx, " | "))
}

// v13ArchSignature nomme un archetype par ce que le registre en dit : nombre de composants et
// familles de prefixes. C'est la seule facon de savoir si un `ti` porteur de dead-states est un
// cousin du vehicule (chassis enfant, tourelle) ou tout autre chose.
func v13ArchSignature(reg *Registry, ti int) string {
	a, ok := reg.Archetype(ti)
	if !ok {
		return "archetype absent du registre"
	}
	seen := map[string]int{}
	var order []string
	for _, c := range a.Components {
		fam := c
		if i := strings.Index(c, "-"); i > 0 {
			fam = c[:i]
		}
		if seen[fam] == 0 {
			order = append(order, fam)
		}
		seen[fam]++
	}
	parts := make([]string, 0, len(order))
	for _, f := range order {
		parts = append(parts, fmt.Sprintf("%s:%d", f, seen[f]))
	}
	return fmt.Sprintf("%d composants [%s]", len(a.Components), strings.Join(parts, " "))
}

// v13GateG1a mesure ce que le FILTRE DE BANDE aurait perdu. Ce n'est PAS un echec d'instrument :
// la marche range par ARCHETYPE (`FrameRecord.TypeIndex`), pas par bande, donc un dead-state
// ti=35 hors bande est compte quand meme. Le chiffre dit seulement que la bande derivee des
// images-cles est incomplete — le film lie aussi des entites par records NEW en cours de flux.
// C'est un argument DE PLUS contre le filtre de bande, pas contre la mesure.
func v13GateG1a(t *testing.T, biped []v13Dead, lo, hi int) {
	t.Helper()
	hors := 0
	for _, d := range biped {
		if int(d.slot) < lo || int(d.slot) > hi {
			hors++
		}
	}
	if hors > 0 {
		t.Logf("  [G1a] %d des %d dead-states ti=35 tombent HORS de la bande derivee %d..%d : le filtre "+
			"de bande des images-cles les aurait PERDUS. La marche les garde (tri par archetype).",
			hors, len(biped), lo, hi)
		return
	}
	t.Logf("  [G1a] les %d dead-states ti=35 tombent tous dans la bande derivee %d..%d", len(biped), lo, hi)
}

// v13GateG1b : la marche doit battre l'ancre sur le bipede, sinon elle n'apporte rien.
func v13GateG1b(t *testing.T, dir, short8 string, marche int) {
	t.Helper()
	if os.Getenv("V13_NOANCRE") != "" {
		t.Logf("  [G1b NON EVALUE] V13_NOANCRE pose : le temoin de l'ancre n'a pas tourne")
		return
	}
	nRec, nI11 := v13AnchorI11(t, dir)
	if nRec < 0 {
		t.Logf("  [G1b NON EVALUE] balayage ancre impossible sur %s", short8)
		return
	}
	t.Logf("  [G1b] ancre (ScanFilmBipedPositions) : %d records ti=35 acceptes, %d portent i11 · marche : %d dead-states ti=35",
		nRec, nI11, marche)
	if marche <= nI11 {
		t.Errorf("[G1b ECHEC] la marche (%d) ne bat pas l'ancre (%d) sur le bipede : elle n'apporte "+
			"rien, la mesure vehicule est nulle", marche, nI11)
		return
	}
	t.Logf("  [G1b OK] la marche bat l'ancre sur le bipede (%d > %d)", marche, nI11)
}

// v13GateG2 : temoin negatif — pas de vehicule declare, pas de dead-state vehicule.
func v13GateG2(t *testing.T, declared, found int) {
	t.Helper()
	if declared > 0 {
		t.Logf("  [G2 NON EVALUE sur ce film] %d slots ti=40 declares — le temoin negatif exige un film "+
			"SANS vehicule", declared)
		return
	}
	if found != 0 {
		t.Errorf("[G2 ECHEC] aucun slot ti=40 declare mais %d dead-states vehicule recoltes : "+
			"l'instrument fabrique des vehicules", found)
		return
	}
	t.Logf("  [G2 OK] film sans vehicule : 0 dead-state en archetype 40")
}

// v13ReportVehicles detaille les dead-states vehicule et chiffre la resolution du champ tueur,
// COMPAREE au meme taux chez le bipede — c'est le critere de decision du gate.
// v13ReportPower est LE garde-fou de lecture du lot : il chiffre COMBIEN de dead-states vehicule
// la mesure pourrait voir SI un vehicule en gravait au meme taux qu'un bipede. Sans ce chiffre,
// « 0 trouve » se lit a tort comme « ca n'existe pas », alors que la marche ne rend que ~1 000
// records ti=40 propres par film contre ~50 000 pour le bipede : l'attendu est de l'ordre de 1.
func v13ReportPower(t *testing.T, st *v13Stats, bipDead, vehDead int) {
	t.Helper()
	bipClean, vehClean := st.recClean[v13BipedTI], st.recClean[VehicleTypeIndex]
	if bipClean == 0 || bipDead == 0 {
		t.Logf("  [PUISSANCE] indeterminable : aucun dead-state bipede de reference")
		return
	}
	rate := float64(bipDead) / float64(bipClean)
	exp := rate * float64(vehClean)
	t.Logf("  [PUISSANCE] taux bipede %.5f dead-state/record propre (%d/%d) · records ti=40 propres %d "+
		"=> ATTENDU au MEME taux : %.2f · OBSERVE : %d", rate, bipDead, bipClean, vehClean, exp, vehDead)
	if exp < 5 {
		t.Logf("  [PUISSANCE] attendu < 5 : mesure SOUS-PUISSANTE — elle ne distingue PAS « le vehicule " +
			"ne grave jamais de dead-state » de « il en grave, et on n'avait pas de quoi le voir ». Le " +
			"verrou n'est plus l'ancre : c'est le RENDEMENT en records ti=40 de la marche.")
	}
}

func v13ReportVehicles(t *testing.T, veh, biped []v13Dead, vehLo, vehHi int) {
	t.Helper()
	if len(veh) == 0 {
		t.Logf("  RESULTAT : 0 dead-state en archetype 40. Se lire AVEC la couverture ti=40 ET la " +
			"puissance ci-dessus.")
		return
	}
	t.Logf("  RESULTAT : %d dead-states en archetype 40 (VEHICULE)", len(veh))
	for i, d := range veh {
		if i >= 40 {
			t.Logf("    ... et %d autres", len(veh)-40)
			break
		}
		bande := "DANS la bande vehicule"
		if int(d.slot) < vehLo || int(d.slot) > vehHi {
			bande = fmt.Sprintf("HORS bande vehicule %d..%d — SUSPECT (slot lie a ti=40 en cours de "+
				"flux : candidat artefact, pas un vehicule du monde)", vehLo, vehHi)
		}
		t.Logf("    t=%9d us · slot %4d · EnumA=%3d EnumB=%3d cat=%d hasRef=%v gid=0x%08x · %s",
			d.ts, d.slot, d.enumA, d.enumB, d.val0c, d.hasRef, d.gid, bande)
	}
	vOK, vN := v13KillerResolved(veh)
	bOK, bN := v13KillerResolved(biped)
	t.Logf("  CHAMP TUEUR (EnumB >= 0) : vehicule %d/%d (%.1f %%) · bipede %d/%d (%.1f %%)",
		vOK, vN, v13Pct(vOK, vN), bOK, bN, v13Pct(bOK, bN))
	t.Logf("  LECTURE : « exploitable » exige un taux vehicule DU MEME ORDRE que le bipede ; sinon " +
		"la datation seule est publiable (end/tEnd), pas l'attribution.")
}

// v13KillerResolved compte les dead-states dont le champ tueur est renseigne.
func v13KillerResolved(ds []v13Dead) (ok, n int) {
	for _, d := range ds {
		n++
		if d.enumB >= 0 {
			ok++
		}
	}
	return ok, n
}

// v13AnchorI11 rejoue le temoin du lot V10 : combien de records ti=35 l'ANCRE accepte, et
// combien portent i11. Rend (-1, -1) si le balayage echoue.
func v13AnchorI11(t *testing.T, dir string) (nRec, nI11 int) {
	t.Helper()
	prev := recordMaskHook
	SetRecordMaskHook(func(idx []int, _ []byte, _ int) {
		nRec++
		for _, id := range idx {
			if id == 11 {
				nI11++
			}
		}
	})
	_, err := ScanFilmBipedPositions(dir, ScanFilmOptions{RequireTag1: true, DropSaturated: true,
		CaptureDirs: true, QuantaOnly: true})
	SetRecordMaskHook(prev)
	if err != nil {
		return -1, -1
	}
	return nRec, nI11
}

// v13Delta : un paquet delta avec son horodatage et son payload.
type v13Delta struct {
	ts  uint64
	pay []byte
}

// v13LoadPackets charge les n chunks et rend les images-cles decodees + les paquets delta,
// tries par horodatage (le curseur de la timeline exige des appels croissants).
func v13LoadPackets(t *testing.T, dir string, n int) ([]v13KF, []v13Delta) {
	t.Helper()
	var kfs []v13KF
	var deltas []v13Delta
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			switch pk.Type {
			case PacketTypeKeyframe:
				kfs = append(kfs, v13KF{ts: pk.TimestampUS, recs: WalkKeyframeWorld(pk.Payload(data))})
			case PacketTypeDelta:
				deltas = append(deltas, v13Delta{ts: pk.TimestampUS, pay: pk.Payload(data)})
			}
		}
	}
	sort.Slice(deltas, func(i, j int) bool { return deltas[i].ts < deltas[j].ts })
	return kfs, deltas
}

// v13Calibrate fixe le CADRE de la boucle de records (idLow / amorce / prologue de mode film).
// Ces trois grandeurs sont du RUNTIME : elles ne sont PAS dans le film, il faut les balayer et
// publier la retenue (cf. BestVariant). On vote sur les premiers paquets SANS liste d'evenements
// — ceux dont la boucle commence a l'amorce, donc les seuls que BestVariant sait juger.
func v13Calibrate(t *testing.T, reg *Registry, kfs []v13KF, deltas []v13Delta) FrameConfig {
	t.Helper()
	if v := os.Getenv("V13_IDLOW"); v != "" {
		cfg := DefaultFrameConfig()
		if k, err := strconv.Atoi(v); err == nil {
			cfg.IDLowBits = k
		}
		if p := os.Getenv("V13_PREAMBLE"); p != "" {
			if k, err := strconv.Atoi(p); err == nil {
				cfg.PacketPreambleBits = k
			}
		}
		t.Logf("  [CADRE] FORCE par l'environnement : idLow=%d amorce=%d", cfg.IDLowBits, cfg.PacketPreambleBits)
		return cfg
	}
	// Le critere de selection est LA MARCHE ELLE-MEME, sur un echantillon : nombre de records
	// PROPRES recoltes. `BestVariant` (bits consommes sur UN paquet, monde neuf) a ete essaye
	// et REJETE : il designe idLow=10 sur `0d76e8f1` la ou la marche complete donne 38 % de
	// paquets a events localises contre 97,4 % a idLow=13. Un cadre faux consomme des bits sans
	// rien decoder de vrai ; seul le rendement en records propres le demasque.
	best := v13Score{located: -1, clean: -1}
	for pre := 0; pre <= 2; pre++ {
		for low := 10; low <= 15; low++ {
			cfg := DefaultFrameConfig()
			cfg.IDLowBits, cfg.PacketPreambleBits = low, pre
			s := v13Trial(reg, kfs, deltas, cfg, v13CalibPackets, v13CalibEvents)
			if s.located > best.located || (s.located == best.located && s.clean > best.clean) {
				best = s
			}
		}
	}
	t.Logf("  [CADRE] balayage 18 cadres : retenu idLow=%d amorce=%d "+
		"(%d/%d paquets a events LOCALISES · %d records propres)",
		best.cfg.IDLowBits, best.cfg.PacketPreambleBits, best.located, best.events, best.clean)
	return best.cfg
}

// v13CalibPackets / v13CalibEvents bornent l'echantillon de calibrage. Le budget est exprime EN
// PAQUETS A EVENEMENTS parce que le critere de selection est le TAUX DE LOCALISATION : la
// signature du slot 123 (delta de 35 bits exactement) est un oracle fort du cadre, la ou le seul
// rendement en records propres se laisse berner par un cadre absurde qui consomme des bits sans
// rien decoder (mesure : idLow=10 rend 492 records « propres » et 0/12 paquets localises, quand
// idLow=13 localise 97,4 % des paquets a events).
const (
	v13CalibPackets = 3000
	v13CalibEvents  = 150
)

// v13Score : le rendement d'un cadre candidat sur l'echantillon de calibrage.
type v13Score struct {
	cfg     FrameConfig
	clean   int
	located int
	events  int
}

// v13Trial marche `n` paquets sous un cadre donne et rend son rendement. Monde neuf a chaque
// essai : le balayage ne doit rien emporter d'un candidat a l'autre.
func v13Trial(reg *Registry, kfs []v13KF, deltas []v13Delta, cfg FrameConfig, maxPk, maxEv int) (out v13Score) {
	out.cfg = cfg
	tl := newV13Timeline(reg, kfs)
	st := newV13Stats(reg)
	var sink []v13Dead
	for i, d := range deltas {
		if i >= maxPk || out.events >= maxEv {
			break
		}
		w := tl.advanceTo(d.ts)
		start := cfg.PacketPreambleBits
		if v13HasEvents(d.pay) {
			out.events++
			s := v13Locate(d.pay, w, cfg)
			if s < 0 {
				continue
			}
			out.located++
			start = s
		}
		sink = v13HarvestPacket(d.pay, w, cfg, start, d.ts, st, sink)
	}
	for ti := range st.recClean {
		out.clean += st.recClean[ti]
	}
	return out
}

func v13Pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func v13ParseFilms(t *testing.T) []string {
	raw := os.Getenv("V13_FILMS")
	if raw == "" {
		t.Skipf("V13_FILMS absent : mesure V13 sautee")
	}
	var out []string
	for _, tok := range strings.Split(raw, ",") {
		if tok = strings.TrimSpace(tok); tok != "" {
			out = append(out, tok)
		}
	}
	if len(out) == 0 {
		t.Skipf("V13_FILMS vide")
	}
	return out
}

func v13Root() string {
	if r := os.Getenv("V13_FILM_ROOT"); r != "" {
		return r
	}
	return `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache`
}

func v13Dir(root, short8 string) string {
	return fmt.Sprintf("%s%c%s%c%s", strings.TrimRight(root, `\/`), os.PathSeparator,
		"film_chunks", os.PathSeparator, short8)
}

func v13MaxChunksEnv() int {
	if v := os.Getenv("V13_MAXCHUNKS"); v != "" {
		if k, err := strconv.Atoi(v); err == nil && k > 0 {
			return k
		}
	}
	return v13MaxChunks
}
