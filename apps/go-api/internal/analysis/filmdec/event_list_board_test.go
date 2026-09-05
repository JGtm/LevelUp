package filmdec

// event_list_board_test.go — INSTRUMENT DE MESURE (lot V3, item B) : L'OCCUPANT DE
// L'EMBARQUEMENT. LECTURE SEULE, garde par V3_BOARD_FILMS / V3_BOARD_ROOT (sans env : saute).
//
// CE QUE GHIDRA A TRANCHE (lecture du 2026-09-02, exe HaloInfinite.exe, LECTURE SEULE).
// Le portage du 2026-09-01 lisait la réf 0 de l'embarquement en domaine 1 avec sonde, et les deux
// suivantes en domaine 7 — trois domaines faux sur trois. Le descripteur de `biped_board_vehicle`
// (vtable 0x143d0d330, reconnue par son thunk de nom vtable+0x08 = 0x14119e9b0 -> chaîne
// 0x143c97f80) porte en vtable+0x58 la fonction 0x142f1556c, qui aiguille sur l'indice de réf :
// 0 -> domaine 2, 1 -> domaine 3, 2 -> domaine 7. Le même emplacement du descripteur de
// `unit_exit_vehicle` (vtable 0x143d0c708, vtable+0x58 = 0x14080a018) rend 1, 1, 7 — exactement
// la grammaire déjà validée pour la sortie, ce qui contrôle la méthode de lecture.
// `FUN_1406d3140` ne lit la SONDE que pour le domaine 1 (`if (param_3 == 1 && ReadBit())`) : les
// réfs de l'embarquement n'en portent AUCUNE, et la « sonde variable » observée le 2026-09-01
// n'était que le premier bit de l'index.
//
// ============================ LE GATE, ECRIT AVANT LA MESURE ============================
//
// GATE B1 — RECOUPEMENT AVEC LE TROU DE POSITION (le même que V1a.4/V2b pour la sortie). Un
//   embarquement résolu doit coïncider avec le DÉBUT d'un trou du flux de position de son
//   occupant : l'enfant attaché cesse de répliquer. On exige que **>= 90 %** des embarquements
//   dont l'occupant tombe dans la bande bipède ouvrent un trou de >= evbTrouMinUS à moins de
//   evbTolUS de l'instant de l'événement.
//   TEMOIN : le même test à l'instant DÉCALÉ de evbDecalageUS (37 s, décalage constant et non
//   résonant — un décalage d'un cran ne fait pas un témoin sur série autocorrélée, leçon V1a.3).
// GATE B2 — L'OCCUPANT EST DANS LA BANDE BIPEDE. Part des embarquements dont
//   `base + index(8)` tombe dans la bande des slots ti=35 : >= 90 % (la sortie est à 95,5 %).
// GATE B3 — CONTROLE DE NON-REGRESSION DE LA SORTIE. Le même test, sur les SORTIES du même
//   corpus, doit rendre les chiffres déjà publiés (occupant en bande ~95 %, recoupement ~100 %).
//   Si la sortie bouge, c'est le portage qui est cassé, pas l'embarquement qui est résolu.
//
// LA LARGEUR DES DOMAINES 2 ET 3 EST RUNTIME (`FUN_1406d310c(count)` = ceil(log2(count)) sur une
// table BSS, illisible statiquement). 8 bits est la valeur de la build de référence. L'instrument
// BALAIE donc les couples (w0, w1) plausibles et publie le recoupement de chacun : c'est la
// mesure, et non une constante devinée, qui départage.
//
//	CGO_ENABLED=0 V3_BOARD_ROOT=<depot>/data/cache \
//	  V3_BOARD_FILMS="0d76e8f1,4898d586,e232ffce" \
//	  go test ./internal/analysis/filmdec/ -run TestV3BoardOccupant -v -timeout 60m

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// Seuils du gate, écrits avant mesure.
const (
	// evbTrouMinUS : durée minimale d'un trou du flux de position pour compter (= attTrouMS de
	// la chaîne véhicule : 3 s).
	evbTrouMinUS = uint64(3_000_000)
	// evbTolUS : écart maximal entre l'instant de l'événement et l'ouverture (embarquement) ou
	// la fermeture (sortie) du trou.
	evbTolUS = uint64(2_000_000)
	// evbDecalageUS : décalage du témoin.
	evbDecalageUS = uint64(37_000_000)
	// evbPartMin : seuil des gates B1 et B2.
	evbPartMin = 0.90
)

