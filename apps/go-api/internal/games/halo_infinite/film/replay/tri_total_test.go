package replay

// tri_total_test.go — LOT J10.1 (2026-09-27, DT-9) : les tris de l'assemblage qui DECIDENT D'UNE
// SORTIE (un calque publie, un « premier gagne », un appariement) et dont la cle n'etait pas
// prouvee unique. Plus de douze elements et peu de cles : sous treize, `sort.Slice` passe par un
// tri par insertion, stable par accident — le hasard que DT-9 retire. L'attendu est calcule sans
// le tri teste : soit l'ordre d'un tri STABLE (rang d'entree), soit un ordre lexicographique.

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

const (
	nExAequo      = 48
	graineExAequo = 20260927
)

func melanger[T any](s []T) {
	rand.New(rand.NewSource(graineExAequo)).Shuffle(len(s), func(a, b int) { s[a], s[b] = s[b], s[a] })
}

// rangsStables : les rangs d'entree ranges par cle, puis par rang (l'ordre d'un tri STABLE).
func rangsStables(cles []int) []int {
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

// lexCroissant echoue si deux tuples consecutifs ne sont pas STRICTEMENT croissants.
func lexCroissant(t *testing.T, quoi string, cles [][]int) {
	t.Helper()
	for i := 1; i < len(cles); i++ {
		a, b := cles[i-1], cles[i]
		k := 0
		for k < len(a) && a[k] == b[k] {
			k++
		}
		if k == len(a) || a[k] > b[k] {
			t.Fatalf("%s : rang %d, %v puis %v — ordre total rompu", quoi, i, a, b)
		}
	}
}

// TestBuildRoster_DeuxBotsDuMemeIndex — RA2-5. Des remplacants successifs partagent un index de
// film et n'ont pas de xuid : la cle (index, xuid) ne les separait pas. Cinquante executions,
// bots permutes a graine fixe : sortie identique, bots d'un meme index par nom.
func TestBuildRoster_DeuxBotsDuMemeIndex(t *testing.T) {
	idx := types.PlayerIndexTable{ByXUID: map[uint64]int{100: 0, 101: 1}}
	var bots []BotIdentity
	for i := range 14 {
		bots = append(bots, BotIdentity{FilmIndex: 5, Name: fmt.Sprintf("343 Bot%02d", 13-i)})
	}
	rng := rand.New(rand.NewSource(graineExAequo))
	var premiere []RosterEntry
	for run := range 50 {
		in := append([]BotIdentity(nil), bots...)
		rng.Shuffle(len(in), func(a, b int) { in[a], in[b] = in[b], in[a] })
		out := buildRoster(idx, nil, in, teamPublication{})
		if run == 0 {
			premiere = out
			continue
		}
		if !reflect.DeepEqual(out, premiere) {
			t.Fatalf("execution %d : roster different de la premiere", run)
		}
	}
	for i := 3; i < len(premiere); i++ {
		if premiere[i-1].Name >= premiere[i].Name {
			t.Fatalf("rang %d : %q puis %q — deux bots du meme index non ranges par nom",
				i, premiere[i-1].Name, premiere[i].Name)
		}
	}
}

func TestTrierArmesAuSol_ExAequoDepartagesParLeContenu(t *testing.T) {
	in := make([]GroundWeapon, nExAequo)
	for i := range in {
		in[i] = GroundWeapon{T0: i % 2, W: "0x0000002A", X: float32(i)}
	}
	melanger(in)
	trierArmesAuSol(in)
	cles := make([][]int, len(in))
	for i, g := range in {
		cles[i] = []int{g.T0, int(g.X)}
	}
	lexCroissant(t, "armes au sol", cles)
}

func TestTrackFrameWindows_ExAequoDepartagesParLaFin(t *testing.T) {
	tracks := make([]Track, nExAequo)
	for i := range tracks {
		tracks[i] = Track{Slot: 3, StartFrame: i % 3, EndFrame: 100 + i}
	}
	melanger(tracks)
	w := trackFrameWindows(tracks, nil)[3]
	cles := make([][]int, len(w))
	for i, f := range w {
		cles[i] = []int{f.from, f.to}
	}
	lexCroissant(t, "fenetres de vie", cles)
}

func TestTrierCreationsParInstant_ExAequoDansLOrdreDuFilm(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]types.EquipmentCreation, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = types.EquipmentCreation{TimestampUS: uint64(i % 3), BitPos: i}
	}
	trierCreationsParInstant(in)
	got := make([]int, len(in))
	for i, c := range in {
		got[i] = c.BitPos
	}
	if want := rangsStables(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("creations :\n got %v\nwant %v", got, want)
	}
}

