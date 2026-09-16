//go:build research

package filmdec

// e191c_premierbit_research_test.go — LOT 1.9.1 bis, PAS 2 QUATER : LOCALISER LE PREMIER BIT
// FAUX, AU LIEU DE VERIFIER COMPOSANT PAR COMPOSANT.
//
// # LA METHODE
//
// Verifier les composants un a un a rendu quinze grammaires conformes et zero cause. La mesure
// du pas 1 dit pourtant qu i9 `object-multiplayer-properties` consomme 1,5 MILLION de bits en
// moyenne sur ti=37 : c est la signature d un COMPTEUR DE BOUCLE lu sur des bits deja
// desalignes, pas d une largeur fausse. Donc le premier composant IMPLAUSIBLE borne le defaut
// par le haut, et le fautif est celui qui le precede — ou le cadre commun.
//
// # CE QUE L INSTRUMENT JOURNALISE
//
// Pour un echantillon de records (deux bobines, N records par archetype), composant
// par composant DANS L ORDRE DU REGISTRE : la position de bit d entree, la largeur consommee,
// et LES COMPTEURS DE BOUCLE lus en tete des composants qui en portent un (i6, i7, i8, i15,
// i17). Un compteur au-dela de ce que le jeu declare valide, ou une largeur qui depasse la
// taille du record, est marque `!!` — c est le point de rupture.
//
// Les deux mots de taille `n1` et `n2` sont journalises AUSSI : `FUN_142e2bfd0` ne lit l etat
// par defaut que si `n1 > 0` et ne lance la boucle de composants que si `n2 > 0` (relu le
// 2026-09-15, lignes `if (0 < (int)uVar7)` avant `vtable[0x60]` et avant `vtable[0x88]`).
//
// LECTURE SEULE, sans garde d environnement (bobines versionnees).
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cPremierBitFaux$' -v -count=1

import (
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// e191cEchantillon : combien de records par (bobine, archetype) le journal detaille.
const e191cEchantillon = 12

// e191cRegionCountMax est la borne de VALIDITE que le jeu lui-meme applique au compteur de
// regions d i6 (`140e1bfa0` : `bVar14 = bVar10 < 0x21`). Au-dela, le jeu marque le composant
// invalide — donc un compteur superieur est la preuve d une lecture sur des bits de bruit.
const e191cRegionCountMax = 32

// e191cCompteur rend le compteur de boucle lu en tete d un composant, et vrai s il en porte un.
func e191cCompteur(pay []byte, nom string, start int) (uint64, bool) {
	switch nom {
	case "object-region-state-component": // R(1) present + R(6) compte
		return kfReadBits(pay, start+1, 6), true
	case "object-damage-sections-component": // R(6) compte
		return kfReadBits(pay, start, 6), true
	case "object-constraint-component": // R(5) n
		return kfReadBits(pay, start, 5), true
	case "object-low-frequency-component": // R(2) tete ; compte a +12 ou +27
		if kfReadBits(pay, start, 2) < 2 {
			return kfReadBits(pay, start+27, 6), true
		}
		return kfReadBits(pay, start+12, 6), true
	case "object-frame-configuration-component": // porte R(1) ; si 1 : R(32) puis R(6)
		if kfReadBits(pay, start, 1) == 1 {
			return kfReadBits(pay, start+33, 6), true
		}
		return 0, false
	}
	return 0, false
}

// TestE191cPremierBitFaux journalise l echantillon.
func TestE191cPremierBitFaux(t *testing.T) {
	t.Logf("######## PAS 2 QUATER — LE PREMIER BIT FAUX, RECORD PAR RECORD ########")
	bobines := closureMiniFilms()
	if len(bobines) > 2 {
		bobines = bobines[:2]
	}
	for _, court := range bobines {
		for _, ti := range []int{14, 17, 13, 37, 38} { // deux qui FERMENT, trois qui echouent
			e191cJournalBobine(t, court, ti)
		}
	}
}

// e191cJournalBobine detaille les premiers records d un archetype sur une bobine.
func e191cJournalBobine(t *testing.T, court string, ti int) {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre %s : %v", court, err)
	}
	t.Logf("")
	t.Logf("==== %s ti=%d ====", court, ti)
	vus := 0
	for _, pay := range e191cPayloads(fc) {
		for _, b := range keyframeBornes(pay) {
			if b.TI != ti || vus >= e191cEchantillon {
				continue
			}
			vus++
			e191cJournalRecord(t, pay, reg, b, vus)
		}
	}
	t.Logf("  (%d records detailles)", vus)
}

