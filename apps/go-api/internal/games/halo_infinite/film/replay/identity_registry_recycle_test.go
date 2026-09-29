package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// identity_registry_recycle_test.go — UN SIEGE RECYCLE N'EST PAS UN SIEGE QUI CHANGE DE PORTEUR (lot
// R2 du plan de suite de l'audit du decodeur, 2026-09-28, constat C3 du G-corpus J11.1).
//
// Le verdict du pont refusait de publier (« non publiable : un slot change de porteur ») des qu'un
// slot portait deux joueurs nommes — regle posee quand seule la generation 1 etait lue. Depuis J5.2,
// les corps de generation >= 2 ont des positions : 66 slots de `4f77afc1` et 114 de `084a804d`
// portaient deux corps SUCCESSIFS de deux joueurs, sans aucune collision reelle.

// TestPont_SiegeRecycleNEstPasUneCollision : deux corps etablis distincts, deux joueurs. Le verdict
// ne compte aucune collision ; le pont APLATI se tait sur le slot, et la lecture a l'instant
// repond PAR CORPS, meme hors de toute vie couvrante.
//
// ROUGE AVANT : `slotCollisions` = 1, verdict « un slot change de porteur », et `XUIDNumAt` muet a
// 13 s (le slot etait ambigu : ni vie couvrante, ni pont).
//
// MUTATIONS : faire rendre `collision` par `classerLeSlot` pour deux corps distincts -> rouge ;
// retirer la lecture par corps de `xuidNumAt` -> rouge a 13 s ; retirer `SlotRecycle` de
// `pontAplatiMuet` -> `PontDeSlot` sert le premier occupant, rouge.
func TestPont_SiegeRecycleNEstPasUneCollision(t *testing.T) {
	in := entreeSlotRecycle(1)
	in.PlayerIndices.ByXUID = map[uint64]int{111: 0, 222: 1, 333: 2}
	fb := fallback.NouveauCompteur()
	in.Fallbacks = fb
	reg := BuildIdentityRegistry(in)
	if n := reg.CollisionsDeSlot(); n != 0 {
		t.Fatalf("slotCollisions = %d, attendu 0 : deux corps etablis distincts ne sont pas une collision", n)
	}
	if v := verdictOfBridge(reg.SanteDuPont()); v == "non publiable : un slot change de porteur" {
		t.Fatalf("verdict du pont %q sur un siege recycle", v)
	}
	if !reg.SlotsAmbigus()[100] {
		t.Fatal("le slot recycle 100 doit rester muet pour les lecteurs du pont APLATI")
	}
	if got := reg.PontDeSlot(100); got != "" {
		t.Fatalf("PontDeSlot(100) = %q : le pont aplati servirait le premier occupant aux deux corps", got)
	}
	if _, ok := reg.PontEpure()[100]; ok {
		t.Fatal("PontEpure porte le slot recycle 100")
	}
	cas := []struct {
		tUS  uint64
		want uint64
	}{
		{2_000_000, 111},  // vie du corps de generation 1
		{15_000_000, 222}, // vie du corps de generation 2
		{13_000_000, 222}, // entre la creation du corps 2 (12 s) et sa premiere position : son corps
	}
	for _, c := range cas {
		if got := reg.XUIDNumAt(100, c.tUS); got != c.want {
			t.Errorf("XUIDNumAt(100, %d) = %d, attendu %d (lecture par corps)", c.tUS, got, c.want)
		}
	}
	if got := reg.XUIDNumAt(100, 100_000); got != 0 {
		t.Errorf("XUIDNumAt(100, 0,1 s) = %d, attendu 0 : avant le premier record, aucun corps etabli", got)
	}
}

// TestPont_DeuxJoueursDansUnMemeCorpsRestentUneCollision : la regle du verdict ne s'affaiblit pas —
// deux joueurs nommes dans des vies du MEME corps (ou d'un corps que rien n'etablit) restent une
// collision, et le pont s'y tait.
func TestPont_DeuxJoueursDansUnMemeCorpsRestentUneCollision(t *testing.T) {
	lives := []lifeSpan{
		{slot: 100, from: 1_000_000, to: 4_000_000, xuid: 111},
		{slot: 100, from: 14_000_000, to: 18_000_000, xuid: 222},
	}
	index := map[uint64]int{111: 0, 222: 1}
	unCorps := corpsParSlot([]grammar.BipedCreation{creationGen(100, 500_000, 0, 1)})
	deuxCorps := corpsParSlot([]grammar.BipedCreation{
		creationGen(100, 500_000, 0, 1), creationGen(100, 12_000_000, 1, 2)})
	for _, c := range []struct {
		nom                string
		corps              map[uint32]corpsLu
		collision, recycle bool
	}{
		{"un seul corps", unCorps, true, false},
		{"aucun corps connu", nil, true, false},
		{"deux corps distincts", deuxCorps, false, true},
	} {
		_, _, amb, rec := ownersFromLives(lives, index, c.corps)
		if amb[100] != c.collision || rec[100] != c.recycle {
			t.Errorf("%s : collision %v recycle %v, attendu %v / %v", c.nom, amb[100], rec[100],
				c.collision, c.recycle)
		}
	}
}
