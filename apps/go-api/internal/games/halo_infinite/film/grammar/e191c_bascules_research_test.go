//go:build research

package grammar

// e191c_bascules_research_test.go — LOT 1.9.1 bis, PAS 2 BIS : LES RECORDS QUI BASCULENT,
// UN PAR UN, AVEC LA PREUVE DE LEUR NOUVELLE LARGEUR.
//
// # POURQUOI CE FICHIER EXISTE
//
// La correction des categories de `FUN_1406d3140` fait MONTER quatre lignes du golden de
// fermeture et DESCENDRE trois. Le ratchet 0.A.3 refuse toute descente, et il a raison de le
// faire par defaut. L arbitrage du pilote (2026-09-15) l autorise ICI a une condition : que
// CHAQUE ligne qui descend soit ecrite, record par record, avec son ancien et son nouveau total
// de bits et la fonction de l ecrivain qui prouve la nouvelle largeur.
//
// # CE QUE L INSTRUMENT PROUVE, ET COMMENT
//
// UN RECORD QUI FERMAIT AVAIT, PAR DEFINITION, `EndBit == Want` : son ANCIEN total est donc
// exactement `Want`, connu sans rejouer l ancien lecteur. Son NOUVEAU total est `EndBit`, et
// l ecart `EndBit - Want` dit combien de bits la correction lui a retires. L instrument colle,
// pour chaque record qui ne ferme plus, cet ecart ET la liste des composants corriges qu il
// traverse, avec la largeur que chacun consomme. Quand l ecart s explique par la somme des
// corrections presentes — 4 bits par i10 dont la sonde bascule, 4 par i21, 5 par i22, 1 par
// i28 — la preuve est complete : le record fermait sur un total que l ecrivain contredit.
//
// LECTURE SEULE, sans garde d environnement (bobines versionnees).
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cBascules$' -v -count=1

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// e191cCorriges nomme les quatre composants dont la CATEGORIE a change le 2026-09-15, avec la
// fonction de l ecrivain qui la prouve et le nombre de bits que la correction leur retire.
var e191cCorriges = map[string]string{
	"object-parent-state-component":                    "FUN_140c1e4d0 @140c1e51d, categorie 1 -> entree 4 quand la sonde rend 1 : -4 bits",
	"equipment-activated-component":                    "FUN_140c1dc80 @140c1dcbb, categorie 4 : -4 bits",
	"equipment-control-signal-component":               "FUN_14101cd94 @14101cdc3, categorie 4, sonde supprimee : -5 bits",
	"equipment-tracked-object-handles-stack-component": "FUN_140f72dec @140f72e41, categorie 0, sonde supprimee : -1 bit",
}

// e191cBascule est un record dont l etat de fermeture a change.
type e191cBascule struct {
	Court        string
	TI, Slot     int
	Want, EndBit int
	Corriges     []string
}

// TestE191cBascules colle les records de ti=37 et ti=38 qui ne ferment PLUS (ancien total =
// `Want`) et ceux qui ferment DESORMAIS, avec les composants corriges qu ils traversent.
func TestE191cBascules(t *testing.T) {
	t.Logf("######## PAS 2 BIS — LES RECORDS QUI BASCULENT, AVEC LEUR PREUVE ########")
	var perdus, gagnes []e191cBascule
	for _, court := range closureMiniFilms() {
		p, g := e191cBasculesBobine(t, court)
		perdus, gagnes = append(perdus, p...), append(gagnes, g...)
	}
	t.Logf("")
	t.Logf("==== RECORDS QUI FERMENT DESORMAIS (%d) ====", len(gagnes))
	for _, b := range gagnes {
		t.Logf("  %s ti=%-2d slot=%-5d nouveau total = frontiere = %d bits ; corriges traverses : %s",
			b.Court, b.TI, b.Slot, b.EndBit, e191cJoin(b.Corriges))
	}
	t.Logf("")
	t.Logf("==== RECORDS QUI NE FERMENT PLUS (%d) — ANCIEN TOTAL = LA FRONTIERE ====", len(perdus))
	for _, b := range perdus {
		t.Logf("  %s ti=%-2d slot=%-5d ancien total %d bits -> nouveau %d bits (%+d)",
			b.Court, b.TI, b.Slot, b.Want, b.EndBit, b.EndBit-b.Want)
		for _, c := range b.Corriges {
			t.Logf("        %s", c)
		}
	}
}

// e191cJoin colle une liste de libelles.
func e191cJoin(l []string) string {
	if len(l) == 0 {
		return "(aucun)"
	}
	out := ""
	for i, s := range l {
		if i > 0 {
			out += " | "
		}
		out += s
	}
	return out
}

// e191cBasculesBobine rend, pour une bobine, les records perdus et gagnes de ti=37 et ti=38.
// Les records PERDUS sont ceux que le golden fige comme fermes et qui ne ferment plus ; ils se
// reconnaissent sans relire le golden : leur `Want` est l ancien total, et l ecart au nouveau
// total est exactement ce que la correction a retire.
func e191cBasculesBobine(t *testing.T, court string) (perdus, gagnes []e191cBascule) {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre %s : %v", court, err)
	}
	for _, pay := range e191cPayloads(fc) {
		for _, b := range keyframeBornes(pay) {
			if b.TI != 37 && b.TI != 38 {
				continue
			}
			tr := WalkKeyframeFullState(pay, b.Bit, reg, contexteDInstrument())
			if tr.DesyncAt >= 0 {
				continue
			}
			bas := e191cBascule{Court: court, TI: b.TI, Slot: b.Slot, Want: b.Want,
				EndBit: tr.EndBit, Corriges: e191cCorrigesDuRecord(tr)}
			if tr.EndBit == b.Want {
				gagnes = append(gagnes, bas)
				continue
			}
			if len(bas.Corriges) > 0 && tr.EndBit < b.Want {
				perdus = append(perdus, bas)
			}
		}
	}
	return perdus, gagnes
}

// e191cCorrigesDuRecord nomme les composants corriges que le record traverse, avec la largeur
// que chacun y consomme (l ecart au composant suivant).
func e191cCorrigesDuRecord(tr EntityTrace) []string {
	var out []string
	for i, c := range tr.Comps {
		why, ok := e191cCorriges[c.Name]
		if !ok {
			continue
		}
		w := tr.EndBit - c.StartBit
		if i+1 < len(tr.Comps) {
			w = tr.Comps[i+1].StartBit - c.StartBit
		}
		out = append(out, fmt.Sprintf("i%d %s : %d bits consommes — %s", c.Index, c.Name, w, why))
	}
	sort.Strings(out)
	return out
}