// evbLargeurs : les triplets de largeurs (domaine 2, domaine 3, domaine 7) balayés. Le premier
// est celui de la build de référence ; les autres bornent l'incertitude runtime (la largeur d'un
// domaine vaut ceil(log2(taille de sa table)), et le domaine 7 suit `FrameConfig.IDLowBits`,
// 11..14 selon le film). L'occupant ne dépend que du domaine 2 ; le SIÈGE dépend des trois.
var evbLargeurs = [][3]int{
	{8, 8, 13}, {8, 8, 11}, {8, 8, 12}, {8, 8, 14},
	{8, 5, 13}, {8, 6, 13}, {8, 7, 13}, {8, 9, 13}, {8, 10, 13}, {8, 13, 13},
	{8, 7, 11}, {8, 7, 12}, {8, 7, 14},
	{7, 7, 13}, {9, 8, 13}, {13, 8, 13},
}

// evbTrou est un trou du flux de position d'un bipède.
type evbTrou struct {
	Slot           uint32
	DebutUS, FinUS uint64
}

func TestV3BoardOccupant(t *testing.T) {
	root := os.Getenv("V3_BOARD_ROOT")
	films := strings.Split(os.Getenv("V3_BOARD_FILMS"), ",")
	if root == "" || os.Getenv("V3_BOARD_FILMS") == "" {
		t.Skipf("mesure non demandee : V3_BOARD_ROOT / V3_BOARD_FILMS vides")
	}
	release := LockProcessDecode()
	defer release()
	tot := evbTotaux{parLargeur: map[[3]int]*evbCompte{}, ecartsBase: map[int64]int{}}
	for _, f := range films {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		evbUnFilm(t, filepath.Join(root, "film_chunks", f), f, &tot)
	}
	evbRapport(t, &tot)
}

// evbCompte agrège un couple de largeurs.
type evbCompte struct {
	board, enBande, recoupe, temoin int
	// sieges : histogramme du siège R(6) lu APRÈS les trois références, pour ce triplet de
	// largeurs. La sortie rend « siège 0 dominant » (le conducteur qui descend) : un triplet
	// qui rend la même forme pour l'embarquement est le bon.
	sieges map[uint32]int
	// paires : embarquements suivis d'une SORTIE du MÊME occupant. seatAccord : parmi elles,
	// celles dont le siège de l'embarquement égale celui de la sortie. trajet : celles dont
	// l'embarquement ouvre un trou que la sortie appariée referme (le trajet complet).
	paires, seatAccord, trajet int
}

// evbTotaux agrège le corpus.
type evbTotaux struct {
	films, trous                       int
	parLargeur                         map[[3]int]*evbCompte
	exits, exitEnBande, exitRecoupe    int
	exitTemoin                         int
	boardTotal, boardAvecOccupantPorte int
	// Diagnostic du décalage de base entre domaines (cf. evbEcartsDeBase).
	ecartsBase                                  map[int64]int
	trousFermesParSortie, trousFermesParSortie0 int
}

