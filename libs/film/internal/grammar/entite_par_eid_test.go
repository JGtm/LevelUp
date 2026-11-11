package grammar

// entite_par_eid_test.go — LE TYPE D UNE ENTITE EST LIE A SON EID COMPLET (plan marche complete,
// A2.T ; jeu : `FUN_1406cbaa0` cas 3 et `FUN_1406caad8`).
//
// La garde de delta du jeu compare `*(uint *)(slot * 200 + t)` a l eid ENTIER : le slot ET les deux
// bits de generation. Une creation ecrit l eid et le type de l entree, une reallocation du slot
// sous une autre generation les remplace, et un delta dont l eid ne vaut pas celui de l entree
// rend le code 3 sans lire un seul bit de corps. Chaque cas fabrique le paquet que l ecrivain pose
// et exige la lecture du jeu au bit pres : sortie de la vue B, eid rejete, type du record lu.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// neuf13Gen ecrit un record NEW d un archetype sans composant : comme [bitWriter.neuf13], sous la
// generation `gen`.
func (w *bitWriter) neuf13Gen(slot uint32, gen uint64, ti uint64) {
	w.bit(0)
	w.bits(recNew, 2)
	w.bits(uint64(slot), 13)
	w.bits(gen, 2)
	w.bits(ti, 6)
	w.bit(0)     // porte du record NEW
	w.bit(0)     // masque clairseme
	w.bits(0, 3) // aucun composant
}

// delta13Gen ecrit un record DELTA vide : comme [bitWriter.delta13], sous la generation `gen`.
func (w *bitWriter) delta13Gen(slot uint32, gen uint64) {
	w.bit(1)
	w.bits(uint64(slot), 13)
	w.bits(gen, 2)
	w.bit(0)
	w.bit(0)
	w.bits(0, 3)
}

// monde5Archetypes rend un monde sans liaison dont les archetypes 0 a 4 n ont aucun composant.
func monde5Archetypes() *World {
	return NewWorld(&Registry{Archetypes: []Archetype{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}, {Index: 4}}})
}

// paquetDeDeltas range un paquet de tete, des deltas `(slot, generation)` dans l ordre, la fin de
// la vue B et une vue C a une entree, sur le monde `w`.
func paquetDeDeltas(w *World, ids ...[2]uint64) *lecture.Paquet {
	var bw bitWriter
	bw.teteDePaquet()
	for _, id := range ids {
		bw.delta13Gen(uint32(id[0]), id[1]) //nolint:gosec // slot de test
	}
	bw.finDeVueB()
	bw.vueCUneEntree()
	return rangerUn(w, bw.buf)
}

// exigerLue dit qu un record DELTA est lu sous le type `ti`, sans desynchronisation.
func exigerLue(t *testing.T, r lecture.Record, ti int16, slot uint32, gen uint32) {
	t.Helper()
	if r.Genre != lecture.GenreDelta || r.TI != ti || r.Desync != lecture.SansDesynchronisation ||
		r.Vie.Slot != slot || r.Vie.Gen != gen {
		t.Errorf("record %+v, attendu un DELTA du slot %d generation %d lu sous ti=%d", r, slot, gen, ti)
	}
}

// Une creation lue puis une mise a jour du meme eid : le delta est lu sous le type de la creation.
func TestUneCreationPuisUneMiseAJourDeSonEid(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.neuf13Gen(200, 2, 2) // la creation : slot 200, generation 2, archetype 2
	bw.delta13Gen(200, 2)   // sa mise a jour
	bw.finDeVueB()
	bw.vueCUneEntree()
	w := monde5Archetypes()
	p := rangerUn(w, bw.buf)

	if p.VueB.Sortie != lecture.SortieTerminateur || len(p.Records) != 2 {
		t.Fatalf("vue B %+v, %d records : attendu une sortie par le terminateur et deux records", p.VueB, len(p.Records))
	}
	exigerLue(t, p.Records[1], 2, 200, 2)
	if ti, lie := w.ArchetypeForEID(2<<30 | 200); !lie || ti != 2 {
		t.Errorf("eid generation 2 lie a %d (%v), attendu 2", ti, lie)
	}
}

// Un slot supprime puis reoccupe sous une autre generation : la mise a jour de l ANCIENNE
// generation est rejetee sans qu un bit de corps soit lu, celle de la nouvelle est lue sous le type
// de la nouvelle. Sans le DEL (la creation seule remplace l entree), meme verdict.
func TestSuppressionPuisReallocationSousUneAutreGeneration(t *testing.T) {
	for _, avecDel := range []bool{true, false} {
		w := monde5Archetypes()
		w.BindFull(1<<30|200, 4) // la creation de l ancienne generation
		if avecDel {
			w.Unbind(200) // son DEL, lu
		}
		w.BindFull(2<<30|200, 3) // la reallocation : generation 2, archetype 3

		ancienne := paquetDeDeltas(w, [2]uint64{200, 1})
		if ancienne.VueB.Sortie != lecture.SortieRejetHorsDatum || ancienne.VueB.EIDRejete != 1<<30|200 ||
			len(ancienne.Records) != 0 {
			t.Errorf("DEL lu %v : vue B %+v, %d records, attendu l eid de generation 1 rejete hors datum, "+
				"aucun record", avecDel, ancienne.VueB, len(ancienne.Records))
		}
		nouvelle := paquetDeDeltas(w, [2]uint64{200, 2})
		if nouvelle.VueB.Sortie != lecture.SortieTerminateur || len(nouvelle.Records) != 1 {
			t.Fatalf("DEL lu %v : vue B %+v, %d records, attendu la generation 2 lue", avecDel, nouvelle.VueB,
				len(nouvelle.Records))
		}
		exigerLue(t, nouvelle.Records[0], 3, 200, 2)
	}
}

