package grammar

// etat_de_creation_test.go — UN RECORD NEW DONT LE LECTEUR D ETAT DU JEU ECHOUE N EST PAS LU
// (`etat_de_creation.go`). Les vecteurs suivent `FUN_14080cfe8` au decoupage par defaut (9/5) :
// le compte R(3) superieur a quatre fait echouer le bloc (`CMP ECX,0x4 ; JA` @14080d238), qui lit
// quand meme `FUN_14080d4d0` et la queue G3.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// blocMPP ecrit un bloc `object-multiplayer-properties` au decoupage par defaut, toutes portes a
// zero sauf celle du « variant-name » (lu par recherche, 0 bit), avec le compte `compte` ; les
// entrees de la liste ne sont ecrites que si le lecteur du jeu les lit (compte <= 4, la valeur de
// l instruction, ecrite ici en litteral pour que le vecteur ne suive pas la constante du port).
func (w *bitWriter) blocMPP(compte uint64) {
	w.bits(0, 9)  // FUN_141fd72c0
	w.bits(0, 32) // FUN_14080d6f0
	w.bit(1)      // « variant-name » par recherche : 0 bit
	w.bit(0)      // porte du R(18)
	w.bit(0)      // FUN_14080d524
	w.bits(0, 2)  // R(2)
	w.bits(0, 5)  // index R(5)
	w.bits(compte, 3)
	if compte <= 4 {
		for range compte {
			w.bits(0, 5) // R(5)
			w.bit(0)     // FUN_14080d69c : porte a 0
		}
	}
	w.bit(0) // FUN_14080d4d0
	w.bit(0) // porte G3
}

// TestLeBlocMPPEchoueAuDelaDeQuatre : pour chaque compte de 0 a 7, le verdict du bloc est celui de
// `FUN_14080cfe8` et le lecteur consomme exactement les bits du jeu — la liste n est lue que si le
// compte est accepte, la suite toujours. MUTATIONS : `mppCompteMax = 5` ou `count < mppCompteMax`
// (seuil change), ROUGE ; `lisible := true` (regle retiree), ROUGE.
func TestLeBlocMPPEchoueAuDelaDeQuatre(t *testing.T) {
	for compte := range uint64(8) {
		var bw bitWriter
		bw.blocMPP(compte)
		fin := bw.n
		bw.bits(0x2a, 8) // la suite du record : un motif que le lecteur ne doit pas lire
		br := LecteurSur(bw.buf)
		lisible := consumeMultiplayerPropertiesBlock(br)
		if want := compte <= 4; lisible != want {
			t.Errorf("compte %d : lisible %v, attendu %v", compte, lisible, want)
		}
		if br.BitPos() != fin {
			t.Errorf("compte %d : %d bits lus, attendu %d", compte, br.BitPos(), fin)
		}
	}
}

// neuf13Etat ecrit l en-tete d un record NEW (slot R(13), generation, archetype `ti`), l etat par
// defaut de `ti=36` (prefixe de version a 0, bloc MPP au compte `compte`), puis porte et masque
// vides.
func (w *bitWriter) neuf13Etat(slot uint32, ti, compte uint64) {
	w.bit(0)
	w.bits(recNew, 2)
	w.bits(uint64(slot), 13)
	w.bits(1, 2)
	w.bits(ti, 6)
	w.bit(0) // prefixe de version de FUN_1407f2224
	w.blocMPP(compte)
	w.bit(0)     // porte du record NEW
	w.bit(0)     // masque clairseme
	w.bits(0, 3) // aucun composant
}

// mondeAEtats rend le monde de [mondeDeCarte] dont le registre va jusqu a l archetype 41.
func mondeAEtats() *World {
	arch := make([]Archetype, ProjectileTypeIndex+1)
	for i := range arch {
		arch[i] = Archetype{Index: i}
	}
	w := NewWorld(&Registry{Archetypes: arch})
	w.BindImageCle(1, 123, 4)
	w.BindImageCle(1, 124, 4)
	return w
}

// TestUnNeufDontLEtatEchoueNEstPasLu : un record NEW `ti=36` au compte MPP 5 arrete la marche a la
// fin de son etat (`FUN_1408f1aa4` ne lit pas le corps), ne lie pas son slot, et le juge le
// contredit ; son temoin au compte 4 se lit, se lie, et la marche continue sur le delta qui suit.
// MUTATION : retirer `lireLeBlocMPPDeLEtat` de [consumeDefaultStateTI36] (regle retiree), ROUGE.
func TestUnNeufDontLEtatEchoueNEstPasLu(t *testing.T) {
	for _, compte := range []uint64{4, 5} {
		var bw bitWriter
		bw.neuf13Etat(300, 36, compte)
		finEtat := bw.n - 5 // porte et masque vide
		bw.delta13(124)
		bw.finDeVueB()
		w := mondeAEtats()
		recs, _ := DecodeFrameInfer(bw.buf, w, cadreDeTete)
		_, lie := w.ArchetypeForSlot(300)
		juge := jugerLePaquet(recs, false, FluxVueC{}, true)
		if compte <= 4 {
			if len(recs) != 2 || recs[0].DesyncAt != -1 || recs[0].Trace.EtatIllisible || !lie ||
				juge.premiere != InvariantAucun {
				t.Errorf("temoin, compte %d : %d records, %+v, lie %v, regle %v", compte, len(recs), recs, lie, juge.premiere)
			}
			continue
		}
		if len(recs) != 1 || !recs[0].Trace.EtatIllisible || recs[0].DesyncAt != 0 || recs[0].FinBit != finEtat {
			t.Fatalf("compte %d : %+v, attendu un seul record arrete au bit %d", compte, recs, finEtat)
		}
		if lie {
			t.Error("le slot 300 est lie : le jeu ne cree pas l entite d un record dont l etat echoue")
		}
		if juge.premiere != InvariantEtatDeCreationIllisible {
			t.Errorf("regle %v, attendu %v", juge.premiere, InvariantEtatDeCreationIllisible)
		}
	}
}

