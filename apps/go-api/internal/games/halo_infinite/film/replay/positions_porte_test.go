package replay

// positions_porte_test.go — la porte des positions de bipede (lot M1 des retours du rejeu,
// 2026-09-23) : la grammaire de la vie (R-B1, R-B2) et l emprise jouee (repli F-1), au travers
// de l assemblage PUBLIC (`BuildFromPositions`). Les temoins de l annexe
// `retours_rejeu_2026-09-23/RAPPORT_positions_limbe.md` y sont rejoues en fixtures : aucune valeur
// de match n est dans le code de production, seulement leur FORME.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// porteFoule rend `n` positions d un joueur temoin (slot 900) qui arpente une carte de 100 m sur
// 100 m sur 10 m, une toutes les 100 ms a partir de `debutMS` : assez pour armer l emprise
// (`boundsMinSamples`), et une emprise connue d avance.
func porteFoule(n, debutMS int) []grammar.BipedPosition {
	out := make([]grammar.BipedPosition, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, pos(900, debutMS+100*i, float32(i%100), float32((i*7)%100), float32(i%10)))
	}
	return out
}

// porteCreation fabrique un record de creation de bipede lu, a `ms` millisecondes.
func porteCreation(slot uint32, ms int, gen uint32) grammar.BipedCreation {
	return grammar.BipedCreation{Slot: slot, Generation: gen, TimestampUS: uint64(ms) * 1000,
		HasIndex: true, ParticipantIndex: 3}
}

// porteTraces rend les traces publiees d un slot.
func porteTraces(doc ReplayDocument, slot uint32) []Track {
	var out []Track
	for _, tr := range doc.Tracks {
		if tr.Slot == slot {
			out = append(out, tr)
		}
	}
	return out
}

// porteRepli rend le compte publie d un repli dans `coverage.fallbacks`.
func porteRepli(doc ReplayDocument, nom fallback.Nom) int {
	if doc.Coverage == nil {
		return 0
	}
	for _, h := range doc.Coverage.Fallbacks {
		if h.Name == string(nom) {
			return h.Hits
		}
	}
	return 0
}

// R-B1 — le temoin de Madina (81c02726, slot 523) : un point 4,65 s AVANT la creation de son corps,
// a 440 m sous la carte, puis le vrai point d apparition. Le trou fait moins de `lifeGapUS` : sans
// la regle, les deux points font UNE vie, et le client « faisait voler » le joueur de l un a
// l autre. La vie commence desormais au record.
func TestPorteVieOuverteParSaCreationCommenceAuRecord(t *testing.T) {
	in := porteFoule(300, 1_000)
	in = append(in,
		// anterieur a la creation, et PREMIER paquet de position du film : ni une position de
		// corps, ni une raison de deplacer l origine
		pos(523, 0, -78.6, 46.38, -325.4),
		pos(523, 4_660, 10, 10, 1),
		pos(523, 4_760, 11, 10, 1),
		pos(523, 4_860, 12, 10, 1),
	)
	opt := Options{FrameIntervalMS: 100, BipedCreations: []grammar.BipedCreation{porteCreation(523, 4_650, 1)}}
	doc := BuildFromPositions("m", "halo_infinite", in, nil, opt)
	trs := porteTraces(doc, 523)
	if len(trs) != 1 {
		t.Fatalf("vies du slot 523 = %d, attendu 1", len(trs))
	}
	if first := trs[0].Points[0]; first.X != 10 || first.T != 46 {
		t.Errorf("premier point = %+v, attendu le vrai point d apparition (x=10) a la frame 46 : "+
			"l origine reste le PREMIER paquet de position du film", first)
	}
	c := doc.Coverage.Tracks
	if c.AvantCreation != 1 || c.ViesAvantPremiereCreation != 0 {
		t.Errorf("couverture = avantCreation %d, viesAvantPremiereCreation %d ; attendu 1 et 0",
			c.AvantCreation, c.ViesAvantPremiereCreation)
	}
	if doc.FrameCount != 310 {
		t.Errorf("frameCount = %d, attendu 310 : l origine et la duree restent lues sur TOUS les "+
			"paquets de position", doc.FrameCount)
	}
}