// evbUnFilm mesure UN film : trous de position, puis embarquements (par largeur) et sorties.
func evbUnFilm(t *testing.T, dir, id string, tot *evbTotaux) {
	t.Helper()
	if CountFilmChunks(dir) == 0 {
		t.Logf("V3b %s : film absent du cache — saute", id)
		return
	}
	evs, err := ScanFilmVehicleEvents(dir)
	if err != nil {
		t.Logf("V3b %s : liste d'evenements : %v", id, err)
		return
	}
	// V3_BOARD_COUNT : mode RECENSEMENT — compte les evenements sans decoder les positions
	// (le nuage bipede coute 60-100 s par film). Sert a choisir les films qui portent des
	// embarquements avant de payer le decodage.
	if os.Getenv("V3_BOARD_COUNT") != "" {
		nb, ne := 0, 0
		for _, e := range evs {
			if e.Kind == EventUnitExitVehicle {
				ne++
			} else {
				nb++
			}
		}
		t.Logf("V3b RECENSEMENT %s — board %d · exit %d", id, nb, ne)
		return
	}
	// QUANTA SEULS, et c'est le bon choix ici : un TROU est une absence d'ECHANTILLON dans le
	// temps, pas une absence de coordonnee. Sans bornes de carte le decodeur refuse de
	// dequantifier, mais la presence des records est identique — et la moitie du corpus a
	// embarquements (CTF/Slayer hors Behemoth SF) n'a pas de bornes au catalogue.
	opt := DefaultScanFilmOptions()
	opt.QuantaOnly = true
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Logf("V3b %s : nuage bipede : %v", id, err)
		return
	}
	trous := evbTrous(pos)
	base, band := evbBande(dir)
	tot.films++
	tot.trous += len(trous)
	// DEUX PASSES, ET C'EST OBLIGATOIRE : l'appariement embarquement -> sortie cherche une
	// sortie POSTERIEURE. En une seule passe, un embarquement ne verrait que les sorties deja
	// rencontrees, c'est-a-dire aucune de celles qui l'interessent — le compte serait
	// systematiquement nul, et le nul ressemblerait a une refutation.
	var sorties []evbSortie
	nb, ne := 0, 0
	for _, e := range evs {
		if e.Kind == EventUnitExitVehicle {
			ne++
			evbNoteSortie(e, trous, tot)
			if e.OccupantPresent {
				sorties = append(sorties, evbSortie{e.OccupantSlot, e.TimestampUS, e.Seat})
			}
			continue
		}
		nb++
	}
	cadre := evbCadre{base, band, trous, sorties}
	for _, e := range evs {
		if e.Kind != EventUnitExitVehicle {
			evbNoteEmbarquement(dir, e, cadre, tot)
		}
	}
	t.Logf("V3b %s — %d trous >= 3 s · %d embarquements · %d sorties (base bande %d, %d slots)",
		id, len(trous), nb, ne, base, band.Count())
}

// evbNoteSortie impute une SORTIE au contrôle de non-régression (gate B3) : la sortie FERME un
// trou.
func evbNoteSortie(e VehicleEvent, trous []evbTrou, tot *evbTotaux) {
	tot.exits++
	if !e.OccupantInBand {
		return
	}
	tot.exitEnBande++
	if evbFerme(trous, e.OccupantSlot, e.TimestampUS) {
		tot.exitRecoupe++
	}
	if evbFerme(trous, e.OccupantSlot, e.TimestampUS+evbDecalageUS) {
		tot.exitTemoin++
	}
}

// evbSortie est une SORTIE datée, gardée pour l'APPARIEMENT embarquement -> sortie.
type evbSortie struct {
	Slot uint32
	TsUS uint64
	Seat uint32
}

// evbCadre regroupe le contexte d'un film (règle des 5 paramètres).
type evbCadre struct {
	base    uint32
	band    SlotBand
	trous   []evbTrou
	sorties []evbSortie
}

// evbSortieApres rend la PREMIÈRE sortie de ce slot après l'instant donné.
func (c evbCadre) sortieApres(slot uint32, at uint64) (evbSortie, bool) {
	best, ok := evbSortie{}, false
	for _, s := range c.sorties {
		if s.Slot != slot || s.TsUS <= at {
			continue
		}
		if !ok || s.TsUS < best.TsUS {
			best, ok = s, true
		}
	}
	return best, ok
}

// evbTrouOuvertA rend le trou de ce slot dont l'OUVERTURE tombe à moins de evbTolUS.
func (c evbCadre) trouOuvertA(slot uint32, at uint64) (evbTrou, bool) {
	for _, g := range c.trous {
		if g.Slot == slot && evbEcart(g.DebutUS, at) <= evbTolUS {
			return g, true
		}
	}
	return evbTrou{}, false
}

