package grammar

// imagecle_oracle_n2_research_test.go — PHASE 5a, OBJECTIF 3 : `n2` COMME ORACLE DE LARGEUR D'ETAT PAR DEFAUT.
//
// # L'IDEE, ET POURQUOI ELLE EST GRATUITE
//
// `FUN_142e2bfd0` lit, autour de l'etat par defaut, DEUX MOTS DE TAILLE :
//
//	[108 bits d'en-tete] R(32) n1 | etat par defaut (largeur w) | R(32) n2 | composants
//
// `n1` et `n2` sont les deux TAILLES DE TAMPON que le jeu alloue pour l'archetype
// (`vtable[0x20]` et `vtable[0x10]` dans `FUN_1408f1aa4`). Elles sont donc CONSTANTES par
// archetype et par build — la phase 4 l'a verifie sur ti=9 (`n1 = 12` sur 2 424 records,
// `n2 = 0x88 = 136` = le `memset` de `vtable[0x88]`).
//
// `n1` se lit a une position FIXE (108) : sa constance teste l'en-tete. `n2` se lit APRES
// l'etat par defaut : une largeur fausse le fait atterrir sur des bits quelconques, et sa
// dispersion le dit. **`n2` est donc un DETECTEUR de largeur fausse et un MESUREUR de la bonne**
// (celle qui le rend constant) : un oracle 100 % INTERNE au film, sans capture ni base.
//
// # LES TROIS EPREUVES, ECRITES AVANT LA MESURE
//
//	A  LA LARGEUR PORTEE   `n2` lu APRES le deserialiseur d'etat par defaut du depot.
//	                       Constant => la largeur portee est indiscernable d'une juste.
//	B  LE BALAYAGE         `n2` lu a `140 + w` pour w de 0 a `imcoBalayageMax` : les w qui
//	                       rendent `n2` CONSTANT et credible. Un flux sature de zeros en
//	                       produit par hasard — les suites qui DOUBLENT (`v, 2v, 4v`) sont une
//	                       zone constante lue par une fenetre glissante, pas des candidats :
//	                       `imcoZones` les replie sur leur debut.
//	C  LA FERMETURE        pour chaque candidat, le corps pose a `172 + w` atterrit-il sur le
//	                       record suivant ? C'est une SECONDE CHAINE, sans etape commune avec
//	                       B — et elle tranche quand B est ambigu.
//
// # CE QUE CET INSTRUMENT NE FAIT PAS
//
// Il ne corrige AUCUN deserialiseur. Il publie la largeur impliquee et l'ecart a la largeur
// portee — la correction est un lot de production, hors perimetre de la phase 5a.
//
// Garde CHUNK00_FILMS. Aucun code de production modifie.

import (
	"fmt"
	"levelup/go-api/internal/games/halo_infinite/film/profile"
	"sort"
	"testing"
)

const (
	// imcoDebutN1 est la position de `n1` depuis le debut du record : la fin de l'en-tete
	// par entite de `FUN_142e2bfd0`.
	imcoDebutN1 = profile.KeyframeEnTeteBits
	// imcoDebutEtat est la position ou commence l'etat par defaut : juste apres `n1`.
	imcoDebutEtat = profile.KeyframeEnTeteBits + profile.KeyframeMotDeTailleBits
	// imcoBalayageMax borne le balayage de largeur. Le plus large etat par defaut connu du
	// dossier est celui du bipede (198 bits mesures en live) ; 1 024 laisse quatre fois la
	// place sans faire exploser le cout.
	imcoBalayageMax = 1024
	// imcoTailleMax borne ce qu'un mot de taille peut valoir pour etre credible : c'est une
	// taille de TAMPON en octets. Les valeurs lues par la phase 4 vont de 2 a 2 712.
	imcoTailleMax = 65536
	// imcoCandidatsMax borne la liste de candidats publiee par ligne.
	imcoCandidatsMax = 16
)

// imcoAncre est UN record : son payload, sa position de depart, et la frontiere visee (le
// debut du record suivant, ou -1 pour le dernier record du payload).
type imcoAncre struct {
	Pay       []byte
	Bit, Want int
}

// imcoGroupe rassemble les records d'UN archetype d'UN build.
type imcoGroupe struct {
	Build   string
	TI      int
	Ancres  []imcoAncre
	Films   map[string]bool
	N1      map[uint64]int
	Portee  map[int]int    // largeur consommee par le deserialiseur du depot, par record
	N2Porte map[uint64]int // `n2` lu APRES cette largeur portee (epreuve A)
}