// Une mise a jour d une generation dont la creation n a jamais ete lue : le slot est lie sous
// l ancienne, le jeu ne lit pas le corps (le type n est pas celui de l ancienne), et la sortie par
// rejet se compte. Aucun type n est devine.
func TestUneGenerationJamaisCreeeNEstPasLueSousLeTypeDeLAncienne(t *testing.T) {
	w := monde5Archetypes()
	w.BindFull(1<<30|200, 4)
	cfg := cadreDeCarte()
	cfg.Obs = NouvelleObservation()

	p := paquetDeDeltas(w, [2]uint64{200, 1}, [2]uint64{200, 3})
	if p.VueB.Sortie != lecture.SortieRejetHorsDatum || p.VueB.EIDRejete != 3<<30|200 {
		t.Fatalf("vue B %+v, attendu l eid de generation 3 rejete hors datum", p.VueB)
	}
	if len(p.Records) != 1 {
		t.Fatalf("%d records, attendu le seul delta de la generation 1", len(p.Records))
	}
	exigerLue(t, p.Records[0], 4, 200, 1)
	if _, lie := w.ArchetypeForEID(3<<30 | 200); lie {
		t.Error("l eid de generation 3 est lie : un type a ete devine")
	}
	if _, rejete := rejetDeVue(recDelta, 3<<30|200, w, cfg); !rejete || cfg.Obs.RejetsHorsDatum != 1 {
		t.Errorf("rejet %v, compteur %d : attendu un rejet hors datum compte une fois", rejete, cfg.Obs.RejetsHorsDatum)
	}
}

// Une liaison de generation inconnue (image-cle, table de datums) repond a toute generation de
// son slot : le jeu y lit l eid de l entree, que l image-cle ne porte pas sous la meme forme.
func TestUneLiaisonDeGenerationInconnueRepondATouteGeneration(t *testing.T) {
	w := monde5Archetypes()
	w.BindImageCle(1, 200, 4)
	w.BindDatum(201, 3)
	for gen := uint64(0); gen < 4; gen++ {
		if _, lie := w.ArchetypeForEID(uint32(gen<<30 | 200)); !lie { //nolint:gosec // generation de test
			t.Errorf("generation %d : la liaison d image-cle ne repond pas", gen)
		}
		if _, lie := w.ArchetypeForEID(uint32(gen<<30 | 201)); !lie { //nolint:gosec // idem
			t.Errorf("generation %d : la liaison de datum ne repond pas", gen)
		}
	}
	if _, lie := w.ArchetypeForEID(1<<30 | 202); lie {
		t.Error("un slot sans liaison repond")
	}
}

// Le repli d anticipation suit la regle : il lie un eid que le monde ne connait pas, y compris
// quand le slot est lie sous une autre generation (sa liaison est alors remplacee), jamais un eid
// deja lie, jamais un eid que la table ne declare pas.
func TestLAnticipationLieUnEidNonLieMemeSurUnSlotLieSousUneAutreGeneration(t *testing.T) {
	w := monde5Archetypes()
	tab := NouvelleTableAnticipee()
	tab.entrees[cleAnticipee{slot: 200, tete: 2}] = []declarationAnticipee{{chunk: 5, ti: 3}}
	w.PoserTableAnticipee(tab)
	w.PoserChunkCourant(2)
	w.BindFull(1<<30|200, 4) // le slot est lie sous la generation 1

	if _, ok := w.LierParRepliDAnticipation(1<<30 | 200); ok {
		t.Error("l eid de generation 1 est deja lie : rien a anticiper")
	}
	if ti, ok := w.LierParRepliDAnticipation(3<<30 | 200); ok {
		t.Errorf("l eid de generation 3 est lie a %d : la table ne le declare pas", ti)
	}
	if ti, ok := w.LierParRepliDAnticipation(2<<30 | 200); !ok || ti != 3 {
		t.Fatalf("anticipation %d (%v), attendu 3 : la table declare la generation 2", ti, ok)
	}
	if ti, lie := w.ArchetypeForEID(2<<30 | 200); !lie || ti != 3 {
		t.Errorf("eid de generation 2 lie a %d (%v), attendu 3 : la liaison de l ancienne est remplacee", ti, lie)
	}
}
