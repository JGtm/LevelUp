package replayverite

// comparaison.go — AVANT / APRES SUR UN MEME TEMOIN, ET LE VERDICT.
//
// Seul le DELTA decide (doc.go). Les regles, par priorite :
//
//	FAUX    un faux positif d'oracle monte, une classe de violation monte, ou un repli NOUVEAU se
//	        declenche (sauf compteur nouvellement branche au registre, decision D-6).
//	MANQUE  un faux negatif d'oracle monte, une preuve interne se degrade, ou une mesure disparait.
//	ok      rien de cela. Les gains s'affichent, ils ne decident rien ; une REATTRIBUTION (totaux
//	        egaux, ecarts par unite qui bougent) aussi, unite par unite (revue finale P1-e).

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// valeurAbsente : ce que le rendu affiche pour une mesure presente d un seul cote.
const valeurAbsente = "absent"

// Statut est le verdict du banc sur un temoin.
type Statut string

// Les trois verdicts, et les sens NON BLOQUANTS d un constat (gain, information, reattribution).
const (
	StatutOK     Statut = "ok"
	StatutFaux   Statut = "FAUX"
	StatutManque Statut = "MANQUE"
	sensGain     Statut = "gain"
	sensInfo     Statut = "info"
	// sensReattribution : les totaux FP/FN n'ont pas bouge mais les ecarts PAR UNITE (joueur, camp)
	// ont bouge (revue finale P1-e). Non bloquant, comme un changement : le banc ne sait pas lequel des
	// deux cotes attribue juste, il le rend visible, unite par unite.
	sensReattribution Statut = "reattribution"
)

// Constat est une mesure qui a bouge, avec son sens et le detail qui permet de l'attribuer.
type Constat struct {
	Mesure string
	Sens   Statut
	Avant  string
	Apres  string
	Detail []string
}

// Comparaison est le verdict du banc sur un temoin, et ce qui l'a decide.
type Comparaison struct {
	Statut   Statut
	Constats []Constat
	Avant    Bulletin
	Apres    Bulletin
}

// RegistreReplis dit, pour la revision d'AVANT, si le compteur de chaque repli du registre
// (`film/internal/facts/fallback`) etait branche. nil = registre inconnu : tout repli nouveau est
// alors un FAUX (le banc ne devine pas qu'un compteur vient d'etre branche).
type RegistreReplis map[string]bool

// Comparer rend le verdict du banc entre deux bulletins d'un meme temoin.
func Comparer(avant, apres Bulletin, registreAvant RegistreReplis) Comparaison {
	cs := comparerScores(avant.Scores, apres.Scores)
	cs = append(cs, comparerPreuves(avant.Preuves, apres.Preuves)...)
	cs = append(cs, comparerViolations(avant.Violations, apres.Violations)...)
	cs = append(cs, comparerReplis(avant.Replis, apres.Replis, registreAvant)...)
	c := Comparaison{Statut: StatutOK, Constats: cs, Avant: avant, Apres: apres}
	for _, x := range cs {
		switch {
		case x.Sens == StatutFaux:
			c.Statut = StatutFaux
		case x.Sens == StatutManque && c.Statut == StatutOK:
			c.Statut = StatutManque
		}
	}
	return c
}

// Bloquants rend les constats FAUX puis MANQUE.
func (c Comparaison) Bloquants() []Constat {
	var out []Constat
	for _, s := range []Statut{StatutFaux, StatutManque} {
		for _, x := range c.Constats {
			if x.Sens == s {
				out = append(out, x)
			}
		}
	}
	return out
}

func comparerScores(avant, apres map[string]Score) []Constat {
	var out []Constat
	for _, id := range clesTriees(union(avant, apres)) {
		a, okA := avant[id]
		b, okB := apres[id]
		switch {
		case okA && !okB:
			out = append(out, Constat{Mesure: id, Sens: StatutManque, Avant: formatScore(a), Apres: valeurAbsente})
			continue
		case !okA:
			out = append(out, Constat{Mesure: id, Sens: sensInfo, Avant: valeurAbsente, Apres: formatScore(b)})
			continue
		}
		if a.NonNote || b.NonNote {
			if a.NonNote != b.NonNote {
				out = append(out, Constat{Mesure: id, Sens: sensInfo, Avant: formatScore(a), Apres: formatScore(b)})
			}
			continue
		}
		out = append(out, constatsDesNotes(id, a, b)...)
	}
	return out
}

