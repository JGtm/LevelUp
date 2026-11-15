package tactical

// reapparition_ajustement.go — LES ECHANTILLONS, L'AJUSTEMENT DES POIDS ET LA VALIDATION
// CROISEE du modele de reapparition (cf. reapparition.go).
//
// # L'AJUSTEMENT
//
// Maximum de vraisemblance du logit conditionnel par montee de gradient a pas constant : la
// vraisemblance est concave, le pas (PasAjustement) et le nombre d'iterations (IterationsAjustement)
// sont fixes, et le resultat est donc DETERMINISTE pour un jeu d'echantillons donne. Un
// masque designe les variables ajustees ; les autres restent a zero.
//
// # LA VALIDATION CROISEE SE FAIT PAR MATCH
//
// Les reapparitions d'un meme match ne sont pas independantes (memes joueurs, memes positions) :
// le pli d'un evenement est celui de son match (PliDuMatch), si bien qu'un match entier est soit
// dans l'apprentissage, soit dans l'evaluation, jamais dans les deux.

import (
	"hash/fnv"
	"sync"
)

// Reglages de la montee de gradient.
const (
	PasAjustement        = 0.5
	IterationsAjustement = 300
	// PlisValidation : la validation croisee est a cinq plis.
	PlisValidation = 5
)

// Echantillon est une reapparition prete a etre notee : les variables de chaque candidat de sa
// carte, et le candidat reel.
type Echantillon struct {
	Pli   int
	Carte string
	// Feats : une ligne par candidat de la carte.
	Feats [][NbVariables]float64
	// Pos et Zones sont PARTAGES par tous les echantillons d'une carte (ne pas les modifier).
	// Zones : l'indice de zone nommee de chaque candidat, nil si on n'a pas de zones.
	Pos   []Point3
	Zones []int
	// Reel : l'indice du candidat reel, -1 s'il est hors des candidats.
	Reel int
	// Au est le point reel exact.
	Au Point3
	// ZoneReelle : l'indice de zone du point reel, -1 si inconnu.
	ZoneReelle int
}

// PliDuMatch rend le pli de validation d'un match, stable d'une execution a l'autre.
func PliDuMatch(matchID string, plis int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(matchID))
	return int(h.Sum32() % uint32(plis))
}

// ReglagesEchantillons regroupe ce que la construction des echantillons demande en plus des
// evenements.
type ReglagesEchantillons struct {
	// MatchsMinCarte : une carte vue dans moins de matchs est ecartee.
	MatchsMinCarte int
	// Vue rend la visibilite d'une carte ; nil ou renvoie nil : pas de septieme variable.
	Vue func(carte string) Visibilite
	// Zone rend l'indice de zone nommee d'un point d'une carte, -1 si aucune ; nil : pas de zone.
	Zone func(carte string, p Point3) int
}

// CandidatsParCarte deduit les candidats de chaque carte depuis TOUS les evenements donnes.
func CandidatsParCarte(evts []EvenementReapparition) map[string][]Candidat {
	par := map[string][]EvenementReapparition{}
	for _, e := range evts {
		par[e.CarteID] = append(par[e.CarteID], e)
	}
	out := make(map[string][]Candidat, len(par))
	for carte, es := range par {
		out[carte] = PointsCandidats(es)
	}
	return out
}