// R-B2 — une vie d un point entierement anterieure a la PREMIERE creation de son slot (20 au parc,
// toutes de 1 ou 2 points) n est plus publiee, et se compte.
func TestPorteAucuneVieAvantLaPremiereCreation(t *testing.T) {
	in := porteFoule(300, 0)
	in = append(in,
		pos(600, 1_000, 50, 50, 1),
		pos(600, 30_020, 20, 20, 1),
		pos(600, 30_120, 21, 20, 1),
	)
	opt := Options{FrameIntervalMS: 100, BipedCreations: []grammar.BipedCreation{porteCreation(600, 30_000, 1)}}
	doc := BuildFromPositions("m", "halo_infinite", in, nil, opt)
	trs := porteTraces(doc, 600)
	if len(trs) != 1 || trs[0].StartFrame != 300 {
		t.Fatalf("vies du slot 600 = %+v, attendu une seule vie, ouverte a la frame 300", trs)
	}
	c := doc.Coverage.Tracks
	if c.AvantCreation != 1 || c.ViesAvantPremiereCreation != 1 {
		t.Errorf("couverture = avantCreation %d, viesAvantPremiereCreation %d ; attendu 1 et 1",
			c.AvantCreation, c.ViesAvantPremiereCreation)
	}
}

// LA GENERATION EST LA GARDE : un premier record lu qui n est PAS celui du premier corps
// (`gen=2`, slot recycle dont la premiere creation n a pas ete lue) desarme la regle — les
// positions anterieures appartiennent a un corps reel.
func TestPorteDesarmeeQuandLePremierRecordNEstPasLePremierCorps(t *testing.T) {
	in := porteFoule(300, 0)
	in = append(in, pos(601, 1_000, 50, 50, 1), pos(601, 1_100, 51, 50, 1),
		pos(601, 30_020, 20, 20, 1))
	opt := Options{FrameIntervalMS: 100, BipedCreations: []grammar.BipedCreation{porteCreation(601, 30_000, 2)}}
	doc := BuildFromPositions("m", "halo_infinite", in, nil, opt)
	n := 0
	for _, tr := range porteTraces(doc, 601) {
		n += len(tr.Points)
	}
	if n != 3 || doc.Coverage.Tracks.AvantCreation != 0 {
		t.Errorf("points publies du slot 601 = %d (avantCreation %d), attendu 3 et 0 : la regle se "+
			"desarme sur un slot dont le premier record n est pas `gen=1`", n,
			doc.Coverage.Tracks.AvantCreation)
	}
	// LE DESARMEMENT EST PUBLIE, avec son denominateur : une derive de la numerotation des
	// generations (un build qui numeroterait le premier corps 0) desarmerait la regle sur tous les
	// slots — la couverture doit le montrer, pas seulement le journal.
	if c := doc.Coverage.Tracks; c.SlotsArmes != 0 || c.SlotsDesarmes != 1 {
		t.Errorf("slotsArmes = %d, slotsDesarmes = %d ; attendu 0 et 1", c.SlotsArmes, c.SlotsDesarmes)
	}
}

// LE PREMIER RECORD D UN SLOT RECYCLE : `gen=1` en tete de film, `gen=2` apres (`084a804d`). C est
// le PLUS PRECOCE qui dit si la regle s arme — lu dans le desordre du chunk, le `gen=2` tardif ne
// doit pas la desarmer.
func TestPortePremierRecordDUnSlotRecycle(t *testing.T) {
	in := porteFoule(300, 0)
	in = append(in, pos(602, 1_000, 50, 50, 1),
		pos(602, 30_020, 20, 20, 1), pos(602, 30_120, 21, 20, 1), pos(602, 60_020, 30, 30, 1))
	opt := Options{FrameIntervalMS: 100, BipedCreations: []grammar.BipedCreation{
		porteCreation(602, 60_000, 2), porteCreation(602, 30_000, 1),
	}}
	doc := BuildFromPositions("m", "halo_infinite", in, nil, opt)
	c := doc.Coverage.Tracks
	if c.AvantCreation != 1 || c.SlotsArmes != 1 || c.SlotsDesarmes != 0 {
		t.Errorf("avantCreation = %d, slotsArmes = %d, slotsDesarmes = %d ; attendu 1, 1, 0 : le point "+
			"anterieur au premier corps est ecarte", c.AvantCreation, c.SlotsArmes, c.SlotsDesarmes)
	}
}

