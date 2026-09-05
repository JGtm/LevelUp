package replay

// vehicules_v2b_occupant_test.go — INSTRUMENT DE MESURE (lot V2b, signal 1 : OCCUPANT par
// l'ATTACHEMENT i10). LECTURE SEULE, garde par V2B_OCC_ROOT.
//
// L'HYPOTHESE UTILISATEUR : un bipede (ti=35) qui MONTE dans un vehicule devient ENFANT du
// vehicule -> son composant object-parent-state (i10) reference l'entite VEHICULE (ti=40). Lire i10
// sur les BIPEDES donnerait donc l'occupant DIRECTEMENT.
//
// CAVEAT CONNU (PLAN_ATTACHEMENT_PARENT_STATE.md, GATE 0 NEGATIF du 2026-08-18) : pour le DRAPEAU,
// i10 n'a designe aucune entite vivante au-dessus du hasard, et le bit de porte est ouvert ~1/3
// UNIFORMEMENT sur les neuf archetypes (signature d'un bit qui n'est pas la porte qu'on croit). La
// condition de reprise #1, JAMAIS TENTEE, est un balayage de param_4 par archetype (SetRecordStateParam)
// — la largeur d'i10 branche dessus, et le seul releve (capture CE) porte sur le bipede sans i10, donc
// i10 retombe sur le defaut 1. CET INSTRUMENT LA TENTE.
//
// CE QUI EST REUTILISE, sans copie : attScanI10 (attachement_phase0_socle_test.go) — la marche
// stateful DecodeFrameViews + la sonde SetObjectParentStateHook + la resolution du parent dans le
// World (ParentTI, ParentLie) + le temoin decorrele (TemoinLie). Et filmdec.ScanFilmVehicleEvents
// (event_list.go, NON MODIFIE) : le RELAIS board/exit qui resout l'occupant a la ms.
//
// LES SEUILS, ECRITS AVANT LA MESURE.
//   - i10 designe l'occupant SI, parmi les lectures i10 ATTACHEES sur des records ti=35, une part
//     nette resout vers un slot ti=40 (ParentTI==40), NETTEMENT au-dessus du temoin (part des
//     lectures dont le slot DECORRELE resout vers du VIVANT) et au-dessus du taux de base des ti=40.
//     Gate : part(ParentTI==40) - part(temoin vivant) > 10 points. Sinon i10 est refute pour ti=35.
//   - Le balayage param_4 {defaut,0,2,3,4} : une valeur qui RE-SYNCHRONISE i10 doit AUGMENTER
//     conjointement (a) la part de records propres et (b) la resolution vers ti=40. Sinon aucune
//     valeur ne sauve i10.
//   - RECOUPEMENT V1a.4 (le RELAIS, pas i10) : l'occupant d'une SORTIE (exit) doit coincider avec
//     la FERMETURE d'un trou du flux de position de son slot (l'enfant re-emet sa position en
//     descendant). Temoin : instant de sortie decale.
//
// UN SEUL decodage filmdec par process (attScanI10 prend le verrou).
//
// USAGE (depuis apps/go-api, cache Go ISOLE) :
//
//	$env:GOCACHE='<scratch>\gocache_v2b'
//	CGO_ENABLED=0 V2B_OCC_ROOT=<repo>/data/cache V2B_OCC_FILM=0d76e8f1 \
//	  go test ./internal/analysis/replay/ -run '^TestV2bOccupantI10$' -v -timeout 60m

import (
	"os"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	v2boBipedTI = 35
	v2boVehiTI  = 40
	v2boGapMS   = 3000 // trou minimal du flux de position (V1a.4)
	v2boTolMS   = 2000 // tolerance d'appariement sortie <-> fermeture de trou
	v2boShiftMS = 37000
)

func TestV2bOccupantI10(t *testing.T) {
	dir := v2boDir(t)

	// --- 1) i10 sur les bipedes : defaut puis balayage param_4 ---
	t.Logf("############## SIGNAL 1 — i10 SUR LES BIPEDES (ti=35 -> ti=40 ?) ##############")
	params := []int{-1, 0, 2, 3, 4} // -1 = defaut (pas d'override, param_4=1 pour i10)
	for _, pv := range params {
		if pv >= 0 {
			filmdec.SetRecordStateParam(uint32(pv))
		}
		reads, st, err := attScanI10(dir)
		if err != nil {
			t.Fatalf("balayage i10 (param=%d) : %v", pv, err)
		}
		v2boReportI10(t, pv, reads, st)
	}
	filmdec.SetRecordStateParam(1) // remise a la valeur par defaut effective

	// --- 2) le RELAIS : event-list board/exit ---
	t.Logf("\n############## RELAIS — event-list board/exit (occupant a la ms) ##############")
	evs, err := filmdec.ScanFilmVehicleEvents(dir)
	if err != nil {
		t.Fatalf("event-list : %v", err)
	}
	v2boReportRelay(t, dir, evs)
}