func TestTrierRentrees_ExAequoDepartagesParDrapeau(t *testing.T) {
	in := make([]flagHomecoming, nExAequo)
	for i := range in {
		in[i] = flagHomecoming{at: int64(i % 3), flag: i % 4, x: float32(i)}
	}
	melanger(in)
	trierRentrees(in)
	cles := make([][]int, len(in))
	for i, h := range in {
		cles[i] = []int{int(h.at), h.flag, int(h.x)}
	}
	lexCroissant(t, "rentrees", cles)
}

func TestTrierGrenades_ExAequoDepartagesParLeRang(t *testing.T) {
	in := make([]Grenade, nExAequo)
	for i := range in {
		in[i] = Grenade{T: i % 2, Rank: i, Src: GrenadeSrcBiped}
	}
	melanger(in)
	trierGrenades(in)
	cles := make([][]int, len(in))
	for i, g := range in {
		cles[i] = []int{g.T, g.Rank}
	}
	lexCroissant(t, "grenades", cles)
}

func TestNaissancesDObjetParSlot_ExAequoDansLOrdreDuFilm(t *testing.T) {
	cles := make([]int, nExAequo)
	c := naissancesDObjetParSlot{}
	for i := range nExAequo {
		cles[i] = i % 3
		c.ajouter(7, naissanceDObjet{tUS: uint64(i % 3), retenue: i%2 == 0, recensee: i%4 < 2})
	}
	signature := func(n naissanceDObjet) [2]bool { return [2]bool{n.retenue, n.recensee} }
	var want [][2]bool
	for _, r := range rangsStables(cles) {
		want = append(want, [2]bool{r%2 == 0, r%4 < 2})
	}
	c.trier()
	got := make([][2]bool, 0, nExAequo)
	for _, n := range c[7] {
		got = append(got, signature(n))
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("naissances d'un slot : ex aequo hors de l'ordre du film")
	}
}

func TestGwMembersByTime_ExAequoDepartagesParRang(t *testing.T) {
	objs := make([]gwPickupObject, nExAequo)
	members := make([]int, nExAequo)
	for i := range objs {
		objs[i].Appar.TUS = uint64(i % 3)
		members[i] = nExAequo - 1 - i
	}
	ms := gwMembersByTime(objs, members)
	cles := make([][]int, len(ms))
	for i, m := range ms {
		cles[i] = []int{int(objs[m].Appar.TUS), m}
	}
	lexCroissant(t, "membres d'un socle", cles)
}

func TestCorpsParSlot_ExAequoDepartagesParIndex(t *testing.T) {
	in := make([]grammar.BipedCreation, nExAequo)
	for i := range in {
		in[i] = grammar.BipedCreation{Slot: 9, Generation: uint32(i % 2), ParticipantIndex: uint32(i),
			HasIndex: true, TimestampUS: uint64(i % 3)}
	}
	melanger(in)
	dates := corpsParSlot(in)[9].dates
	cles := make([][]int, len(dates))
	for i, d := range dates {
		cles[i] = []int{int(d.tUS), int(d.gen), int(d.index)}
	}
	lexCroissant(t, "dates de creation", cles)
}

// Cinq vies finissent au meme instant, vingt morts y tombent : tous les candidats sont a egale
// distance. L'appariement doit etre (vie i, mort i), dans l'ordre des indices.
func TestApparierMortsEtVies_ExAequoDepartagesParLaMort(t *testing.T) {
	var lives []lifeSpan
	for s := range 5 {
		lives = append(lives, lifeSpan{slot: uint32(s), from: 0, to: 10_000_000})
	}
	var deaths []types.Death
	for i := range 20 {
		deaths = append(deaths, types.Death{XUID: uint64(i + 1), TimeMS: 10_000})
	}
	got := apparierMortsEtVies(lives, deaths, 0)
	want := []deathPair{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("appariement :\n got %v\nwant %v", got, want)
	}
}