// ConstruireEchantillons note chaque evenement contre les candidats de sa carte. Un evenement
// sans ennemi, ou dont la carte n'a pas assez de matchs ou moins de deux candidats, est ecarte.
func ConstruireEchantillons(evts []EvenementReapparition, cands map[string][]Candidat, r ReglagesEchantillons) []Echantillon {
	matchs := map[string]map[string]bool{}
	for _, e := range evts {
		if matchs[e.CarteID] == nil {
			matchs[e.CarteID] = map[string]bool{}
		}
		matchs[e.CarteID][e.MatchID] = true
	}
	pos := map[string][]Point3{}
	zones := map[string][]int{}
	vues := map[string]Visibilite{}
	for carte, cs := range cands {
		for _, c := range cs {
			pos[carte] = append(pos[carte], c.Pos)
			if r.Zone != nil {
				zones[carte] = append(zones[carte], r.Zone(carte, c.Pos))
			}
		}
		if r.Vue != nil {
			vues[carte] = r.Vue(carte)
		}
	}
	var out []Echantillon
	for _, e := range evts {
		cs := cands[e.CarteID]
		if len(matchs[e.CarteID]) < r.MatchsMinCarte || len(e.Ennemis) == 0 || len(cs) < 2 {
			continue
		}
		s := Echantillon{
			Pli: PliDuMatch(e.MatchID, PlisValidation), Carte: e.CarteID, Pos: pos[e.CarteID],
			Zones: zones[e.CarteID], Reel: -1, Au: e.Reel, ZoneReelle: -1,
		}
		if r.Zone != nil {
			s.ZoneReelle = r.Zone(e.CarteID, e.Reel)
		}
		cle := CleDe(e.Reel)
		s.Feats = make([][NbVariables]float64, len(cs))
		for i, c := range cs {
			s.Feats[i] = Variables(e, c, c.Total-c.ParMatch[e.MatchID], vues[e.CarteID])
			if c.Cle == cle {
				s.Reel = i
			}
		}
		out = append(out, s)
	}
	return out
}

// Scores rend le score de chaque candidat d'un echantillon.
func (s Echantillon) Scores(w [NbVariables]float64) []float64 {
	out := make([]float64, len(s.Feats))
	for i, f := range s.Feats {
		out[i] = Score(f, w)
	}
	return out
}

// Ajuster rend les poids qui maximisent la vraisemblance des echantillons retenus par `garder`
// (ceux dont le candidat reel est connu), sur les seules variables du masque.
func Ajuster(echs []Echantillon, garder func(Echantillon) bool, masque [NbVariables]bool) [NbVariables]float64 {
	var w [NbVariables]float64
	for it := 0; it < IterationsAjustement; it++ {
		var g [NbVariables]float64
		n := 0.0
		for _, s := range echs {
			if s.Reel < 0 || !garder(s) {
				continue
			}
			n++
			p := Probabilites(s.Scores(w))
			for j := 0; j < NbVariables; j++ {
				attendu := 0.0
				for i, f := range s.Feats {
					attendu += p[i] * f[j]
				}
				g[j] += s.Feats[s.Reel][j] - attendu
			}
		}
		if n == 0 {
			return w
		}
		for j := range w {
			if masque[j] {
				w[j] += PasAjustement * g[j] / n
			}
		}
	}
	return w
}

// ToutesLesVariables est le masque qui ajuste les six variables de la mesure de depart ; la
// septieme s'y ajoute avec AvecVisibilite.
var ToutesLesVariables = [NbVariables]bool{true, true, true, true, true, true, false}

// AvecVisibilite est le masque des sept variables.
var AvecVisibilite = [NbVariables]bool{true, true, true, true, true, true, true}

// FrequenceSeule est le masque de la reference « frequence seule ».
var FrequenceSeule = [NbVariables]bool{VarFrequence: true}

// ScoresValides rend, pour chaque echantillon, les scores d'un modele ajuste SANS son pli
// (validation croisee a PlisValidation plis). Les plis s'ajustent en parallele : ils sont
// independants, et le resultat ne depend pas de l'ordre d'execution.
func ScoresValides(echs []Echantillon, masque [NbVariables]bool) [][]float64 {
	pred := make([][]float64, len(echs))
	var wg sync.WaitGroup
	for pli := 0; pli < PlisValidation; pli++ {
		wg.Add(1)
		go func(pli int) {
			defer wg.Done()
			w := Ajuster(echs, func(s Echantillon) bool { return s.Pli != pli }, masque)
			for i, s := range echs {
				if s.Pli == pli {
					pred[i] = s.Scores(w)
				}
			}
		}(pli)
	}
	wg.Wait()
	return pred
}