// v2boReportI10 juge une passe (defaut ou param donne) : lectures ti=35 attachees, resolution
// vers ti=40, temoin, et proprete globale.
func v2boReportI10(t *testing.T, pv int, reads []attI10, st attStat) {
	label := "defaut(1)"
	if pv >= 0 {
		label = "param_4=" + strconv.Itoa(pv)
	}
	var n35, attached35, resolve40, parentLive, witnessLive int
	tiHist := map[uint32]int{}
	for _, r := range reads {
		if r.TI != v2boBipedTI {
			continue
		}
		n35++
		if !r.St.Attached {
			continue
		}
		attached35++
		if r.ParentLie {
			parentLive++
			tiHist[r.ParentTI]++
			if r.ParentTI == v2boVehiTI {
				resolve40++
			}
		}
		if r.TemoinLie {
			witnessLive++
		}
	}
	propre := 100 * attPart(st.RecordsPropres, st.Records)
	t.Logf("  [%s] ti=35 : %d lectures i10 (%d attachees) · records propres %.1f %% (%d/%d)",
		label, n35, attached35, propre, st.RecordsPropres, st.Records)
	if attached35 == 0 {
		t.Logf("    aucune lecture i10 attachee sur ti=35 — i10 n'est pas emis a bord (denominateur nul)")
		return
	}
	pr40 := 100 * attPart(resolve40, attached35)
	pLive := 100 * attPart(parentLive, attached35)
	pWit := 100 * attPart(witnessLive, attached35)
	t.Logf("    parent resout VERS ti=40 : %d/%d = %.1f %% · parent VIVANT (tout ti) : %.1f %% · TEMOIN vivant : %.1f %%",
		resolve40, attached35, pr40, pLive, pWit)
	t.Logf("    GATE : %s (part ti=40 %.1f %% vs temoin vivant %.1f %%)", v2boVerdict(pr40 > pWit+10), pr40, pWit)
	t.Logf("    histogramme ParentTI (parents vivants) : %s", v2boTIHist(tiHist))
}

// v2boReportRelay juge le relais event-list et le recoupe avec le trou V1a.4.
func v2boReportRelay(t *testing.T, dir string, evs []filmdec.VehicleEvent) {
	var board, exit, exitInBand int
	for _, e := range evs {
		switch e.Kind {
		case filmdec.EventBipedBoardVehicle:
			board++
		case filmdec.EventUnitExitVehicle:
			exit++
			if e.OccupantInBand {
				exitInBand++
			}
		}
	}
	t.Logf("  board=%d · exit=%d (occupant en-bande %d = %.0f %%)", board, exit, exitInBand,
		100*attPart(exitInBand, exit))
	if exit == 0 {
		t.Logf("  aucune sortie decodee sur ce film — recoupement V1a.4 impossible")
		return
	}
	// RECOUPEMENT V1a.4 : l'occupant d'une sortie re-emet sa position -> un TROU se ferme a l'exit.
	tracks := v2boBipedTracks(t, dir)
	real, wit, sampled := 0, 0, 0
	for _, e := range evs {
		if e.Kind != filmdec.EventUnitExitVehicle || !e.OccupantInBand {
			continue
		}
		ts := int64(e.TimestampUS / 1000)
		tr := tracks[e.OccupantSlot]
		if len(tr) < 2 {
			continue
		}
		sampled++
		if v2boGapClosesNear(tr, ts, v2boTolMS) {
			real++
		}
		if v2boGapClosesNear(tr, ts+v2boShiftMS, v2boTolMS) {
			wit++
		}
	}
	t.Logf("  RECOUPEMENT V1a.4 (fermeture de trou a la sortie) : %d/%d = %.0f %% · TEMOIN (decale) : %d/%d = %.0f %%",
		real, sampled, 100*attPart(real, sampled), wit, sampled, 100*attPart(wit, sampled))
	t.Logf("  LECTURE : le RELAIS (exit) resout l'occupant a la ms la ou i10 echoue ; le recoupement teste le modele V1a.4.")
}

// v2boBipedTracks rend les instants (ms) de position par slot bipede (QuantaOnly, sans bornes).
func v2boBipedTracks(t *testing.T, dir string) map[uint32][]int64 {
	opt := filmdec.ScanFilmOptions{RequireTag1: true, DropSaturated: true, QuantaOnly: true}
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Logf("    positions bipede illisibles (%v) — recoupement saute", err)
		return map[uint32][]int64{}
	}
	out := map[uint32][]int64{}
	for _, p := range pos {
		out[p.Slot] = append(out[p.Slot], int64(p.TimestampUS/1000))
	}
	for s := range out {
		sort.Slice(out[s], func(i, j int) bool { return out[s][i] < out[s][j] })
	}
	return out
}

// v2boGapClosesNear dit si un trou (>= v2boGapMS) du trace se ferme a moins de tol de tMS.
func v2boGapClosesNear(tr []int64, tMS, tol int64) bool {
	for i := 1; i < len(tr); i++ {
		if tr[i]-tr[i-1] >= v2boGapMS {
			if d := tr[i] - tMS; d >= -tol && d <= tol {
				return true
			}
		}
	}
	return false
}

// ------------------------------------------------------------------ helpers

func v2boDir(t *testing.T) string {
	root := os.Getenv("V2B_OCC_ROOT")
	if root == "" {
		t.Skipf("V2B_OCC_ROOT absent : instrument occupant saute")
	}
	film := os.Getenv("V2B_OCC_FILM")
	if film == "" {
		film = "0d76e8f1"
	}
	return root + `\film_chunks\` + film
}

func v2boVerdict(pass bool) string {
	if pass {
		return "VALIDE (i10 designe le vehicule)"
	}
	return "REFUTE (i10 ne designe pas le vehicule au-dessus du temoin)"
}

func v2boTIHist(m map[uint32]int) string {
	tis := make([]uint32, 0, len(m))
	for ti := range m {
		tis = append(tis, ti)
	}
	sort.Slice(tis, func(i, j int) bool { return m[tis[i]] > m[tis[j]] })
	out := ""
	for k, ti := range tis {
		if k >= 6 {
			out += " ..."
			break
		}
		out += " ti" + strconv.Itoa(int(ti)) + ":" + strconv.Itoa(m[ti])
	}
	if out == "" {
		return "(aucun parent vivant)"
	}
	return out
}