// imcoCle identifie un groupe.
type imcoCle struct {
	Build string
	TI    int
}

// imcoCollecter accumule tous les records d'un film dans les groupes par (build, archetype).
func imcoCollecter(f imcFilm, groupes map[imcoCle]*imcoGroupe) {
	for _, pay := range f.Pays {
		for _, s := range KeyframeRecordSpans(pay) {
			cle := imcoCle{Build: f.Build, TI: s.TI}
			g := groupes[cle]
			if g == nil {
				g = &imcoGroupe{
					Build: f.Build, TI: s.TI, Films: map[string]bool{},
					N1: map[uint64]int{}, Portee: map[int]int{}, N2Porte: map[uint64]int{},
				}
				groupes[cle] = g
			}
			g.Films[f.Nom] = true
			want := s.BitEnd
			if want >= len(pay)*8 {
				want = -1 // dernier record du payload : aucune frontiere annoncee
			}
			g.Ancres = append(g.Ancres, imcoAncre{Pay: pay, Bit: s.BitStart, Want: want})
			g.N1[kfReadBits(pay, s.BitStart+imcoDebutN1, profile.KeyframeMotDeTailleBits)]++
			e := profilLireEtatComplet(pay, s.BitStart, s.TI)
			g.Portee[e.DSBits]++
			g.N2Porte[e.N2]++
		}
	}
}

// imcoN1Modal rend la valeur modale de `n1` et le nombre de records qui ne la portent pas.
//
// POURQUOI CE FILTRE EXISTE. `n1` est une taille de tampon, CONSTANTE par archetype. Un record
// qui en porte une autre — en pratique 0 — est une ANCRE FORTUITE du balayeur (3 a 15 par
// groupe, mesure du 2026-09-13). Les garder rend tout balayage de largeur impossible.
func (g *imcoGroupe) imcoN1Modal() (modal uint64, ecartes int) {
	best := -1
	for v, n := range g.N1 {
		if n > best || (n == best && v < modal) {
			modal, best = v, n
		}
	}
	for v, n := range g.N1 {
		if v != modal {
			ecartes += n
		}
	}
	return modal, ecartes
}

// imcoRetenues rend les ancres du groupe qui portent la valeur modale de `n1`.
func (g *imcoGroupe) imcoRetenues() []imcoAncre {
	modal, ecartes := g.imcoN1Modal()
	if ecartes == 0 {
		return g.Ancres
	}
	out := make([]imcoAncre, 0, len(g.Ancres))
	for _, a := range g.Ancres {
		if kfReadBits(a.Pay, a.Bit+imcoDebutN1, profile.KeyframeMotDeTailleBits) == modal {
			out = append(out, a)
		}
	}
	return out
}

// imcoBalayer rend les largeurs w pour lesquelles `n2` est CONSTANT et credible sur tout le
// groupe (ancres fortuites retirees), et la valeur qu'il prend pour chacune.
func (g *imcoGroupe) imcoBalayer(ancres []imcoAncre) (ws []int, vals map[int]uint64) {
	vals = map[int]uint64{}
	if len(ancres) == 0 {
		return nil, vals
	}
	for w := 0; w <= imcoBalayageMax; w++ {
		p := imcoDebutEtat + w
		v := kfReadBits(ancres[0].Pay, ancres[0].Bit+p, profile.KeyframeMotDeTailleBits)
		if v == 0 || v > imcoTailleMax {
			continue
		}
		bon := true
		for _, a := range ancres[1:] {
			if kfReadBits(a.Pay, a.Bit+p, profile.KeyframeMotDeTailleBits) != v {
				bon = false
				break
			}
		}
		if bon {
			ws = append(ws, w)
			vals[w] = v
		}
	}
	return ws, vals
}

// imcoZones replie les candidats en ZONES DE CONSTANCE.
//
// POURQUOI. Une suite `w, w+1, w+2` dont les valeurs doublent (`v, 2v, 4v`) n'est PAS une suite
// de candidats independants : c'est UNE zone de bits constante lue a des decalages successifs
// par une fenetre glissante de 32 bits. Les compter comme huit candidats fabrique une ambiguite
// qui n'existe pas ; les replier rend le vrai compte — le DEBUT de zone est le seul candidat.
func imcoZones(ws []int, vals map[int]uint64) []int {
	var debuts []int
	for i, w := range ws {
		if i > 0 && ws[i-1] == w-1 && (vals[w-1]<<1)&0xFFFFFFFF == vals[w] {
			continue // meme zone, fenetre glissee d'un bit
		}
		debuts = append(debuts, w)
	}
	return debuts
}