// etatTI41 ecrit le debut de l etat de `ti=41` jusqu a `FUN_1406d00ec` : version absente, bloc MPP
// au compte `compte`, porte du R(5) a 0, drapeau 2, et sous lui `FUN_1408f0ac4` absent puis la
// porte de l index (`absent` : 1, pas de R(2)).
func etatTI41(compte uint64, drapeau2, absent bool) []byte {
	var bw bitWriter
	bw.bit(0)
	bw.blocMPP(compte)
	bw.bit(0)
	if drapeau2 {
		bw.bit(1)
		bw.bit(0)
		if absent {
			bw.bit(1)
		} else {
			bw.bit(0)
			bw.bits(0, 2)
		}
	} else {
		bw.bit(0)
	}
	bw.bits(0, 256) // la suite de l etat, lue a zero
	return bw.buf
}

// TestLEtatDuProjectileSuitSonPropreVerdict : `FUN_1408efb58` ignore l echec du bloc MPP quand le
// drapeau 2 est pose et que `FUN_1406d00ec` rend 0xffffffff (son verdict vaut alors un predicat
// hors du film) ; dans tous les autres cas, le bloc qui echoue fait echouer l etat. MUTATION :
// retirer `&& !indexAbsent`, ROUGE au cas d exception.
func TestLEtatDuProjectileSuitSonPropreVerdict(t *testing.T) {
	cas := []struct {
		nom              string
		compte           uint64
		drapeau2, absent bool
		illisible        bool
	}{
		{"compte 4 (temoin)", 4, false, false, false},
		{"compte 5, sans drapeau 2", 5, false, false, true},
		{"compte 5, drapeau 2, index lu", 5, true, false, true},
		{"compte 5, drapeau 2, index absent", 5, true, true, false},
	}
	for _, c := range cas {
		br := LecteurSur(etatTI41(c.compte, c.drapeau2, c.absent))
		consumeDefaultStateTI41(br)
		if br.etatIllisible != c.illisible {
			t.Errorf("%s : etat illisible %v, attendu %v", c.nom, br.etatIllisible, c.illisible)
		}
	}
}

// appelDuBlocMPP : un appel du lecteur du bloc MPP ; l ecriture par classe de caractere evite que
// ce fichier se denonce lui-meme.
var appelDuBlocMPP = regexp.MustCompile(`consumeMultiplayerPropertiesBloc[k][(]`)

// TestLeBlocMPPDUnEtatPasseParSonVerdict : GARDE-RAIL de la regle 6. Un lecteur d etat de creation
// lit le bloc MPP par [lireLeBlocMPPDeLEtat], qui fait echouer l etat quand le bloc echoue ; un
// appel direct oublierait le verdict. Hors de l hote, seuls l etat de `ti=41` (verdict a exception)
// et les instruments de recherche (*_research_test.go, hors production) l appellent.
func TestLeBlocMPPDUnEtatPasseParSonVerdict(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("aucun fichier Go vu (%v) : le garde-rail ne garde rien", err)
	}
	permis := map[string]int{"etat_de_creation.go": 1, "default_state_ti41.go": 1, "default_state.go": 1}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f) //nolint:gosec // fichiers du paquet lui-meme
		if err != nil {
			t.Fatalf("lecture de %s : %v", f, err)
		}
		if n := len(appelDuBlocMPP.FindAll(data, -1)); n != permis[f] {
			t.Errorf("%s : %d appel(s) du bloc MPP, attendu %d — passer par lireLeBlocMPPDeLEtat", f, n, permis[f])
		}
	}
}

// TestLeVerdictDEtatEstCeluiDuRecord : un lecteur dont un etat de creation a echoue HORS d une
// traversee (le lecteur d etat appele seul, comme la marche de creation d equipement) traverse
// ensuite un record NEW lisible : son verdict est celui de CE record. MUTATION : retirer la remise
// a faux de `etatIllisible` dans [TraverseEntity], ROUGE.
func TestLeVerdictDEtatEstCeluiDuRecord(t *testing.T) {
	var bw bitWriter
	bw.bit(0) // prefixe de version de FUN_1407f2224
	bw.blocMPP(5)
	bw.bits(36, 6) // le record NEW suivant : archetype 36, etat lisible
	bw.bit(0)
	bw.blocMPP(4)
	bw.bit(0)     // porte du record NEW
	bw.bit(0)     // masque clairseme
	bw.bits(0, 3) // aucun composant
	br := LecteurSur(bw.buf)
	consumeDefaultStateTI36(br)
	if !br.etatIllisible {
		t.Fatal("l etat au compte 5 n a pas echoue : le vecteur ne teste rien")
	}
	tr := TraverseEntity(br, mondeAEtats().Reg, 0)
	if tr.EtatIllisible || tr.DesyncAt != -1 {
		t.Errorf("record lisible juge illisible (%v, desync %d) : le verdict de l etat precedent a survecu",
			tr.EtatIllisible, tr.DesyncAt)
	}
}