func TestViesParIdentite_ExAequoDepartagesParLaFin(t *testing.T) {
	tracks := make([]Track, nExAequo)
	for i := range tracks {
		tracks[i] = Track{Slot: uint32(i), XUID: "42", StartFrame: i % 3, EndFrame: 100 + i}
	}
	melanger(tracks)
	vs := viesParIdentite(tracks)["42"]
	cles := make([][]int, len(vs))
	for i, v := range vs {
		cles[i] = []int{v[0], v[1]}
	}
	lexCroissant(t, "vies par identite", cles)
}

func TestTrierTirs_ExAequoDansLOrdreDuFilm(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]Shot, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = Shot{T: i % 3, Slot: uint32(i)}
	}
	trierTirs(in)
	got := make([]int, len(in))
	for i, s := range in {
		got[i] = int(s.Slot)
	}
	if want := rangsStables(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("tirs :\n got %v\nwant %v", got, want)
	}
}

func TestIndexBySlot_ExAequoDansLOrdreDuFilm(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]grammar.BipedPosition, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = grammar.BipedPosition{Slot: 2, TimestampUS: uint64(i % 3), X: float32(i)}
	}
	pts := indexBySlot(in)[2].pts
	got := make([]int, len(pts))
	for i, p := range pts {
		got[i] = int(p.X)
	}
	if want := rangsStables(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("positions d'un slot :\n got %v\nwant %v", got, want)
	}
}

func TestTrierPortagesDeCrane_ExAequoDepartagesParLaFin(t *testing.T) {
	in := make([]skullRawCarry, nExAequo)
	for i := range in {
		in[i] = skullRawCarry{xuid: "", round: 1, t0MS: i % 2, t1MS: 1000 + i}
	}
	melanger(in)
	trierPortagesDeCrane(in)
	cles := make([][]int, len(in))
	for i, c := range in {
		cles[i] = []int{c.t0MS, c.round, c.t1MS}
	}
	lexCroissant(t, "portages du crane", cles)
}

// Quatre vies d'un meme slot recensees aux memes images-cles : la cle (slot, premiere image-cle)
// ne les separait pas, et la liste est batie en iterant une MAP.
func TestVehicleLives_ExAequoDepartagesParGeneration(t *testing.T) {
	kf := grammar.WorldObjectKeyframes{TimesUS: []uint64{1000, 2000, 3000}, SeenUS: map[types.LifeKey][]uint64{}}
	for g := range uint32(4) {
		kf.SeenUS[types.LifeKey{Slot: 7, Gen: g}] = []uint64{1000, 2000}
	}
	for run := range 50 {
		lives, _ := vehicleLives(kf, nil)
		for i, l := range lives {
			if l.key.Gen != uint32(i) {
				t.Fatalf("execution %d, rang %d : generation %d — vies d'un slot hors de l'ordre", run, i, l.key.Gen)
			}
		}
	}
}

func TestTrierPeriodesDeLunette_ExAequoDepartagesParLaFin(t *testing.T) {
	in := make([]zoomPeriode, nExAequo)
	for i := range in {
		in[i] = zoomPeriode{debut: uint64(i % 3), fin: uint64(100 + i), niveau: 1}
	}
	melanger(in)
	trierPeriodesDeLunette(in)
	cles := make([][]int, len(in))
	for i, p := range in {
		cles[i] = []int{int(p.debut), int(p.fin)}
	}
	lexCroissant(t, "periodes de lunette", cles)
}

func TestTrierPointsParInstant_ExAequoDansLOrdreDuFilm(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]point, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = point{tMS: int64(i % 3), x: float64(i)}
	}
	trierPointsParInstant(in)
	got := make([]int, len(in))
	for i, p := range in {
		got[i] = int(p.x)
	}
	if want := rangsStables(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("positions d'un joueur :\n got %v\nwant %v", got, want)
	}
}
