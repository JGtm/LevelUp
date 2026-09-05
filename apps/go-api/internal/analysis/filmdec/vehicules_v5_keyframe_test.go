package filmdec

// vehicules_v5_keyframe_test.go — LOT V5, PISTE 1 : L'ÉTAT D'OCCUPATION DANS L'IMAGE-CLÉ.
//
// L'AXIOME (utilisateur, 2026-09-02) : le mode Théâtre affiche conducteur / passager /
// artilleur À TOUT INSTANT et autorise le saut dans le temps. L'état d'occupation est donc
// REJOUABLE, donc sérialisé comme ÉTAT et ré-émis périodiquement. L'image-clé EST le point de
// reprise du seek, et elle porte un ÉTAT COMPLET (tous les composants de l'archétype, sans
// masque épars — cf. keyframe_record_walk.go).
//
// POURQUOI PAS LE FLUX DELTA (piste 5, mesurée d'abord). Le recensement `TestV5Recensement`
// montre que les composants `unit-*` porteurs de références n'apparaissent que dans ~0,15 %
// des records delta bipède (50 lectures d'i10 sur 33 531 records propres), et que leurs
// valeurs ne se répètent JAMAIS (autant de valeurs distinctes que de lectures) — signature
// d'une lecture désalignée, pas d'un état. Côté véhicule c'est pire : `ti=40` désynchronise
// avant d'atteindre i18 (i2/i3 réfutés en V1a/V2b).
//
// LA MÉTHODE, SANS GRAMMAIRE. Le corps d'un record d'image-clé n'est PAS bit-exact (aucun
// décalage ne rend une marche exacte, cf. keyframe_record_walk.go). On ne le PARSE donc pas :
// on BALAIE, exactement comme `ScanFilmKeyframeLoadouts` balaie les identifiants de famille
// d'arme. Pour chaque DÉCALAGE en bits relatif au début d'un record, on demande : la valeur
// lue à ce décalage désigne-t-elle le partenaire attendu (le véhicule pour un occupant, ou
// l'occupant pour un véhicule) PENDANT les épisodes attestés, et rien sinon ?
//
// LA VÉRITÉ TERRAIN. Un épisode attesté = [début du trou du flux de position d'un slot bipède,
// instant de la SORTIE qui le referme]. La sortie est décodée et validée (occupant 100 % en
// bande, fermeture de trou 90,7 % contre 0 % au témoin, cf. V3_EMBARQUEMENT § 2.4).
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=0d76e8f1,fccc61cd \
//	  go test ./internal/analysis/filmdec/ -run TestV5Keyframe -v -timeout 60m

import (
	"fmt"
	"sort"
	"testing"
)

// v5TrouMinUS est la durée minimale d'une absence du flux de position pour compter comme un
// TROU (même seuil que V1a.4 / V2b : 3 s).
const v5TrouMinUS = 3_000_000

// v5FermetureUS est la tolérance entre l'instant d'une sortie et la fin du trou qu'elle
// referme (même seuil que V2b : 2 s).
const v5FermetureUS = 2_000_000

// v5DecalTemoinUS est le décalage du témoin temporel (même valeur que V2b/V3 : 37 s).
const v5DecalTemoinUS = 37_000_000

// v5Episode est UN trajet attesté : le slot bipède occupant, et la fenêtre pendant laquelle il
// est à bord (début du trou -> instant de la sortie).
type v5Episode struct {
	Slot         uint32
	DebutUS      uint64 // dernier échantillon de position avant le trou
	FinUS        uint64 // instant de la sortie
	Seat         uint32
	SeatValid    bool
	DureeSeconde float64
}

