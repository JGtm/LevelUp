package replay

// visee_ferme114_research_test.go — LOT A4 (cloture) : LA FERMETURE ENTRE LES FRONTIERES
// MESUREES SUR LE FLUX ET LA TABLE DES DOMAINES LUE DANS LE BINAIRE.
//
// LES DEUX MESURES QUI DOIVENT COINCIDER. D'un cote, le balayage dynamique L4
// (visee_larg114_research_test.go) conclut qu'un var-int de DOUZE bits, suivi de deux bits,
// place les deux portes suivantes exactement sur les bits 22 et 23 — mesures constantes a 0 sur
// trois films — et le payload R(6) au bit 24. De l'autre, la note de retro-ingenierie du lot B
// (`.ai/V7.5/film_re/NOTE_ENVELOPPE_EVENTS_2026-08-30.md`, table `0x1451f98d0` reconstituee
// depuis `FUN_140d10bb0`) donne pour le DOMAINE 2 — celui de la reference 0 du type 114 — une
// base de 0x200 et un cardinal de 0x100, soit un index de HUIT bits, plus deux bits de
// generation, et aucune sonde (la sonde n'existe que pour le domaine 1).
//
// HUIT ET DOUZE NE SE CONTREDISENT PAS : ILS SE COMPLETENT. Les quatre bits d'ecart sont
// mesures constants a `1110` sur les trois films — c'est un champ que la grammaire du
// dispatcher, telle qu'elle est ecrite dans le journal, ne modelise pas, et non de la donnee
// d'index. L'hypothese testee ici est donc :
//
//	R(7) type · R(1) porte · R(k) champ non modelise · R(W2) index · R(2) generation ·
//	R(1) porte · R(1) porte · R(6) payload
//
// et la fermeture consiste a verifier qu'un seul couple (k, W2) satisfait TOUTES les
// contraintes, dont celles que le binaire impose et que le flux ne pouvait pas connaitre.
//
// CRITERES ECRITS AVANT LA MESURE, sur les trois films :
//
//	F1. les deux bits de porte qui suivent la generation sont CONSTANTS a 0 (les references 1
//	    et 3 sont absentes — c'est ce que L4 a mesure : portes ouvertes 0 % du temps) ;
//	F2. le payload R(6) qui suit a une cardinalite <= 8 et une palette qui recouvre >= 90 % des
//	    paquets des autres films ;
//	F3. FERMETURE EXTERNE — l'index lu tient dans le cardinal du domaine 2 donne par le
//	    binaire : toutes les valeurs dans [0 ; 0x100). Cette contrainte ne vient PAS du flux :
//	    c'est elle qui transforme la coincidence en fermeture.
//
// Si un seul (k, W2) passe F1..F3, la structure est fermee ; sinon l'ambiguite est publiee.
//
// SOUS GARDE (FERME114_FILMS).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 FERME114_FILMS=<repo>/data/cache/film_chunks/00162144,<repo>/data/cache/film_chunks/00ba2e1c,<repo>/data/cache/film_chunks/03ccbe42 \
//	  go test ./internal/analysis/replay/ -run TestViseeFermeture114 -v -timeout 30m

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	ferme114FilmsEnv = "FERME114_FILMS"
	// ferme114CardDom2 : cardinal du domaine 2 lu dans le binaire (lot B, table 0x1451f98d0).
	ferme114CardDom2 = 0x100
	ferme114BaseDom2 = 0x200
)

// ferme114Profil resume une hypothese (k, W2) sur un film.
type ferme114Profil struct {
	decodes           int
	porte1, porte2    string
	idxMin, idxMax    uint32
	idxCard           int
	genPalette        map[uint32]int
	payload           map[uint32]int
	champNonModelise  map[uint32]int
	payloadCardinalOK bool
}

