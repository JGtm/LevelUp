package filmdec

// equipment_creation_width_test.go — LA LARGEUR PAR FILM, et ce qui pourrait la fixer.
//
// LA QUESTION, et elle est falsifiable : la largeur du premier champ du bloc MPP varie d'un film
// à l'autre (9 bits sur les films d'arène, 6 sur d'autres — mesuré le 2026-08-17). Est-elle
// FIXÉE par une quantité structurelle du film (largeurs d'axe de la carte, nombre de slots,
// nombre de bipèdes, taille de l'archétype), comme le sont les largeurs de région ? Si oui, on
// la DÉRIVE et la calibration devient un garde-rail ; sinon, la calibration EST la source.
//
// CE QUE CET INSTRUMENT PUBLIE, par film : la largeur mesurée par l'oracle de position, le
// détail de la calibration (accords par largeur candidate, ancres, vies, chunks lus), et les
// quantités structurelles candidates. C'est le tableau de la phase 0.1 du plan — la comparaison
// se fait ENSUITE, sur les 12 lignes, pas film par film.
//
// LECTURE SEULE, gardé par EQUIP_CREATION_FILM. UN SEUL décodage filmdec par process.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 EQUIP_CREATION_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestEquipmentCreationWidth$' -timeout 60m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

func TestEquipmentCreationWidth(t *testing.T) {
	dir := os.Getenv(equipCreationFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", equipCreationFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	prev := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prev })
	SetWorldObjectPrecisionFromLayout(lay)
	prevW := CurrentMPPWidths()
	t.Cleanup(func() { SetMPPWidths(prevW) })

	n := CountFilmChunks(dir)
	band := worldObjectSlotBandDir(dir, n, EquipmentTypeIndex)
	arch, err := EquipmentArchetypeDir(dir)
	if err != nil {
		t.Fatalf("archétype ti=%d illisible : %v", EquipmentTypeIndex, err)
	}
	tracks, err := ScanFilmWorldObjects(dir, &equipCreationUnitRange, EquipmentTypeIndex)
	if err != nil {
		t.Fatalf("trajectoires ti=%d illisibles : %v", EquipmentTypeIndex, err)
	}
	spans := EquipmentLifeSpans(tracks)

	cal, ok := CalibrateMPPWidths(dir, &equipCreationUnitRange, band, spans)
	t.Logf("FILM %s", dir)
	t.Logf("   LARGEUR : %s", cal)
	t.Logf("   accords par découpage candidat (lead/index) : %s", equipWidthScores(cal.ByWidths))
	t.Logf("   STRUCTURE : axes i0 %v (somme %d) · %d chunks · %d slots ti=%d · %d slots bipède"+
		" · %d composants d'archétype · %d vies ti=%d",
		lay.AxisW, lay.AxisW[0]+lay.AxisW[1]+lay.AxisW[2], n, len(band), EquipmentTypeIndex,
		bipedSlotBandDir(dir, equipWidthChunkList(n)).Count(), len(arch.Components), len(tracks),
		EquipmentTypeIndex)
	if !ok {
		t.Logf("   VERDICT : la calibration NE TRANCHE PAS — ce film ne publierait aucune pose")
		return
	}
	SetMPPWidths(cal.Widths)
	equipWidthConfirm(t, dir, band)
	equipWidthPlacements(t, dir)
}

// equipWidthConfirm rejoue le balayage complet à la largeur retenue et publie la largeur du
// default-state effectivement consommée. C'est le CONTRÔLE de la calibration par une seconde
// grandeur : à la bonne largeur, la largeur dominante du default-state vaut equipMinDefaultState
// Bits + la largeur du premier champ (chemin minimal du déserialiseur porté).
//
// ATTENTION À LA LECTURE : cette distribution porte le BRUIT de l'ancre, qui domine sur les
// films à grande bande de slots. C'est la cohorte confirmée par l'oracle (equipWidthPlacements)
// qui est la mesure ; celle-ci sert à voir si la grammaire tient sur les records réels.
func equipWidthConfirm(t *testing.T, dir string, band map[uint32]bool) {
	t.Helper()
	cre, st, err := ScanFilmEquipmentCreationsForBand(dir, &equipCreationUnitRange, band)
	if err != nil {
		t.Fatalf("balayage à la largeur retenue impossible : %v", err)
	}
	widths := map[uint32]int{}
	ids := map[uint32]int{}
	for _, c := range cre {
		widths[uint32(c.DefaultStateBits)]++
		ids[uint32(c.MPPVal[MPPWord32])]++
	}
	w := CurrentMPPWidths()
	t.Logf("   BRUT à %s : %d ancres · %d records acceptés · %d identifiants `eqip` distincts",
		w, st.Anchors, st.Accepted, len(ids))
	t.Logf("      largeurs du default-state (attendu : %d dominant) :%s",
		equipMinDefaultStateBits+w.Lead+w.Index, equipCreationLine(widths, 10))
}

// equipWidthPlacements publie la cohorte CONFIRMÉE par l'oracle : les poses que la production
// publierait. C'est le seul chiffre qui compte pour le gate — le reste est du dénominateur.
func equipWidthPlacements(t *testing.T, dir string) {
	t.Helper()
	pl, st, err := ScanFilmEquipmentPlacements(dir, &equipCreationUnitRange)
	if err != nil {
		t.Fatalf("balayage des poses impossible : %v", err)
	}
	ids := map[uint32]int{}
	widths := map[uint32]int{}
	for _, p := range pl {
		ids[p.GlobalID]++
		widths[uint32(p.Points)]++
	}
	t.Logf("   POSES CONFIRMÉES : %d ancres · %d records acceptés · %d confirmés par l'oracle"+
		" · %d POSES · %d identifiants `eqip` distincts",
		st.Anchors, st.Accepted, st.Confirmed, st.Placements, len(ids))
	t.Logf("      identifiants `eqip` des poses :%s", equipCreationLine(ids, 24))
}

// equipMinDefaultStateBits est la largeur du default-state de ti=37 sur le CHEMIN MINIMAL
// (toutes les portes fermées), HORS les DEUX champs de largeur variable : deux préfixes de
// version (1 bit chacun), le bloc MPP sans son premier champ ni son index inline
// (32 + 1 + 1 + 1 + 2 + 3 + 1 + 1 = 42), puis les deux portes fermées du default-state de ti=37
// (1 + 1). Cf. consumeDefaultStateTI37 et consumeMultiplayerPropertiesBlock — cette constante
// n'est PAS une mesure, c'est la somme des largeurs que le déserialiseur porté consomme, et
// l'écart à la mesure est le contrôle.
const equipMinDefaultStateBits = 1 + 1 + 42 + 1 + 1

func equipWidthChunkList(n int) []int {
	out := make([]int, 0, n)
	for c := 1; c <= n; c++ {
		out = append(out, c)
	}
	return out
}

func equipWidthScores(byWidths map[MPPWidths]int) string {
	ws := make([]MPPWidths, 0, len(byWidths))
	for w, n := range byWidths {
		if n > 0 {
			ws = append(ws, w)
		}
	}
	sort.Slice(ws, func(i, j int) bool { return byWidths[ws[i]] > byWidths[ws[j]] })
	out := ""
	for _, w := range ws {
		out += fmt.Sprintf(" %s:%d", w, byWidths[w])
	}
	if out == "" {
		return " (aucun accord)"
	}
	return out
}