// v5Episodes construit les épisodes attestés d'un film : pour chaque SORTIE, le trou du flux
// de position de son occupant qui se referme à cet instant.
func v5Episodes(dir string) ([]v5Episode, map[uint32][]uint64, error) {
	evs, err := ScanFilmVehicleEvents(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("événements véhicule : %w", err)
	}
	pos, err := ScanFilmBipedPositions(dir, ScanFilmOptions{QuantaOnly: true})
	if err != nil {
		return nil, nil, fmt.Errorf("positions bipède : %w", err)
	}
	parSlot := map[uint32][]uint64{}
	for _, p := range pos {
		parSlot[p.Slot] = append(parSlot[p.Slot], p.TimestampUS)
	}
	for s := range parSlot {
		sort.Slice(parSlot[s], func(i, j int) bool { return parSlot[s][i] < parSlot[s][j] })
	}
	var out []v5Episode
	for _, e := range evs {
		if e.Kind != EventUnitExitVehicle || !e.OccupantInBand {
			continue
		}
		d, f, ok := v5TrouFermePar(parSlot[e.OccupantSlot], e.TimestampUS)
		if !ok {
			continue
		}
		out = append(out, v5Episode{
			Slot: e.OccupantSlot, DebutUS: d, FinUS: f,
			Seat: e.Seat, SeatValid: e.SeatValid,
			DureeSeconde: float64(f-d) / 1e6,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DebutUS < out[j].DebutUS })
	return out, parSlot, nil
}

// v5TrouFermePar cherche, dans la suite d'horodatages `ts`, le trou (>= v5TrouMinUS) dont la
// FIN tombe à moins de v5FermetureUS de `at`. Il rend le début et la fin du trou.
func v5TrouFermePar(ts []uint64, at uint64) (debut, fin uint64, ok bool) {
	for i := 1; i < len(ts); i++ {
		if ts[i]-ts[i-1] < v5TrouMinUS {
			continue
		}
		if v5AbsDiff(ts[i], at) <= v5FermetureUS {
			return ts[i-1], at, true
		}
	}
	return 0, 0, false
}

func v5AbsDiff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// v5KfRec est UN record d'image-clé avec ses bornes en bits.
type v5KfRec struct {
	Slot, TI       int
	BitStart, Fin  int
	Payload        []byte
	TS             uint64
	Chunk, Paquet  int
	LongueurEnBits int
	// SautSlot est l'écart de SLOT jusqu'au record suivant TROUVÉ par le balayeur. Il vaut 1
	// quand aucun record n'a été sauté. IL EST LE CONTRÔLE DE CONFUSION de la mesure de
	// longueur : `WalkKeyframeWorld` saute les records dont `Field26` n'est pas nul
	// (keyframe_record_walk.go), donc une « longueur » anormale peut n'être que la somme de
	// plusieurs records. Sans ce chiffre, un écart de longueur ne prouve rien.
	SautSlot int
}

// v5Keyframes rend, pour chaque image-clé du film, les records bornés. Les emprises viennent
// de `KeyframeRecordSpans` (production) : l'instrument ne recalcule PAS de bornes pour son
// compte, sinon la mesure et le décodeur livré pourraient diverger.
func v5Keyframes(dir string) [][]v5KfRec {
	var out [][]v5KfRec
	for c := 1; c <= CountFilmChunks(dir); c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(data)
			spans := KeyframeRecordSpans(pay)
			bornes := make([]v5KfRec, 0, len(spans))
			for _, s := range spans {
				bornes = append(bornes, v5KfRec{
					Slot: s.Slot, TI: s.TI, BitStart: s.BitStart, Fin: s.BitEnd, Payload: pay,
					TS: p.TimestampUS, Chunk: c, Paquet: p.Index,
					LongueurEnBits: s.LengthBits, SautSlot: s.SlotGap,
				})
			}
			out = append(out, bornes)
		}
	}
	return out
}

// TestV5KeyframeOracle — ÉTAPE 2 : publier la vérité terrain et sa COUVERTURE par les
// images-clés. Sans image-clé À L'INTÉRIEUR d'un épisode, la piste 1 n'a rien à mesurer :
// c'est le premier chiffre à donner, avant tout balayage.
func TestV5KeyframeOracle(t *testing.T) {
	for _, dir := range v5Films(t) {
		eps, _, err := v5Episodes(dir)
		if err != nil {
			t.Logf("V5 ORACLE %s : %v", dir, err)
			continue
		}
		kfs := v5Keyframes(dir)
		var tsKf []uint64
		for _, k := range kfs {
			if len(k) > 0 {
				tsKf = append(tsKf, k[0].TS)
			}
		}
		dedans, avecKf := 0, 0
		for _, e := range eps {
			n := 0
			for _, ts := range tsKf {
				if ts > e.DebutUS && ts < e.FinUS {
					n++
				}
			}
			dedans += n
			if n > 0 {
				avecKf++
			}
		}
		t.Logf("V5 ORACLE %s — %d épisodes attestés, %d images-clés ; épisodes contenant au "+
			"moins une image-clé : %d/%d, instants (épisode x image-clé) : %d",
			dir, len(eps), len(tsKf), avecKf, len(eps), dedans)
		for i, e := range eps {
			n := 0
			for _, ts := range tsKf {
				if ts > e.DebutUS && ts < e.FinUS {
					n++
				}
			}
			t.Logf("    ep%-2d slot=%-5d [%8.2f s -> %8.2f s] durée=%6.2f s siège=%d(%v) images-clés dedans=%d",
				i, e.Slot, float64(e.DebutUS)/1e6, float64(e.FinUS)/1e6, e.DureeSeconde,
				e.Seat, e.SeatValid, n)
		}
		// Recensement des records d'image-clé par archétype (le balayage a-t-il de quoi
		// travailler ?).
		parTI := map[int]int{}
		for _, k := range kfs {
			for _, r := range k {
				parTI[r.TI]++
			}
		}
		t.Logf("    records d'image-clé : ti=35 %d, ti=40 %d, total %d",
			parTI[v5BipedeTI], parTI[v5VehiculeTI], v5Somme(parTI))
	}
}

func v5Somme(m map[int]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}