// constatsDesNotes rend les constats d UN score note des deux cotes : faux positifs ou negatifs en
// hausse, gain, ou reattribution.
func constatsDesNotes(id string, a, b Score) []Constat {
	var out []Constat
	detail := ecartsQuiBougent(a.Ecarts, b.Ecarts)
	if b.FP > a.FP {
		out = append(out, Constat{Mesure: id + " (faux positifs)", Sens: StatutFaux, Avant: formatScore(a), Apres: formatScore(b), Detail: detail})
	}
	if b.FN > a.FN {
		out = append(out, Constat{Mesure: id + " (faux negatifs)", Sens: StatutManque, Avant: formatScore(a), Apres: formatScore(b), Detail: detail})
	}
	if (b.FP < a.FP && b.FN <= a.FN) || (b.FN < a.FN && b.FP <= a.FP) {
		out = append(out, Constat{Mesure: id, Sens: sensGain, Avant: formatScore(a), Apres: formatScore(b), Detail: detail})
	}
	// LA REATTRIBUTION (revue finale P1-e) : une unite qui gagne ce qu une autre perd laisse les
	// totaux egaux ; seul le detail par unite la montre.
	if b.FP == a.FP && b.FN == a.FN && len(detail) > 0 {
		out = append(out, Constat{Mesure: id, Sens: sensReattribution, Avant: formatScore(a), Apres: formatScore(b), Detail: detail})
	}
	return out
}

func formatScore(s Score) string {
	if s.NonNote {
		return fmt.Sprintf("non note (circulaire, %d exclus)", s.Exclus)
	}
	txt := fmt.Sprintf("VP %d / FP %d / FN %d", s.VP, s.FP, s.FN)
	if s.Exclus > 0 {
		txt += fmt.Sprintf(" (%d exclus)", s.Exclus)
	}
	return txt
}

// ecartsQuiBougent : les unites dont (publie, officiel) change entre les deux bulletins.
func ecartsQuiBougent(avant, apres map[string]Ecart) []string {
	var out []string
	for _, k := range clesTriees(union(avant, apres)) {
		a, okA := avant[k]
		b, okB := apres[k]
		if okA && okB && a == b {
			continue
		}
		out = append(out, fmt.Sprintf("%s : %s -> %s", k, formatEcart(a, okA), formatEcart(b, okB)))
	}
	return out
}

func formatEcart(e Ecart, ok bool) string {
	if !ok {
		return "exact"
	}
	return fmt.Sprintf("publie %d / officiel %d", e.Pub, e.Off)
}

func comparerPreuves(avant, apres map[string]Preuve) []Constat {
	var out []Constat
	for _, id := range clesTriees(union(avant, apres)) {
		a, okA := avant[id]
		b, okB := apres[id]
		switch {
		case okA && !okB:
			out = append(out, Constat{Mesure: id, Sens: StatutManque, Avant: fmt.Sprint(a.Valeur), Apres: valeurAbsente})
		case !okA:
			out = append(out, Constat{Mesure: id, Sens: sensInfo, Avant: valeurAbsente, Apres: fmt.Sprint(b.Valeur)})
		case a.Valeur != b.Valeur:
			sens := sensGain
			if (b.Valeur < a.Valeur) == b.PlusEstMieux {
				sens = StatutManque
			}
			out = append(out, Constat{Mesure: id, Sens: sens, Avant: fmt.Sprint(a.Valeur), Apres: fmt.Sprint(b.Valeur)})
		}
	}
	return out
}