// F-1 — un point hors de l emprise jouee (la meme garde que `boundsOf`) n est plus publie ; il se
// compte dans la couverture ET dans les replis du document.
func TestPortePointHorsEmpriseEcarte(t *testing.T) {
	in := porteFoule(300, 0)
	in = append(in, pos(610, 5_000, 30, 30, 1), pos(610, 5_100, 31, 30, 1),
		pos(611, 7_000, -78.6, 46.38, -325.4)) // vie d un point, 330 m sous une carte de 10 m de haut
	doc := BuildFromPositions("m", "halo_infinite", in, nil, Options{FrameIntervalMS: 100})
	if trs := porteTraces(doc, 611); len(trs) != 0 {
		t.Errorf("vie hors emprise publiee : %+v", trs)
	}
	if len(porteTraces(doc, 610)) != 1 {
		t.Error("la vie dans l emprise doit rester publiee")
	}
	if doc.Coverage.Tracks.HorsEmprise != 1 {
		t.Errorf("horsEmprise = %d, attendu 1", doc.Coverage.Tracks.HorsEmprise)
	}
	if got := porteRepli(doc, fallback.NomPositionHorsEmpriseEcartee); got != 1 {
		t.Errorf("repli publie = %d, attendu 1 declenchement", got)
	}
}

// F-1 DESARMEE sous `boundsMinSamples` : sur une poignee de points, des centiles ne disent rien,
// et rien n est ecarte.
func TestPorteEmpriseDesarmeeSurUnePoignee(t *testing.T) {
	in := []grammar.BipedPosition{pos(620, 0, 1, 1, 1), pos(620, 100, 2, 1, 1), pos(621, 200, -900, 900, 1)}
	doc := BuildFromPositions("m", "halo_infinite", in, nil, Options{FrameIntervalMS: 100})
	if len(porteTraces(doc, 621)) != 1 || doc.Coverage.Tracks.HorsEmprise != 0 {
		t.Errorf("sous %d positions, l emprise ne doit rien ecarter (horsEmprise %d)",
			boundsMinSamples, doc.Coverage.Tracks.HorsEmprise)
	}
}

// NEGATIF DE F-1 — une CHUTE REELLE hors de la carte (un joueur qui tombe dans un vide, une
// position toutes les 100 ms) sort de l emprise PAR CONTINUITE : elle reste publiee. Un faux
// en-tete, lui, surgit isole.
func TestPorteChuteContinueHorsEmpriseConservee(t *testing.T) {
	// Une foule de 10 000 positions : la chute (65) reste sous le centile 1, comme au parc, et ne
	// deplace pas l emprise qu elle traverse.
	in := porteFoule(10_000, 0)
	for i := 0; i <= 64; i++ {
		// de z = 5 a z = -148,6 : 2,4 m par 100 ms (24 m/s, la chute mesuree au corpus temoin),
		// au-dela du plancher de l emprise (~ -108 m)
		in = append(in, pos(630, 5_000+100*i, 50, 50, 5-2.4*float32(i)))
	}
	in = append(in, pos(631, 9_000, 50, 50, -325.4)) // isole, plus profond encore
	doc := BuildFromPositions("m", "halo_infinite", in, nil, Options{FrameIntervalMS: 100})
	n := 0
	for _, tr := range porteTraces(doc, 630) {
		n += len(tr.Points)
	}
	if n != 65 {
		t.Errorf("chute continue : %d points publies, attendu 65 — la fin de la chute est vraie", n)
	}
	if len(porteTraces(doc, 631)) != 0 || doc.Coverage.Tracks.HorsEmprise != 1 {
		t.Errorf("le point isole doit etre ecarte seul (horsEmprise %d)", doc.Coverage.Tracks.HorsEmprise)
	}
}
