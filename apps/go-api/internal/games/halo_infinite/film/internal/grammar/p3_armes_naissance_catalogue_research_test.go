//go:build research

package grammar

// p3_armes_naissance_catalogue_research_test.go — LA LECTURE PAR CATALOGUE de la sonde P3 (voir
// l en-tete de `p3_armes_naissance_research_test.go`).
//
// La traversee de production arrive DESALIGNEE sur i43 dans le record NEW du bipede (P3.3/P3.6) ;
// la question de la sonde n est pas la grammaire des composants i1..i42 (c est M3.2) mais : le
// record porte-t-il les armes de naissance, et les lit-on juste ? Mesure SANS oracle de vie : dans
// le record trouve, le bit `o` de [en-tete + 1000, en-tete + 1700) ou l enchainement des
// emplacements annonces au masque par le deserialiseur de production
// (`consumeWeaponStateTypeInfoVariant`) donne au moins une famille presente et UNIQUEMENT des
// familles du CATALOGUE du film (`weaponLabels` du document, toutes armes du match confondues —
// jamais l oracle de la vie). La lecture ainsi obtenue est ensuite confrontee aux oracles, au temoin
// et au hasard. La distribution de `o - en-tete` est publiee : elle dit si la position est fixe.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
)

// p3Catalogue rend les familles (moities hautes, 8 chiffres) que le document nomme.
func p3Catalogue(t *testing.T, chemin string) map[uint32]bool {
	t.Helper()
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("document : %v", err)
	}
	var d struct {
		WeaponLabels map[string]json.RawMessage `json:"weaponLabels"`
	}
	if err := json.Unmarshal(blob, &d); err != nil {
		t.Fatalf("document : %v", err)
	}
	out := map[uint32]bool{}
	for k := range d.WeaponLabels {
		if len(k) == 10 {
			out[p3Hex(k)] = true
		}
	}
	return out
}

// p3LireParCatalogue remplace la lecture de production des records trouves par la lecture a la
// position localisee par catalogue, et publie la distribution des positions.
func p3LireParCatalogue(t *testing.T, tc t516Temoin, vies []*p3Vie, origine int64, cat map[uint32]bool) {
	t.Helper()
	paquets := p3Paquets(tc, origine)
	pay := map[[2]uint64][]byte{}
	for _, q := range paquets {
		pay[[2]uint64{uint64(q.chunk), q.pk.TimestampUS}] = q.pay //nolint:gosec // chunk positif
	}
	rel, horsCat := map[int]int{}, map[uint32]int{}
	var records, lus int
	var autreQuePremier int
	for _, v := range vies {
		if len(v.candidats) == 0 {
			continue
		}
		records++
		v.trouvee = nil // la vie n est lue que si un en-tete de sa fenetre se lit par catalogue
		for i := range v.candidats {
			n := &v.candidats[i]
			b := pay[[2]uint64{uint64(n.chunk), n.ts}] //nolint:gosec // chunk positif
			armes, meilleur := p3ScanCatalogue(b, n, tc.cfg, cat)
			if meilleur < 0 {
				continue
			}
			n.armes = armes
			v.trouvee = n
			if i > 0 {
				autreQuePremier++
			}
			rel[meilleur-n.bit]++
			lus++
			for _, a := range n.armes {
				if a.horsCat {
					horsCat[a.hi]++
				}
			}
			break
		}
	}
	t.Logf("   vies dont l en-tete lu n est PAS le premier de la fenetre : %d", autreQuePremier)
	cles := make([]int, 0, len(rel))
	for c := range rel {
		cles = append(cles, c)
	}
	sort.Ints(cles)
	var s string
	for _, c := range cles {
		s += fmt.Sprintf(" +%d:%d", c, rel[c])
	}
	t.Logf("== P3.8 LECTURE PAR CATALOGUE (%d familles au catalogue) : %d/%d records lus ; position de la "+
		"porte du 1er emplacement / en-tete :%s ; valeurs presentes HORS catalogue (exclues) %v", len(cat), lus,
		records, s, p3Hexes(horsCat))
}

// p3ScanDebut / p3ScanFin bornent la lecture par catalogue, relativement a l en-tete du record NEW.
// Mesure (trois films) : la porte du premier emplacement tombe dans [+1311, +1388]. Une fenetre plus
// large ([+200, +4000) au premier essai) laisse un en-tete FORTUIT lire les armes du record NEW
// VOISIN (temoin d a0c36016 : en-tete a 3537, lecture a +419).
const (
	p3ScanDebut = 1000
	p3ScanFin   = 1700
)

// p3ScanCatalogue rend la lecture de meilleur score dans [en-tete + p3ScanDebut, en-tete +
// p3ScanFin), et sa position (-1 : aucune).
func p3ScanCatalogue(b []byte, n *p3Naissance, cfg FrameConfig, cat map[uint32]bool) ([]p3Arme, int) {
	k := n.annoncees
	if k == 0 {
		k = 3
	}
	var best []p3Arme
	meilleur, score := -1, 0
	for o := n.bit + p3ScanDebut; o < n.bit+p3ScanFin && o+33 <= len(b)*8; o++ {
		armes, sc := p3Enchainer(b, o, k, cfg, cat)
		if sc > score {
			best, meilleur, score = armes, o, sc
		}
	}
	return best, meilleur
}

// p3Enchainer lit `k` emplacements d arme depuis `o` et rend leur SCORE : le nombre de familles
// presentes au catalogue ; 0 si le PREMIER emplacement est vide ou hors catalogue (un bipede nait
// arme — sans cette regle, une porte fortuite plus tot decale toute la lecture d un emplacement,
// mesure sur 81c02726 : [vide, AR, Sidekick] au lieu de [AR, Sidekick, 00007CA9], et sur b1f01a33 :
// [24E674E0, FD98554C, 230447B1] au lieu de [FD98554C, 230447B1, vide]), si aucune famille n est au
// catalogue ou si la lecture sort du paquet. Une valeur presente HORS catalogue ne compte pas et
// est marquee. Le meilleur score gagne, le premier bit a egalite.
func p3Enchainer(b []byte, o, k int, cfg FrameConfig, cat map[uint32]bool) ([]p3Arme, int) {
	br := LecteurSur(b)
	br.poserCadre(cfg)
	br.SetBitPos(o)
	var out []p3Arme
	dedans := 0
	for i := 0; i < k; i++ {
		debut := br.BitPos()
		if debut+1 > len(b)*8 {
			return nil, 0
		}
		a := p3Arme{idx: 43 + i, hi: noVariant, lo: noVariant, startBit: debut, cadreOK: true}
		br2 := LecteurSur(b)
		br2.SetBitPos(debut)
		if br2.ReadBit() {
			a.present, a.hi, a.lo = true, uint32(br2.ReadBits(32)), uint32(br2.ReadBits(32))
			switch {
			case cat[a.hi]:
				dedans++
			case i == 0:
				return nil, 0
			default:
				a.horsCat = true
			}
		} else if i == 0 {
			return nil, 0
		}
		consumeWeaponStateTypeInfoVariant(br)
		if br.BitPos() > len(b)*8 {
			return nil, 0
		}
		out = append(out, a)
	}
	if dedans == 0 {
		return nil, 0
	}
	return out, dedans
}

// p3Hexes formate un compte de valeurs.
func p3Hexes(m map[uint32]int) string {
	s := ""
	for k, v := range m {
		s += fmt.Sprintf(" %08X:%d", k, v)
	}
	return "[" + s + " ]"
}