func comparerViolations(avant, apres map[string]Violation) []Constat {
	var out []Constat
	for _, id := range clesTriees(union(avant, apres)) {
		a, b := avant[id], apres[id]
		if a.Total() == b.Total() && a.Exemptees == b.Exemptees && equivalentes(a.Instances, b.Instances) {
			continue
		}
		nouvelles, disparues := differenceMultiensemble(a.Instances, b.Instances)
		detail := make([]string, 0, len(nouvelles)+len(disparues)+1)
		for _, n := range nouvelles {
			detail = append(detail, "+ "+n)
		}
		for _, n := range disparues {
			detail = append(detail, "- "+n)
		}
		if b.Anonymes != a.Anonymes {
			detail = append(detail, fmt.Sprintf("compteurs publies %d -> %d", a.Anonymes, b.Anonymes))
		}
		sens := sensInfo
		switch {
		case b.Total() > a.Total():
			sens = StatutFaux
		case b.Total() < a.Total():
			sens = sensGain
		}
		out = append(out, Constat{Mesure: id, Sens: sens, Avant: formatViolation(a), Apres: formatViolation(b), Detail: detail})
	}
	return out
}

func formatViolation(v Violation) string {
	txt := fmt.Sprint(v.Total())
	if v.Exemptees > 0 {
		txt += fmt.Sprintf(" (%d exemptees)", v.Exemptees)
	}
	return txt
}

func equivalentes(a, b []string) bool {
	n, d := differenceMultiensemble(a, b)
	return len(n) == 0 && len(d) == 0
}

// differenceMultiensemble rend les elements de `apres` absents d'`avant` (nouveaux) et l'inverse.
func differenceMultiensemble(avant, apres []string) (nouveaux, disparus []string) {
	compte := map[string]int{}
	for _, x := range avant {
		compte[x]++
	}
	for _, x := range apres {
		if compte[x] > 0 {
			compte[x]--
			continue
		}
		nouveaux = append(nouveaux, x)
	}
	for _, x := range clesTriees(compte) {
		for i := 0; i < compte[x]; i++ {
			disparus = append(disparus, x)
		}
	}
	sort.Strings(nouveaux)
	return nouveaux, disparus
}

// comparerReplis : R-1. Un nom qui se declenche apres sans s'etre declenche avant est un FAUX, sauf
// si le registre d'avant dit que son compteur n'etait pas branche (le repli decidait deja, on ne le
// comptait pas). Une hausse de declenchements est informative (decision D-6).
func comparerReplis(avant, apres map[string]int, registreAvant RegistreReplis) []Constat {
	var out []Constat
	for _, nom := range clesTriees(union(avant, apres)) {
		a, b := avant[nom], apres[nom]
		if a == b {
			continue
		}
		c := Constat{Mesure: "R-1 repli " + nom, Sens: sensInfo, Avant: fmt.Sprint(a), Apres: fmt.Sprint(b)}
		if a == 0 && b > 0 {
			branche, connu := registreAvant[nom]
			switch {
			case registreAvant != nil && connu && !branche:
				c.Detail = []string{"compteur nouvellement branche (registre d'avant : non branche)"}
			case registreAvant == nil:
				c.Sens, c.Detail = StatutFaux, []string{"repli nouveau (registre d'avant inconnu)"}
			default:
				c.Sens, c.Detail = StatutFaux, []string{"repli nouveau"}
			}
		}
		out = append(out, c)
	}
	return out
}

// union rend l'union des cles de deux cartes.
func union[V any](a, b map[string]V) map[string]bool {
	out := make(map[string]bool, len(a)+len(b))
	for k := range a {
		out[k] = true
	}
	for k := range b {
		out[k] = true
	}
	return out
}

// LireRegistre desserialise le registre des replis ecrit par `replay-verite -registre`
// (nom -> compteur branche). Un registre vide est refuse : il ferait passer tout repli nouveau pour
// « absent du registre d'avant » sans le dire.
func LireRegistre(blob []byte) (RegistreReplis, error) {
	var reg RegistreReplis
	if err := json.Unmarshal(blob, &reg); err != nil {
		return nil, fmt.Errorf("replayverite : registre des replis illisible : %w", err)
	}
	if len(reg) == 0 {
		return nil, errors.New("replayverite : registre des replis vide")
	}
	return reg, nil
}