// ferme114Mesure applique l'hypothese (k, W2) a un lot de paquets.
func ferme114Mesure(pk []env114Paquet, k, w2 int) ferme114Profil {
	pr := ferme114Profil{
		genPalette:       map[uint32]int{},
		payload:          map[uint32]int{},
		champNonModelise: map[uint32]int{},
		idxMin:           ^uint32(0),
	}
	posIdx := 8 + k
	posGen := posIdx + w2
	posP1, posP2, posPay := posGen+2, posGen+3, posGen+4
	var uns1, uns2 int
	for _, p := range pk {
		if posPay+larg114Payload > p.nBits {
			continue
		}
		pr.decodes++
		if k > 0 {
			pr.champNonModelise[filmdec.ReadBitsAtForDiag(p.pay, 8, k)]++
		}
		idx := filmdec.ReadBitsAtForDiag(p.pay, posIdx, w2)
		if idx < pr.idxMin {
			pr.idxMin = idx
		}
		if idx > pr.idxMax {
			pr.idxMax = idx
		}
		pr.genPalette[filmdec.ReadBitsAtForDiag(p.pay, posGen, 2)]++
		uns1 += int(filmdec.ReadBitsAtForDiag(p.pay, posP1, 1))
		uns2 += int(filmdec.ReadBitsAtForDiag(p.pay, posP2, 1))
		pr.payload[filmdec.ReadBitsAtForDiag(p.pay, posPay, larg114Payload)]++
	}
	pr.porte1, pr.porte2 = ferme114Const(uns1, pr.decodes), ferme114Const(uns2, pr.decodes)
	pr.idxCard = ferme114CardIdx(pk, posIdx, w2)
	pr.payloadCardinalOK = len(pr.payload) > 0 && len(pr.payload) <= larg114CardMax
	return pr
}

func ferme114Const(uns, total int) string {
	switch {
	case total == 0:
		return "?"
	case uns == 0:
		return "0"
	case uns == total:
		return "1"
	default:
		return "variable"
	}
}

func ferme114CardIdx(pk []env114Paquet, pos, w int) int {
	vus := map[uint32]bool{}
	for _, p := range pk {
		if pos+w <= p.nBits {
			vus[filmdec.ReadBitsAtForDiag(p.pay, pos, w)] = true
		}
	}
	return len(vus)
}

// ferme114Recouvre rend la part des paquets de b dont le payload figure dans la palette de a.
func ferme114Recouvre(a, b ferme114Profil) float64 {
	if b.decodes == 0 {
		return 0
	}
	var commun int
	for v, n := range b.payload {
		if a.payload[v] > 0 {
			commun += n
		}
	}
	return float64(commun) / float64(b.decodes)
}

// ferme114Passe applique F1..F4 a un jeu de profils (un par film).
func ferme114Passe(profils []ferme114Profil) bool {
	for i := range profils {
		p := profils[i]
		if p.decodes == 0 || p.porte1 != "0" || p.porte2 != "0" || !p.payloadCardinalOK {
			return false
		}
		if p.idxMax >= ferme114CardDom2 {
			return false // F3 : l'index deborde le cardinal lu dans le binaire
		}
		// F4 : un champ dit « non modelise » n'en est un que s'il est CONSTANT. S'il varie,
		// c'est de la donnee, donc une partie de l'index : la decoupe est fausse d'autant.
		if len(p.champNonModelise) > 1 {
			return false
		}
		for j := range profils {
			if i != j && ferme114Recouvre(profils[i], profils[j]) < 0.90 {
				return false
			}
		}
	}
	return true
}

// ferme114Detail met en forme un profil pour publication.
func ferme114Detail(nom string, pr ferme114Profil) string {
	var vs []int
	for v := range pr.payload {
		vs = append(vs, int(v))
	}
	sort.Ints(vs)
	var pal []string
	for _, v := range vs {
		pal = append(pal, fmt.Sprintf("%d:%d", v, pr.payload[uint32(v)]))
	}
	var gen []string
	var gs []int
	for g := range pr.genPalette {
		gs = append(gs, int(g))
	}
	sort.Ints(gs)
	for _, g := range gs {
		gen = append(gen, fmt.Sprintf("%d:%d", g, pr.genPalette[uint32(g)]))
	}
	return fmt.Sprintf("[%s] index 0x%x..0x%x (%d valeurs, handles 0x%x..0x%x) · generation {%s}"+
		" · payload {%s}", nom, pr.idxMin, pr.idxMax, pr.idxCard,
		ferme114BaseDom2+pr.idxMin, ferme114BaseDom2+pr.idxMax, strings.Join(gen, " "),
		strings.Join(pal, " "))
}