// imcoHistoInt rend un histogramme trie et borne.
func imcoHistoInt(m map[int]int, max int) string {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	s := ""
	for i, k := range ks {
		if i >= max {
			s += fmt.Sprintf(" ...(%d valeurs)", len(ks))
			break
		}
		if s != "" {
			s += " "
		}
		s += fmt.Sprintf("%d:x%d", k, m[k])
	}
	return s
}

// imcoLigne est ce que l'instrument publie pour un groupe (build x archetype).
type imcoLigne struct {
	Build              string
	TI, Records, Films int
	// Fortuites : ancres retirees parce que leur `n1` n'est pas la valeur modale.
	Fortuites int
	N1Val     uint64
	N1Stable  bool
	// PorteeUnique : largeur consommee par le deserialiseur du depot si elle est la meme sur
	// tous les records retenus, -1 sinon. N2PorteStabl : `n2` lu APRES cette largeur.
	PorteeUnique int
	N2PorteStabl bool
	N2PorteVal   uint64
	Portee       map[int]int
	N2Porte      map[uint64]int
	// Zones : debuts de zone de constance (candidats replies). Vals : la valeur de `n2` par
	// candidat. Fermetures : ce que l'epreuve de fermeture rend pour chacun.
	Zones         []int
	Vals          map[int]uint64
	Fermetures    map[int]int
	Eprouvees     []int
	FermeturesTot int
}

// imcoAnalyser remplit la ligne d'un groupe : filtre les ancres fortuites, rejoue le
// deserialiseur porte, balaie les largeurs, replie les zones, et confronte chaque candidat a
// l'epreuve de FERMETURE (le corps pose a `172 + w` doit atterrir sur le record suivant).
func imcoAnalyser(g *imcoGroupe, regs map[string]*Registry) imcoLigne {
	anc := g.imcoRetenues()
	l := imcoLigne{
		Build: g.Build, TI: g.TI, Records: len(anc), Films: len(g.Films),
		Fortuites: len(g.Ancres) - len(anc), PorteeUnique: -1,
		Portee: map[int]int{}, N2Porte: map[uint64]int{},
	}
	l.N1Val, _ = g.imcoN1Modal()
	l.N1Stable = len(g.N1) == 1
	for _, a := range anc {
		e := profilLireEtatComplet(a.Pay, a.Bit, g.TI)
		l.Portee[e.DSBits]++
		l.N2Porte[e.N2]++
	}
	if len(l.Portee) == 1 {
		for w := range l.Portee {
			l.PorteeUnique = w
		}
	}
	l.N2PorteStabl = len(l.N2Porte) == 1
	for v := range l.N2Porte {
		l.N2PorteVal = v
	}
	ws, vals := g.imcoBalayer(anc)
	l.Zones, l.Vals = imcoZones(ws, vals), vals
	l.Eprouvees = imcoLargeursAEprouver(l.Zones, l.PorteeUnique)
	l.Fermetures, l.FermeturesTot = imcoEprouverZones(l.Eprouvees, anc, regs[g.Build])
	return l
}

// imcoLargeursAEprouver rend les largeurs soumises a l'epreuve de fermeture : les premieres
// zones de constance, PLUS un petit jeu systematique autour de la largeur portee.
//
// POURQUOI LE JEU SYSTEMATIQUE. La constance de `n2` et la fermeture sont DEUX CHAINES
// distinctes, et elles peuvent se contredire : une zone de bits constante peut exister a une
// largeur qui ne ferme pas, et une largeur qui ferme peut ne pas rendre `n2` constant. Ne
// tester que les zones ferait manquer le second cas — donc taire une contradiction au lieu de
// la publier.
func imcoLargeursAEprouver(zones []int, portee int) []int {
	vus := map[int]bool{}
	var out []int
	ajoute := func(w int) {
		if w >= 0 && !vus[w] {
			vus[w] = true
			out = append(out, w)
		}
	}
	for i, w := range zones {
		if i >= imcoCandidatsMax {
			break
		}
		ajoute(w)
	}
	for _, w := range []int{0, 1, 2, portee, portee + 1, portee + 2, portee - 1} {
		ajoute(w)
	}
	sort.Ints(out)
	return out
}

