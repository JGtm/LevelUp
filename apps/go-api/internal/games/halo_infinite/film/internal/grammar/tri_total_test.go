package grammar

// tri_total_test.go — LOT J10.1 (2026-09-27, DT-9) : chaque tri de la grammaire qui DECIDE D'UNE
// SORTIE et dont la cle n'etait pas prouvee unique recoit ici son test d'ex aequo.
//
// LA FORME DES ENTREES EST VOULUE : plus de douze elements (sous treize, `sort.Slice` passe par un
// tri par insertion, stable par accident — c'est exactement le hasard que DT-9 retire) et
// quelques cles seulement, donc beaucoup d'ex aequo. Chaque element porte une IDENTITE (son rang
// d'entree, ou un champ distinct) ; l'attendu est calcule independamment du tri teste.

import (
	"math/rand"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// nExAequo : la taille des entrees d'ex aequo — au-dela du seuil du tri par insertion.
const nExAequo = 48

// graineExAequo : la graine FIXE des permutations (un test d'ordre doit etre reproductible).
const graineExAequo = 20260927

// attenduStable rend les identites rangees par cle, puis par rang d'entree : l'ordre qu'un tri
// STABLE sur la seule cle doit produire.
func attenduStable(cles []int) []int {
	maxi := 0
	for _, c := range cles {
		maxi = max(maxi, c)
	}
	out := make([]int, 0, len(cles))
	for c := 0; c <= maxi; c++ {
		for i, k := range cles {
			if k == c {
				out = append(out, i)
			}
		}
	}
	return out
}

func TestTrierMortsDuFil_ExAequoDepartagesParXUID(t *testing.T) {
	in := make([]types.Death, nExAequo)
	for i := range in {
		in[i] = types.Death{TimeMS: int64(i%4) * 1000, XUID: uint64(1000 + (i*37)%nExAequo)}
	}
	rand.New(rand.NewSource(graineExAequo)).Shuffle(len(in), func(a, b int) { in[a], in[b] = in[b], in[a] })
	trierMortsDuFil(in)
	for i := 1; i < len(in); i++ {
		a, b := in[i-1], in[i]
		if a.TimeMS > b.TimeMS || (a.TimeMS == b.TimeMS && a.XUID >= b.XUID) {
			t.Fatalf("ordre (instant, xuid) rompu au rang %d : %+v puis %+v", i, a, b)
		}
	}
}

// TestEquipmentChanges_OrdreStableSurExAequo — GB-3. Dix vies du MEME slot (generations
// distinctes) emettent dans le MEME paquet : la cle (instant, chunk, paquet, slot) ne les separe
// pas, et la liste finale etait batie en iterant une MAP. Cinquante executions, entrees permutees
// a graine fixe : la sortie doit etre identique, et rangee par generation.
func TestEquipmentChanges_OrdreStableSurExAequo(t *testing.T) {
	var base []abilityEmission
	for g := range uint32(10) {
		base = append(base, abilityEmission{Slot: 535, Gen: g, Chunk: 4, PacketIndex: 7,
			Bit: 100 + int(g), TimestampUS: 5_000, Counter: 1, Rank: int(g)})
	}
	rng := rand.New(rand.NewSource(graineExAequo))
	var premiere []types.EquipmentChange
	for run := range 50 {
		in := append([]abilityEmission(nil), base...)
		rng.Shuffle(len(in), func(a, b int) { in[a], in[b] = in[b], in[a] })
		out, _ := assembleEquipmentChanges(in, nil, nil)
		if run == 0 {
			premiere = out
			continue
		}
		if !reflect.DeepEqual(out, premiere) {
			t.Fatalf("execution %d : sortie differente de la premiere", run)
		}
	}
	// Le rang de palette porte la generation : la sortie doit etre rangee par vie (slot, generation).
	for i, ch := range premiere {
		if ch.Rank != i {
			t.Fatalf("rang %d : palette %d, attendu la generation %d (ordre par vie)", i, ch.Rank, i)
		}
	}
}

// Deux emissions STRICTES d'une meme vie dans un meme paquet ne se departagent que par la
// position de leur record : le tri de la chaine d'une vie doit la lire.
func TestSortEmissionsByFilmOrder_DepartageParBit(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]abilityEmission, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = abilityEmission{Slot: 9, TimestampUS: uint64(i%3) * 10, Chunk: 2, PacketIndex: i % 3, Bit: i}
	}
	sortEmissionsByFilmOrder(in)
	if got, want := bitsDesEmissions(in), attenduStable(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("ordre du film rompu :\n got %v\nwant %v", got, want)
	}
}