// evbNoteEmbarquement rejoue les trois références de l'embarquement pour chaque triplet de
// largeurs balayé, et impute : occupant en bande, ouverture de trou, témoin, siège, et
// l'APPARIEMENT embarquement -> sortie du même occupant (avec accord des sièges et trajet
// complet ouverture/fermeture du même trou).
func evbNoteEmbarquement(dir string, e VehicleEvent, c evbCadre, tot *evbTotaux) {
	tot.boardTotal++
	pay := evbPayload(dir, e)
	if pay == nil {
		return
	}
	tot.boardAvecOccupantPorte++
	base, band, trous := c.base, c.band, c.trous
	for _, w := range evbLargeurs {
		cw := tot.parLargeur[w]
		if cw == nil {
			cw = &evbCompte{sieges: map[uint32]int{}}
			tot.parLargeur[w] = cw
		}
		cw.board++
		r0 := readPlainRef(pay, eventPayloadStartBit, w[0])
		r1 := readPlainRef(pay, r0.EndBit, w[1])
		r2 := readPlainRef(pay, r1.EndBit, w[2])
		var seat uint32
		seatOK := false
		if b := r2.EndBit; b+vehicleSeatBits <= len(pay)*8 {
			seat = readBitsAt(pay, b, vehicleSeatBits)
			seatOK = true
			cw.sieges[seat]++
		}
		if !r0.Present {
			continue
		}
		slot := base + r0.Index
		if !band.Has(slot) {
			continue
		}
		cw.enBande++
		g, ouvre := c.trouOuvertA(slot, e.TimestampUS)
		if ouvre {
			cw.recoupe++
		}
		if evbOuvre(trous, slot, e.TimestampUS+evbDecalageUS) {
			cw.temoin++
		}
		if w[0] == dom2RefWidth && w[1] == dom3RefWidth && w[2] == dom7RefWidth {
			evbEcartsDeBase(e, slot, g, ouvre, c, tot)
		}
		sortie, appariee := c.sortieApres(slot, e.TimestampUS)
		if !appariee {
			continue
		}
		cw.paires++
		if seatOK && seat == sortie.Seat {
			cw.seatAccord++
		}
		if ouvre && evbEcart(g.FinUS, sortie.TsUS) <= evbTolUS {
			cw.trajet++
		}
	}
}

// evbEcartsDeBase teste l'hypothèse d'un DÉCALAGE DE BASE entre le domaine 2 (embarquement) et
// le domaine 1 (sortie) : si les deux domaines numérotent le même bipède depuis deux bases
// différentes, l'occupant d'embarquement et celui de la sortie qui referme SON trou diffèrent
// d'une constante. On relève donc, pour chaque embarquement qui ouvre un trou, l'écart entre le
// slot des sorties qui tombent à la fermeture de ce trou et le slot lu à l'embarquement.
func evbEcartsDeBase(e VehicleEvent, slot uint32, g evbTrou, ouvre bool, c evbCadre, tot *evbTotaux) {
	if !ouvre {
		return
	}
	tot.trousFermesParSortie0++
	for _, s := range c.sorties {
		if evbEcart(s.TsUS, g.FinUS) <= evbTolUS {
			tot.ecartsBase[int64(s.Slot)-int64(slot)]++
			tot.trousFermesParSortie++
			return
		}
	}
}

// evbPayload relit le payload du paquet qui porte l'événement (le décodeur ne le conserve pas).
func evbPayload(dir string, e VehicleEvent) []byte {
	data, err := ReadFilmChunk(dir, e.Chunk)
	if err != nil {
		return nil
	}
	for _, p := range WalkPackets(data) {
		if p.Index == e.PacketIndex && p.Type == PacketTypeDelta {
			return p.Payload(data)
		}
	}
	return nil
}

// evbBande relève la bande de slots bipèdes et sa base (min), comme ScanVehicleEvents.
func evbBande(dir string) (uint32, SlotBand) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return 0, SlotBand{}
	}
	band := NewFilmContext(film).BipedSlots()
	base := uint32(0)
	if slots := band.Slots(); len(slots) > 0 {
		base = slots[0]
	}
	return base, band
}

// evbTrous relève les trous >= evbTrouMinUS du flux de position (points monde seulement, comme
// la primitive V1a.4).
func evbTrous(pos []BipedPosition) []evbTrou {
	parSlot := map[uint32][]uint64{}
	for _, p := range pos {
		parSlot[p.Slot] = append(parSlot[p.Slot], p.TimestampUS)
	}
	var out []evbTrou
	for s, ts := range parSlot {
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		for i := 1; i < len(ts); i++ {
			if ts[i]-ts[i-1] >= evbTrouMinUS {
				out = append(out, evbTrou{Slot: s, DebutUS: ts[i-1], FinUS: ts[i]})
			}
		}
	}
	return out
}

// evbOuvre : un trou de ce slot s'OUVRE à moins de evbTolUS de l'instant.
func evbOuvre(trous []evbTrou, slot uint32, at uint64) bool {
	for _, g := range trous {
		if g.Slot == slot && evbEcart(g.DebutUS, at) <= evbTolUS {
			return true
		}
	}
	return false
}

// evbFerme : un trou de ce slot se FERME à moins de evbTolUS de l'instant.
func evbFerme(trous []evbTrou, slot uint32, at uint64) bool {
	for _, g := range trous {
		if g.Slot == slot && evbEcart(g.FinUS, at) <= evbTolUS {
			return true
		}
	}
	return false
}

func evbEcart(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

func evbPart(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}