// e191cJournalRecord colle UN record : ses deux mots de taille, puis chaque composant.
func e191cJournalRecord(t *testing.T, pay []byte, reg *Registry, b keyframeBorne, n int) {
	t.Helper()
	n1 := kfReadBits(pay, b.Bit+keyframeFullStateHeaderBits, 32)
	// n2 se lit APRES l etat par defaut : on rejoue le bloc pour le localiser.
	br2 := NewBitReader(pay)
	br2.SetBitPos(b.Bit + keyframeFullStateHeaderBits)
	br2.ReadBits(keyframeFullStateSizeBits)
	if int32(n1) > 0 { //nolint:gosec // 32 bits
		consumeKeyframeDefaultState(br2, uint32(kfReadBits(pay, b.Bit+keyframeRecordTIBit, 6))) //nolint:gosec // 6 bits
	}
	n2 := kfReadBits(pay, br2.BitPos(), 32)
	tr := WalkKeyframeFullState(pay, b.Bit, reg, profilDInstrument)
	taille := b.Want - b.Bit
	t.Logf("  -- record %d slot=%d bit=%d taille=%d bits, fin lue=%d, ecart=%+d, n1=%d, n2=%d",
		n, b.Slot, b.Bit, taille, tr.EndBit, tr.EndBit-b.Want, n1, int32(n2)) //nolint:gosec // 32 bits
	for i, c := range tr.Comps {
		w := tr.EndBit - c.StartBit
		if i+1 < len(tr.Comps) {
			w = tr.Comps[i+1].StartBit - c.StartBit
		}
		marque := ""
		if w > taille {
			marque = "  !! largeur > taille du record"
		}
		det := ""
		if cnt, ok := e191cCompteur(pay, c.Name, c.StartBit); ok {
			det = fmt.Sprintf(" compteur=%d", cnt)
			if cnt > e191cRegionCountMax {
				marque = "  !! compteur au-dela de la borne de validite du jeu"
			}
		}
		t.Logf("     i%-2d %-52s entree=%-7d largeur=%-8d%s%s",
			c.Index, c.Name, c.StartBit-b.Bit, w, det, marque)
	}
}

// LE RESULTAT DU PAS 2 QUATER, ET IL RETOURNE LE LOT (2026-09-15).
//
// `n1` se lit a une position FIXE (108 bits) et `n2` APRES l etat par defaut. Les deux sont des
// tailles de tampon, donc CONSTANTES par archetype et par build. Mesure sur les deux bobines :
//
//	ti=14  n1=4   n2=28      constants   -> ferme 100 %
//	ti=17  n1=4   n2=432     constants   -> ferme 100 %
//	ti=22  n1=12  n2=12      constants   -> ferme 100 %
//	ti=29  n1=1   n2=256     constants   -> ferme 92,7 %
//	ti=6   n1=4   n2=7896    constants   -> ferme 90,1 %
//	ti=13  n1=136 n2 = BRUIT (-1 x48, 2147483392 x24, 32768, 98304, 0, 68, 422710486, ...)
//	ti=37  n1=100 n2 = BRUIT (-1073741824 x222, -1744002798, 1308139706, ...)
//	ti=38  n1=100 n2 = BRUIT (0, 110, 14112, 14113, 14114, 3528, 3612672, 451709, ...)
//
// `n1` est CONSTANT PARTOUT : l en-tete de 108 bits est donc exact, y compris sur les
// archetypes qui ne ferment pas. `n2` separe les deux populations sans exception. Or `n2` se
// lit juste apres l ETAT PAR DEFAUT : **le premier bit faux de ti=13, 37 et 38 est DANS LEUR
// ETAT PAR DEFAUT, avant le premier composant.** Relire quinze composants ne pouvait rien
// donner — le defaut est en amont d eux.
//
// CE QUE CELA DIT DU GARDE-RAIL EXISTANT : `default_state_n2_constant_test.go` porte exactement
// cet oracle, et il ECARTE de son jugement « les archetypes a etat VARIABLE (ti=3, 8, 10, 13,
// 24, 36..39, 48, le bipede) ». Il etait donc desarme PRECISEMENT sur les archetypes qui
// echouent. La suite du chantier est la : faire juger ces archetypes-la par `n2`.