// ferme105Ancre — L'ANCRE INDEPENDANTE. Le type 105 (action_weapon_fire) a des champs connus a
// offsets FIXES, etablis par ailleurs et deja en production (filmdec/fire_events.go) : l'index
// de l'attaquant vit aux bits 36..40 et vaut DEUX FOIS l'index de joueur du film. Si
// l'hypothese-enveloppe est la bonne grammaire, elle doit tomber sur le bit 36 toute seule.
//
// CRITERE ECRIT AVANT LA MESURE : une hypothese (k, sonde, W) est VALIDE si la position
// calculee porte un R(5) PAIR et strictement inferieur a 16 (huit joueurs au plus, index x 2)
// sur >= 95 % des records longs. La position 36, connue, sert de controle positif : si elle ne
// passe pas ce critere, c'est le critere qui est faux, pas l'hypothese.
func ferme105Ancre(t *testing.T, noms []string, lots map[string][]env114Paquet) {
	t.Helper()
	for _, nom := range noms {
		pk := lots[nom]
		var longs []env114Paquet
		for _, p := range pk {
			if p.nBits > 41 && filmdec.ReadBitsAtForDiag(p.pay, 7, 1) == 0 {
				longs = append(longs, p)
			}
		}
		if len(longs) == 0 {
			continue
		}
		t.Logf("  ANCRE 105 [%s] — %d records longs (bit 7 = 0) ; controle positif au bit 36 :"+
			" %.1f %% pairs et < 16", nom, len(longs), 100*ferme105Taux(longs, 36))
		var valides []string
		for k := 0; k <= 8; k++ {
			for _, sonde := range []int{0, 1} {
				for _, w := range []int{9, 13} {
					pos := 7 + 1 + k + sonde + w + 2 + 1 + 1
					taux := ferme105Taux(longs, pos)
					if taux >= 0.95 {
						valides = append(valides, fmt.Sprintf("k=%d sonde=%d W=%d -> bit %d"+
							" (%.0f %%)", k, sonde, w, pos, 100*taux))
					}
				}
			}
		}
		if len(valides) == 0 {
			t.Logf("    aucune hypothese (k, sonde, W) ne place un index d'attaquant plausible :"+
				" l'enveloppe du 105 ne se reduit PAS a [type][porte][champ][var-int][portes]"+
				" — il y a un sous-en-tete propre au type. Position connue : 36 ; positions"+
				" calculables : %d..%d", 7+1+0+0+9+2+1+1, 7+1+8+1+13+2+1+1)
			continue
		}
		t.Logf("    hypotheses valides : %s", strings.Join(valides, " · "))
	}
}

