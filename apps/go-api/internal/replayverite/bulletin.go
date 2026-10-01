package replayverite

// bulletin.go — CE QUE LE BANC DIT D'UN ARTEFACT : scores, preuves, violations, replis.

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// Les identifiants stables des mesures (conception §4 et §5). Ils sont la cle de comparaison
// avant/apres et le libelle du rendu : un renommage casse la comparaison avec un bulletin ancien.
const (
	ScoreKills      = "O-K1 kills (statborg)"
	ScoreMorts      = "O-K2 morts (statborg)"
	ScoreAssists    = "O-K3 assistances (statborg)"
	ScoreMortsVies  = "O-V1 morts par les vies"
	ScoreFinsDeVie  = "O-V2 fins de vie (sans identite)"
	ScoreCamps      = "O-S1 score par camp"
	ScoreMarque     = "O-S2 actions de marque par camp"
	ScoreEquipes    = "O-T1 equipes du roster"
	ScorePersonnel  = "O-S3 score personnel"
	PreuveFermeture = "P-1 paquets fermes au bit pres"
	PreuveContradic = "P-2 preuves contradictoires (image-cle)"
	PreuveVerdicts  = "P-4 verdicts de couverture"
	ViolSaut        = "V-1 saut"
	ViolHorsEmprise = "V-2 objet hors emprise"
	ViolHorsVie     = "V-3 action hors vie"
	ViolDeuxCorps   = "V-4 deux corps"
	ViolAbsent      = "V-5 acteur absent ou mort"
	ViolTrajetLoin  = "V-6 trajet loin du vehicule"
	ViolIdentite    = "V-7 identite discordante"
	ViolMortSansVie = "V-8 mort sans fin de vie"
)

// Score est le resultat d'un oracle : vrais positifs, faux positifs, faux negatifs, et les unites
// NON NOTEES parce qu'elles ont ete appariees sur l'oracle lui-meme (circularite).
type Score struct {
	VP, FP, FN int
	// Exclus : unites non notees (circularite), affichees.
	Exclus int
	// NonNote : le score entier est circulaire sur ce temoin (ex. O-S1 avec `teamIdentity` `a`).
	NonNote bool
	// Ecarts : par unite (xuid, camp), `pub/off` quand ils different — le detail qui permet
	// d'attribuer une hausse de FP ou de FN sans relancer d'outil.
	Ecarts map[string]Ecart
}

// Ecart est la valeur publiee et la valeur officielle d'une unite.
type Ecart struct{ Pub, Off int }

// noter ajoute une unite au score.
func (s *Score) noter(cle string, pub, off int) {
	s.VP += min(pub, off)
	s.FP += max(0, pub-off)
	s.FN += max(0, off-pub)
	if pub != off {
		if s.Ecarts == nil {
			s.Ecarts = map[string]Ecart{}
		}
		s.Ecarts[cle] = Ecart{Pub: pub, Off: off}
	}
}

// Preuve est une preuve interne de lecture, sans oracle externe.
type Preuve struct {
	Valeur int
	// PlusEstMieux : vrai = une baisse est un MANQUE ; faux = une hausse est un MANQUE.
	PlusEstMieux bool
}

// Violation est une classe de vraisemblance : des instances nommees (cle stable pour l'avant/
// apres) et des instances ANONYMES (compteurs publies sans detail).
type Violation struct {
	Instances []string
	Anonymes  int
	// Exemptees : instances ecartees par une exemption nommee (portes de carte...), affichees.
	Exemptees int
}

// Total rend le nombre d'instances.
func (v Violation) Total() int { return len(v.Instances) + v.Anonymes }

// Bulletin est ce que le banc dit d'UN artefact.
type Bulletin struct {
	MatchID    string
	Scores     map[string]Score
	Preuves    map[string]Preuve
	Violations map[string]Violation
	// Replis : les replis declenches (R-1), par nom.
	Replis map[string]int
}

// Noter juge un artefact contre les faits du match qui l'ont cuit, et contre l'oracle officiel
// quand il est fourni (nil = les scores qui l'attendent sont absents du bulletin).
//
// Pur : aucune E/S. Les faits sont ceux de la cuisson (`domain.MatchFacts`) ; ils servent d'oracle
// SAUF la ou la methode publiee dit qu'ils ont servi d'appariement (cf. doc.go). L'oracle
// (`domain.MatchOracle`) n'est jamais une entree de la cuisson : il n'est circulaire nulle part.
func Noter(d *Document, faits domain.MatchFacts, oracle *domain.MatchOracle) Bulletin {
	b := Bulletin{
		MatchID:    d.MatchID,
		Scores:     map[string]Score{},
		Preuves:    map[string]Preuve{},
		Violations: map[string]Violation{},
		Replis:     map[string]int{},
	}
	noterKDA(d, faits, b.Scores)
	noterVies(d, faits, b.Scores)
	noterCamps(d, faits, b.Scores)
	if oracle != nil {
		b.Scores[ScorePersonnel] = scorePersonnel(d, *oracle)
	}
	noterPreuves(d, b.Preuves)
	b.Violations[ViolSaut] = violationsSauts(d, faits.MapID)
	b.Violations[ViolHorsEmprise] = violationsHorsEmprise(d)
	b.Violations[ViolHorsVie] = violationsHorsVie(d)
	b.Violations[ViolDeuxCorps] = violationsDeuxCorps(d)
	b.Violations[ViolAbsent] = violationsAbsents(d)
	b.Violations[ViolTrajetLoin] = violationsTrajets(d)
	b.Violations[ViolIdentite] = violationsIdentite(d)
	b.Violations[ViolMortSansVie] = violationsMortsSansVie(d)
	for _, r := range d.Coverage.Fallbacks {
		b.Replis[r.Name] += r.Hits
	}
	return b
}

// clesTriees rend les cles d'une carte dans un ordre stable.
func clesTriees[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