func TestMergeEquipEmissions_StrictesDuMemePaquetParBit(t *testing.T) {
	in := make([]abilityEmission, nExAequo)
	for i := range in {
		in[i] = abilityEmission{Slot: 9, TimestampUS: 10, Chunk: 2, PacketIndex: 5, Bit: nExAequo - i,
			Counter: uint32(i % 8)}
	}
	var st types.EquipmentChangeStats
	for _, list := range mergeEquipEmissions(in, nil, &st) {
		for i := 1; i < len(list); i++ {
			if list[i-1].Bit >= list[i].Bit {
				t.Fatalf("rang %d : bit %d puis %d — deux strictes du meme paquet non departagees",
					i, list[i-1].Bit, list[i].Bit)
			}
		}
	}
}

func bitsDesEmissions(in []abilityEmission) []int {
	out := make([]int, len(in))
	for i, e := range in {
		out[i] = e.Bit
	}
	return out
}

func TestTrierSerieNavpoint_ExAequoDansLOrdreDuFilm(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]types.NavpointRadialRead, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = types.NavpointRadialRead{Slot: 4, TMS: int32(i % 3), Q: uint8(i)}
	}
	trierSerieNavpoint(in)
	got := make([]int, len(in))
	for i, r := range in {
		got[i] = int(r.Q)
	}
	if want := attenduStable(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("serie navpoint :\n got %v\nwant %v", got, want)
	}
}

func TestTrierDeclarationsAnticipees_ExAequoDansLOrdreDuBalayage(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]declarationAnticipee, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = declarationAnticipee{chunk: i % 3, ti: uint32(i)}
	}
	trierDeclarationsAnticipees(in)
	got := make([]int, len(in))
	for i, d := range in {
		got[i] = int(d.ti)
	}
	if want := attenduStable(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("declarations :\n got %v\nwant %v", got, want)
	}
}

func TestTrierEchantillonsI0_ExAequoDansLOrdreDuBalayage(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]i0Sample, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = i0Sample{slot: 1, chunk: 1, pkt: i % 3, bits: [2]uint64{uint64(i), 0}}
	}
	trierEchantillonsI0(in)
	got := make([]int, len(in))
	for i, s := range in {
		got[i] = int(s.bits[0])
	}
	if want := attenduStable(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("echantillons i0 :\n got %v\nwant %v", got, want)
	}
}

func TestTrierTeleportations_ExAequoDepartagesParSlot(t *testing.T) {
	in := make([]types.TranslocatorTeleport, nExAequo)
	for i := range in {
		in[i] = types.TranslocatorTeleport{TimestampUS: uint64(i%3) * 1000, Slot: uint32(i)}
	}
	rand.New(rand.NewSource(graineExAequo)).Shuffle(len(in), func(a, b int) { in[a], in[b] = in[b], in[a] })
	trierTeleportations(in)
	for i := 1; i < len(in); i++ {
		a, b := in[i-1], in[i]
		if a.TimestampUS > b.TimestampUS || (a.TimestampUS == b.TimestampUS && a.Slot >= b.Slot) {
			t.Fatalf("ordre (instant, slot) rompu au rang %d : %+v puis %+v", i, a, b)
		}
	}
}

func TestTrierBasculesDeLunette_ExAequoDepartagesParSlot(t *testing.T) {
	in := make([]ZoomEvent, nExAequo)
	for i := range in {
		in[i] = ZoomEvent{TimestampUS: uint64(i%3) * 1000, Slot: uint32(i)}
	}
	rand.New(rand.NewSource(graineExAequo)).Shuffle(len(in), func(a, b int) { in[a], in[b] = in[b], in[a] })
	trierBasculesDeLunette(in)
	for i := 1; i < len(in); i++ {
		a, b := in[i-1], in[i]
		if a.TimestampUS > b.TimestampUS || (a.TimestampUS == b.TimestampUS && a.Slot >= b.Slot) {
			t.Fatalf("ordre (instant, slot) rompu au rang %d : %+v puis %+v", i, a, b)
		}
	}
}

func TestTrierEchantillonsDePosition_ExAequoDansLOrdreDuFilm(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]hitPosSample, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = hitPosSample{ts: uint64(i % 3), x: float32(i)}
	}
	trierEchantillonsDePosition(in)
	got := make([]int, len(in))
	for i, s := range in {
		got[i] = int(s.x)
	}
	if want := attenduStable(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("echantillons de position :\n got %v\nwant %v", got, want)
	}
}

func TestTrierDegatsParInstant_ExAequoDansLOrdreDuFilm(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]dmgSlot, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = dmgSlot{ts: uint64(i % 3), dmg: WeaponDamage{VictimIdx: i}}
	}
	trierDegatsParInstant(in)
	got := make([]int, len(in))
	for i, s := range in {
		got[i] = s.dmg.VictimIdx
	}
	if want := attenduStable(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("degats :\n got %v\nwant %v", got, want)
	}
}