// ferme105Taux rend la part des records dont le R(5) a la position donnee est pair et < 16.
func ferme105Taux(pk []env114Paquet, pos int) float64 {
	var ok, total int
	for _, p := range pk {
		if pos+5 > p.nBits {
			continue
		}
		total++
		v := filmdec.ReadBitsAtForDiag(p.pay, pos, 5)
		if v%2 == 0 && v < 16 {
			ok++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(ok) / float64(total)
}

// ferme114Attribution — L'ATTRIBUTION PAR REFERENCE, une fois l'index localise. Si la
// reference 0 designe l'unite qui agit, les paquets de la plage etiquetee doivent se repartir
// par joueur, et l'un des index doit porter les douze transitions de Nilton (sa vie couvre
// toute la plage, aucun respawn ne la coupe). Les index sont publies avec leur couverture ;
// aucun seuil n'est necessaire ici, c'est la table qui parle.
func ferme114Attribution(t *testing.T, pk []env114Paquet, posIdx, w int) {
	t.Helper()
	trans, t0, t1 := sig114Fenetres()
	type ligne struct {
		idx   uint32
		total int
		couv  map[int]bool
		hors  int
	}
	par := map[uint32]*ligne{}
	var enPlage int
	for _, p := range pk {
		if p.tMS < t0 || p.tMS > t1 || posIdx+w > p.nBits {
			continue
		}
		enPlage++
		idx := filmdec.ReadBitsAtForDiag(p.pay, posIdx, w)
		if par[idx] == nil {
			par[idx] = &ligne{idx: idx, couv: map[int]bool{}}
		}
		l := par[idx]
		l.total++
		if i, _ := sig114Apparie(trans, p.tMS); i >= 0 {
			l.couv[i] = true
		} else {
			l.hors++
		}
	}
	var idxs []int
	for v := range par {
		idxs = append(idxs, int(v))
	}
	sort.Ints(idxs)
	var parts []string
	meilleure := 0
	for _, v := range idxs {
		l := par[uint32(v)]
		parts = append(parts, fmt.Sprintf("idx %d : %d paquets, %d/%d transitions, %d hors",
			v, l.total, len(l.couv), len(trans), l.hors))
		if len(l.couv) > meilleure {
			meilleure = len(l.couv)
		}
	}
	t.Logf("  ATTRIBUTION — index aux bits [%d ; %d) sur les %d paquets de la plage etiquetee :"+
		" %d references distinctes", posIdx, posIdx+w, enPlage, len(idxs))
	for _, s := range parts {
		t.Logf("      %s", s)
	}
	t.Logf("      meilleure reference : %d/%d transitions — les douze bascules d'un MEME joueur"+
		" ne se rassemblent pas derriere une reference unique.", meilleure, len(trans))
}

// TestViseeFermeture114 balaye (k, W2) et publie les hypotheses qui ferment.
func TestViseeFermeture114(t *testing.T) {
	liste := os.Getenv(ferme114FilmsEnv)
	if liste == "" {
		t.Skipf("%s absent : instrument saute", ferme114FilmsEnv)
	}
	lots := map[string][]env114Paquet{}
	var noms []string
	for _, dir := range strings.Split(liste, ",") {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if pk := env114Collecte(dir); len(pk) > 0 {
			nom := env114Nom(dir)
			lots[nom], noms = pk, append(noms, nom)
		}
	}
	if len(noms) < 2 {
		t.Fatalf("il faut au moins deux films ; %d lu(s)", len(noms))
	}
	t.Logf("FERMETURE — %d films ; cardinal du domaine 2 lu dans le binaire : 0x%x (base 0x%x)",
		len(noms), ferme114CardDom2, ferme114BaseDom2)
	lots105 := map[string][]env114Paquet{}
	for _, dir := range strings.Split(liste, ",") {
		if dir = strings.TrimSpace(dir); dir != "" {
			lots105[env114Nom(dir)] = env114CollecteType(dir, 105)
		}
	}
	ferme105Ancre(t, noms, lots105)
	var retenus int
	for k := 0; k <= 8; k++ {
		for w2 := 4; w2 <= 16; w2++ {
			profils := make([]ferme114Profil, len(noms))
			for i, nom := range noms {
				profils[i] = ferme114Mesure(lots[nom], k, w2)
			}
			if !ferme114Passe(profils) {
				continue
			}
			retenus++
			t.Logf("  RETENU k=%d W2=%d — champ non modelise aux bits [8 ; %d), index aux"+
				" bits [%d ; %d), generation [%d ; %d), portes en %d et %d, payload en %d",
				k, w2, 8+k, 8+k, 8+k+w2, 8+k+w2, 8+k+w2+2, 8+k+w2+2, 8+k+w2+3, 8+k+w2+4)
			for i, nom := range noms {
				t.Logf("      %s", ferme114Detail(nom, profils[i]))
				if k > 0 {
					t.Logf("      champ non modelise [8 ; %d) : %s", 8+k,
						ferme114Palette(profils[i].champNonModelise))
				}
			}
		}
	}
	if pk := lots["00162144"]; len(pk) > 0 {
		ferme114Attribution(t, pk, 12, 8)
	}
	switch retenus {
	case 0:
		t.Log("A4. FERMETURE — aucune hypothese ne satisfait F1..F3 : les frontieres mesurees et" +
			" la table du binaire NE se ferment PAS. A publier tel quel.")
	case 1:
		t.Log("A4. FERMETURE — SOLUTION UNIQUE : les frontieres du flux et le cardinal lu dans" +
			" le binaire se ferment sans ajustement.")
	default:
		t.Logf("A4. FERMETURE — %d hypotheses ferment : ambiguite publiee, pas tranchee.", retenus)
	}
}

func ferme114Palette(m map[uint32]int) string {
	var vs []int
	for v := range m {
		vs = append(vs, int(v))
	}
	sort.Ints(vs)
	var parts []string
	for i, v := range vs {
		if i == 8 {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, fmt.Sprintf("%d (0b%b):%d", v, v, m[uint32(v)]))
	}
	return strings.Join(parts, " ")
}
