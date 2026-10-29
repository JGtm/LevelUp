package grammar

// frame_closure_temoins_test.go — LES TEMOINS DECALES DE LA CARTE (`frame_closure_temoins.go`).

import "testing"

// TestBitDeDecalage : k de -8 a -1 sur les bits 0 a 7, k de 1 a 8 sur les bits 8 a 15.
func TestBitDeDecalage(t *testing.T) {
	for k, want := range map[int]int{-8: 0, -1: 7, 1: 8, 8: 15} {
		if got := BitDeDecalage(k); got != want {
			t.Errorf("BitDeDecalage(%d) = %d, attendu %d", k, got, want)
		}
	}
}

// TestTemoinsDecales : un paquet ferme dont la vue C est vide et suivie de zeros se referme depuis
// les departs decales qui tombent encore sur un zero suivi d au plus sept zeros ; un depart qui lit
// un `1` (une entree) ne le referme pas.
func TestTemoinsDecales(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(123)
	bw.finDeVueB()
	debutVueC := bw.n
	bw.bit(0)
	d := marcherUnDetaille(t, mondeDeCarte(), bw.buf)
	if !d.Fermee || d.FinVueB != debutVueC {
		t.Fatalf("paquet %+v : attendu ferme, vue C a %d", d, debutVueC)
	}
	reste := len(bw.buf)*8 - (debutVueC + 1)
	for k := 1; k <= DecalageTemoinMax; k++ {
		attendu := k <= reste // un zero lu, puis au plus sept zeros derriere
		if got := d.TemoinsDecales&(1<<uint(BitDeDecalage(k))) != 0; got != attendu {
			t.Errorf("decalage %+d : referme %t, attendu %t (reste %d)", k, got, attendu, reste)
		}
	}
	// Depart a -3 : le premier des trois zeros du terminateur de la vue B se lit comme une vue C
	// vide ; le reste derriere elle vaut alors `reste + 3` bits nuls.
	if got := d.TemoinsDecales&(1<<uint(BitDeDecalage(-3))) != 0; got != (3+reste <= bourrageMaxBits) {
		t.Errorf("decalage -3 : referme %t, reste %d", got, reste)
	}
}

// TestTemoinDecaleJugeParLesReglesDeLaVueC : un depart decale dont la vue C se lit jusqu a son
// terminateur et laisse un reste nul (ferme au bit pres) mais contredit une regle de la vue C de
// l ecrivain (`kind` 3) ne referme PAS le paquet. Le paquet ferme porte une vue C d une entree
// d index 0b11100 ; relue depuis +4, elle lit `1`, `kind` 3 (zero bit, l ecrivain reboucle), puis
// le terminateur, et quatre bits nuls derriere.
//
// MUTATION — juger le temoin sans les regles de la vue C (`return true` au lieu de
// `return j.premiere == InvariantAucun` dans [vueCFermeDepuis]) : ROUGE.
func TestTemoinDecaleJugeParLesReglesDeLaVueC(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(123)
	bw.finDeVueB()
	debutVueC := bw.n
	bw.bit(1) // une entree suit
	bw.bits(kindVueCControle, LargeurKindVueC)
	bw.bit(0)           // en-tete `FUN_1406cdc04` absent
	bw.bits(0b11100, 5) // index de controle
	bw.bits(0, 2)       // ni bloc de 0x68, ni bloc de 0xbc
	bw.bit(0)           // terminateur de la vue C
	const k = 4
	if reste := len(bw.buf)*8 - (debutVueC + k + 4); reste > bourrageMaxBits {
		t.Fatalf("construction : %d bits derriere la vue C decalee, attendu au plus %d", reste, bourrageMaxBits)
	}
	d := marcherUnDetaille(t, mondeDeCarte(), bw.buf)
	if !d.Fermee || d.FinVueB != debutVueC {
		t.Fatalf("paquet %+v : attendu ferme, vue C a %d", d, debutVueC)
	}
	br := LecteurSur(bw.buf)
	br.SetBitPos(debutVueC + k)
	c := consumeVueC(br, len(bw.buf)*8)
	if !c.Porte || !vueCFermee(bw.buf, br.BitPos()) || len(c.Kinds) != 1 || c.Kinds[0] != kindVueCNeant {
		t.Fatalf("depart %+d : %+v jusqu au bit %d, attendu `kind` 3 ferme au bit pres", k, c, br.BitPos())
	}
	if d.TemoinsDecales&(1<<uint(BitDeDecalage(k))) != 0 {
		t.Errorf("decalage %+d : la vue C contredit l ecrivain (`kind` 3), elle ne referme pas le paquet", k)
	}
}
