package replayverite

import "strings"

// preuves.go — LES PREUVES INTERNES DE LECTURE (aucun oracle externe).
//
// P-3 (part `credit-concordant` des kills) n'est PAS ici : les kills individuels ne sont pas dans
// l'artefact, ils arrivent avec la section 5 des faits du film (phase 2b, decision D-3).

// rangDesVerdicts : l'ordre des verdicts de `coverage.verdict` (film/replay/coverage_bridge.go).
// `nominal` vaut mieux que `partiel : ...`, qui vaut mieux que tout le reste (`aucune donnee`,
// `non publiable : ...`). Un verdict qui descend d'un rang est un MANQUE.
const (
	rangNonPubliable = 0
	rangPartiel      = 1
	rangNominal      = 2
)

const (
	verdictNominal        = "nominal"
	prefixeVerdictPartiel = "partiel"
)

// noterPreuves : P-1 (paquets fermes au bit pres), P-2 (preuves contradictoires), P-4 (verdicts).
func noterPreuves(d *Document, preuves map[string]Preuve) {
	if c := d.Coverage.ContinuousFire; c != nil {
		preuves[PreuveFermeture] = Preuve{Valeur: c.Closed, PlusEstMieux: true}
	}
	if k := d.Coverage.Keyframes; k != nil {
		preuves[PreuveContradic] = Preuve{Valeur: k.ContradictoryProofs + k.Refutations}
	}
	for cle, v := range d.Coverage.Verdict {
		preuves[PreuveVerdicts+" "+cle] = Preuve{Valeur: rangDuVerdict(v), PlusEstMieux: true}
	}
}

// rangDuVerdict classe un verdict publie.
func rangDuVerdict(v string) int {
	switch {
	case v == verdictNominal:
		return rangNominal
	case strings.HasPrefix(v, prefixeVerdictPartiel):
		return rangPartiel
	default:
		return rangNonPubliable
	}
}