// imcoEprouverZones confronte chaque debut de zone a l'epreuve de FERMETURE : en posant le
// corps du record a `172 + w`, combien de records atterrissent exactement sur l'ancre du
// record suivant ? C'est une SECONDE CHAINE, sans etape commune avec la constance de `n2`.
//
// Le corps se pose sans toucher au code : `HeaderBits = 108 + w` avec les deux mots de taille
// et SANS deserialiseur d'etat par defaut place la boucle de composants a `108 + w + 32 + 32`,
// c'est-a-dire exactement `172 + w`.
func imcoEprouverZones(zones []int, anc []imcoAncre, reg *Registry) (map[int]int, int) {
	out := map[int]int{}
	if reg == nil || len(zones) == 0 {
		return out, 0
	}
	total := 0
	for _, a := range anc {
		if a.Want < 0 {
			continue
		}
		total++
		for _, w := range zones {
			// L'ORACLE `n2` : l'etat par defaut REMPLACE par un decalage de `w` bits, pour
			// mesurer une largeur qu'aucun deserialiseur ne porte encore. C'est l'un des deux
			// temoins nommes de `keyframeFullStateTemoin` — jamais une lecture de production.
			tr := walkKeyframeFullState(a.Pay, a.Bit, reg, contexteDInstrument(), keyframeFullStateTemoin{
				EnTeteBits: profile.KeyframeEnTeteBits + w, SansEtatParDefaut: true,
			})
			if tr.DesyncAt < 0 && tr.EndBit == a.Want {
				out[w]++
			}
		}
	}
	return out, total
}

// imcoVerdict nomme ce que l'oracle dit de cet archetype.
func (l imcoLigne) imcoVerdict() string {
	switch {
	case l.N2PorteStabl && l.PorteeUnique >= 0:
		return fmt.Sprintf("OK largeur portee %d bits FIXE, n2 = %d constant",
			l.PorteeUnique, l.N2PorteVal)
	case l.N2PorteStabl:
		return fmt.Sprintf("OK n2 = %d constant malgre une largeur portee VARIABLE (%s)",
			l.N2PorteVal, imcoHistoInt(l.Portee, 3))
	case len(l.Zones) == 0:
		return "MUET : aucune largeur ne rend n2 constant — etat par defaut VARIABLE"
	case len(l.Zones) == 1:
		return fmt.Sprintf("A CORRIGER : n2 implique %d bits (porte %s) -> ecart %+d, n2 = %d",
			l.Zones[0], imcoHistoInt(l.Portee, 2), l.Zones[0]-l.PorteeUnique, l.Vals[l.Zones[0]])
	default:
		return fmt.Sprintf("AMBIGU : %d zones de constance", len(l.Zones))
	}
}

// imcoZonesTexte rend les debuts de zone avec la valeur de `n2` et l'epreuve de fermeture.
func (l imcoLigne) imcoZonesTexte() string {
	s := ""
	for i, w := range l.Zones {
		if i >= imcoCandidatsMax {
			s += fmt.Sprintf(" ...(%d zones)", len(l.Zones))
			break
		}
		if s != "" {
			s += " · "
		}
		s += fmt.Sprintf("w=%d n2=%d ferme %d/%d", w, l.Vals[w], l.Fermetures[w], l.FermeturesTot)
	}
	if s == "" {
		s = "-"
	}
	return s
}

// TestImageCleOracleN2 EST LA MESURE : par build et par archetype, `n1`, la largeur portee, la
// constance de `n2`, les largeurs que le balayage designe et ce que la fermeture en dit.
func TestImageCleOracleN2(t *testing.T) {
	groupes := map[imcoCle]*imcoGroupe{}
	regs := map[string]*Registry{}
	films := 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		f, ok := imcCharger(t, dir)
		if !ok {
			continue
		}
		films++
		regs[f.Build] = f.Reg
		imcoCollecter(f, groupes)
	}
	if films == 0 {
		t.Skip("aucun film exploitable")
	}
	lignes := make([]imcoLigne, 0, len(groupes))
	for _, cle := range imcoCles(groupes) {
		l := imcoAnalyser(groupes[cle], regs)
		lignes = append(lignes, l)
		imcoPublierLigne(t, l)
	}
	imcoBilan(t, lignes)
}

