package main

// comparaison.go — CONFRONTER DEUX EMPREINTES ET NOMMER LE SENS DE CHAQUE ECART.
//
// LE SENS EST LE PRODUIT DE CET OUTIL. « Les deux artefacts different » n'apprend rien : ce
// qu'un balayage de non-regression doit rendre, c'est la difference entre « le nouveau porte un
// calque que l'ancien n'avait pas » (attendu, quarante bumps de schema) et « le nouveau a perdu
// ce que l'ancien savait dire » (la seule chose qu'on cherche).
//
// LA TOLERANCE NUMERIQUE existe parce que certaines mesures sont des FLOTTANTS de
// dequantification (les bornes de la carte) : les comparer a l'egalite stricte remplirait le
// rapport d'ecarts au dernier bit, et noierait les pertes reelles.

import "math"

// Sens d'un ecart.
const (
	SensPerte       = "perte"
	SensGain        = "gain"
	SensChangement  = "changement"
	SensApparu      = "apparu"
	SensDisparu     = "disparu"
	toleranceRelate = 1e-6
)

// Difference est UN ecart entre deux empreintes.
type Difference struct {
	Axe      string  `json:"axe"`
	Metrique string  `json:"metrique"`
	Ancien   string  `json:"ancien"`
	Nouveau  string  `json:"nouveau"`
	Sens     string  `json:"sens"`
	Delta    float64 `json:"delta,omitempty"`
}

// BilanAxe resume un axe : combien de mesures ont perdu, gagne, change.
type BilanAxe struct {
	Axe         string `json:"axe"`
	Pertes      int    `json:"pertes"`
	Gains       int    `json:"gains"`
	Changements int    `json:"changements"`
	Identiques  int    `json:"identiques"`
}

// Rapport est le resultat complet d'une paire.
type Rapport struct {
	FichierAncien  string              `json:"fichierAncien"`
	FichierNouveau string              `json:"fichierNouveau"`
	MatchID        string              `json:"matchId"`
	SchemaAncien   int                 `json:"schemaAncien"`
	SchemaNouveau  int                 `json:"schemaNouveau"`
	Differences    []Difference        `json:"differences"`
	Identiques     int                 `json:"identiques"`
	Bilans         map[string]BilanAxe `json:"bilans"`
}

// Comparer confronte deux empreintes. L'ordre des arguments porte le SENS : `a` est la
// reference (l'ancien), `b` le re-cuit.
func Comparer(a, b Empreinte) Rapport {
	rap := Rapport{
		MatchID: b.MatchID, SchemaAncien: a.Schema, SchemaNouveau: b.Schema,
		Bilans: map[string]BilanAxe{},
	}
	if rap.MatchID == "" {
		rap.MatchID = a.MatchID
	}
	vues := map[string]bool{}
	for k, ma := range a.Mesures {
		vues[k] = true
		rap.ajouter(k, &ma, valeurOuNil(b.Mesures, k))
	}
	for k, mb := range b.Mesures {
		if vues[k] {
			continue
		}
		rap.ajouter(k, nil, &mb)
	}
	trierDifferences(rap.Differences)
	return rap
}

// valeurOuNil rend un pointeur sur la mesure, ou nil quand elle est absente — l'absence est un
// resultat, pas un zero.
func valeurOuNil(m map[string]Mesure, k string) *Mesure {
	v, ok := m[k]
	if !ok {
		return nil
	}
	return &v
}

// ajouter classe UN ecart et le range dans le rapport.
func (r *Rapport) ajouter(k string, a, b *Mesure) {
	axe, metrique := decouper(k)
	d, ecart := classer(a, b)
	if !ecart {
		r.Identiques++
		bil := r.Bilans[axe]
		bil.Axe, bil.Identiques = axe, bil.Identiques+1
		r.Bilans[axe] = bil
		return
	}
	d.Axe, d.Metrique = axe, metrique
	r.Differences = append(r.Differences, d)
	bil := r.Bilans[axe]
	bil.Axe = axe
	switch d.Sens {
	case SensPerte, SensDisparu:
		bil.Pertes++
	case SensGain, SensApparu:
		bil.Gains++
	default:
		bil.Changements++
	}
	r.Bilans[axe] = bil
}

// classer nomme le sens d'un ecart, ou dit qu'il n'y en a pas.
//
// LES ZEROS NE COMPTENT PAS COMME ECART D'EXISTENCE : une mesure absente d'un cote et NULLE de
// l'autre dit la meme chose (« ce calque ne porte rien »). Les distinguer remplirait le rapport
// de bruit a chaque champ optionnel.
func classer(a, b *Mesure) (Difference, bool) {
	switch {
	case a == nil && b == nil:
		return Difference{}, false
	case a == nil:
		if b.EstNum && b.Num == 0 {
			return Difference{}, false
		}
		return Difference{Nouveau: b.String(), Sens: SensApparu, Delta: deltaDe(b)}, true
	case b == nil:
		if a.EstNum && a.Num == 0 {
			return Difference{}, false
		}
		return Difference{Ancien: a.String(), Sens: SensDisparu, Delta: -deltaDe(a)}, true
	case a.EstNum != b.EstNum:
		return Difference{Ancien: a.String(), Nouveau: b.String(), Sens: SensChangement}, true
	case !a.EstNum:
		if a.Txt == b.Txt {
			return Difference{}, false
		}
		return Difference{Ancien: a.Txt, Nouveau: b.Txt, Sens: SensChangement}, true
	}
	return classerNombres(a.Num, b.Num)
}

// classerNombres compare deux mesures numeriques a la tolerance des flottants.
func classerNombres(na, nb float64) (Difference, bool) {
	if proches(na, nb) {
		return Difference{}, false
	}
	sens := SensGain
	if nb < na {
		sens = SensPerte
	}
	return Difference{
		Ancien: formater(na), Nouveau: formater(nb), Sens: sens, Delta: nb - na,
	}, true
}

func deltaDe(m *Mesure) float64 {
	if m.EstNum {
		return m.Num
	}
	return 0
}

// proches dit si deux flottants sont egaux a la tolerance relative pres.
func proches(a, b float64) bool {
	if a == b {
		return true
	}
	ecart := math.Abs(a - b)
	echelle := math.Max(math.Abs(a), math.Abs(b))
	return ecart <= 1e-9+toleranceRelate*echelle
}

// decouper separe l'axe de la metrique dans une cle d'empreinte.
func decouper(k string) (string, string) {
	for i := 0; i < len(k); i++ {
		if k[i] == '/' {
			return k[:i], k[i+1:]
		}
	}
	return k, ""
}