// imcoPublierLigne ecrit les trois lignes d'un groupe.
func imcoPublierLigne(t *testing.T, l imcoLigne) {
	t.Helper()
	t.Logf("%-10s ti=%-2d | %6d records (%d ancres fortuites retirees), %d films | "+
		"n1 = %d%s | largeur portee %s | n2 porte %s",
		l.Build, l.TI, l.Records, l.Fortuites, l.Films, l.N1Val,
		imcoSiStable(l.N1Stable), imcoHistoInt(l.Portee, 4),
		imcoHistoInt(imcoU64VersInt(l.N2Porte), 4))
	t.Logf("              -> %s", l.imcoVerdict())
	t.Logf("              -> zones : %s", l.imcoZonesTexte())
	t.Logf("              -> largeurs qui FERMENT : %s", l.imcoFermeturesTexte())
}

// imcoFermeturesTexte rend les largeurs eprouvees qui font atterrir la marche sur le record
// suivant — la SECONDE CHAINE. Vide si aucune ne ferme.
func (l imcoLigne) imcoFermeturesTexte() string {
	ws := make([]int, 0, len(l.Fermetures))
	for w, n := range l.Fermetures {
		if n > 0 {
			ws = append(ws, w)
		}
	}
	sort.Ints(ws)
	s := ""
	for _, w := range ws {
		if s != "" {
			s += " · "
		}
		s += fmt.Sprintf("w=%d ferme %d/%d (%.1f %%)", w, l.Fermetures[w], l.FermeturesTot,
			100*float64(l.Fermetures[w])/float64(l.FermeturesTot))
	}
	if s == "" {
		s = fmt.Sprintf("AUCUNE sur %d largeurs eprouvees, %d records bornes",
			len(l.Eprouvees), l.FermeturesTot)
	}
	return s
}

// imcoSiStable annote `n1` quand toutes les ancres du groupe ne portent pas la meme valeur.
func imcoSiStable(stable bool) string {
	if stable {
		return ""
	}
	return " (ancres fortuites presentes)"
}

// imcoU64VersInt convertit un histogramme 64 bits en histogramme d'entiers pour l'affichage
// (les tailles de tampon tiennent largement dans un int).
func imcoU64VersInt(m map[uint64]int) map[int]int {
	out := make(map[int]int, len(m))
	for k, v := range m {
		out[int(k)] = v
	}
	return out
}

// imcoCles rend les cles triees par build puis archetype.
func imcoCles(m map[imcoCle]*imcoGroupe) []imcoCle {
	out := make([]imcoCle, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Build != out[j].Build {
			return out[i].Build < out[j].Build
		}
		return out[i].TI < out[j].TI
	})
	return out
}

// imcoBilan publie LA LISTE que le pilote attend : les etats par defaut a corriger, avec la
// largeur que `n2` implique.
func imcoBilan(t *testing.T, lignes []imcoLigne) {
	t.Helper()
	ok, corriger, ambigu, muet := 0, 0, 0, 0
	t.Logf("---- ETATS PAR DEFAUT A CORRIGER (largeur impliquee par n2) ----")
	for _, l := range lignes {
		switch {
		case l.N2PorteStabl:
			ok++
		case len(l.Zones) == 1:
			corriger++
			t.Logf("%-10s ti=%-2d : porte %s, n2 implique %d bits (ecart %+d), n2 = %d, "+
				"fermeture %d/%d", l.Build, l.TI, imcoHistoInt(l.Portee, 2), l.Zones[0],
				l.Zones[0]-l.PorteeUnique, l.Vals[l.Zones[0]],
				l.Fermetures[l.Zones[0]], l.FermeturesTot)
		case len(l.Zones) == 0:
			muet++
		default:
			ambigu++
			t.Logf("%-10s ti=%-2d : AMBIGU, %d zones — %s",
				l.Build, l.TI, len(l.Zones), l.imcoZonesTexte())
		}
	}
	t.Logf("BILAN : %d groupes (build x archetype) avec n2 CONSTANT a la largeur portee, "+
		"%d A CORRIGER avec une largeur unique, %d ambigus (plusieurs zones), "+
		"%d muets (aucune largeur constante)", ok, corriger, ambigu, muet)
}
