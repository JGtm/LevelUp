package analysis

import (
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// AnalysisPendingWindow est l'âge de synchro au-delà duquel un match n'est plus jamais
// « en cours d'analyse », quoi qu'il manque : l'indicateur ne doit pas durer éternellement,
// les sections reprennent leur état normal « indisponible ».
const AnalysisPendingWindow = 24 * time.Hour

// AnalysisPendingInput regroupe les faits d'un match pour `IsAnalysisPending`.
type AnalysisPendingInput struct {
	Probe domain.AnalysisProbe
	// Now : l'instant de la décision (injecté, jamais lu dans la règle).
	Now time.Time
	// ReplayExpected : le titre produit des artefacts de rejeu (capability
	// `film.replay_artifact`). Faux : le rejeu manquant n'est jamais une attente.
	ReplayExpected bool
	// ReplayKnown : la présence des artefacts a pu être lue. Faux (service non câblé, listing
	// en échec) : on ne conclut pas « rejeu manquant » sur une lecture ratée.
	ReplayKnown bool
	// HasReplay : l'artefact de rejeu 2D du match existe.
	HasReplay bool
}

// AnalysisPendingParts rend les causes pour lesquelles l'analyse du film d'un match
// synchronisé n'est pas finie, dans l'ordre stable `kill_source`, `citations`, `replay`
// (constantes `domain.AnalysisPart*`). Nil si rien ne manque.
//
// Une cause n'est rendue QUE si le match a été synchronisé il y a MOINS de
// `AnalysisPendingWindow` et que :
//   - kill_source : la source du kill n'est pas traitée (`Probe.KillSourceDone`) ;
//   - citations : les citations ne sont pas calculées (`Probe.HasCitationRows`) ;
//   - replay : le rejeu est absent alors que le titre en produit, que la lecture des
//     artefacts a abouti et que le film n'est pas déclaré absent.
//
// C'est la SEULE copie de la règle : `IsAnalysisPending` s'en dérive. La porte de titre
// (capability) n'est pas évaluée ici : un titre sans la capability n'a pas de résolveur
// câblé, donc jamais d'appel.
func AnalysisPendingParts(in AnalysisPendingInput) []string {
	p := in.Probe
	if p.FirstSyncAt == nil {
		return nil
	}
	if in.Now.Sub(*p.FirstSyncAt) >= AnalysisPendingWindow {
		return nil
	}
	var parts []string
	if !p.KillSourceDone {
		parts = append(parts, domain.AnalysisPartKillSource)
	}
	if !p.HasCitationRows {
		parts = append(parts, domain.AnalysisPartCitations)
	}
	if in.ReplayExpected && in.ReplayKnown && !in.HasReplay && !p.FilmAbsent {
		parts = append(parts, domain.AnalysisPartReplay)
	}
	return parts
}

// IsAnalysisPending dit si l'analyse du film d'un match synchronisé n'est pas finie : vrai si
// et seulement si `AnalysisPendingParts` rend au moins une cause.
func IsAnalysisPending(in AnalysisPendingInput) bool {
	return len(AnalysisPendingParts(in)) > 0
}

// AnalysisStateOf dit où en est l'analyse d'un match dont `AnalysisPendingParts` est non vide :
// `domain.AnalysisStateInProgress` si et seulement si le décodage en cours sur la machine
// (`decodingMatchID`, vide quand aucun décodage ne tourne) porte sur CE match, sinon
// `domain.AnalysisStateQueued`. « En cours » n'est dit que si c'est vrai : une analyse en attente
// derrière le verrou d'un autre film, ou d'un outil qui travaille ailleurs, reste « en attente ».
func AnalysisStateOf(matchID, decodingMatchID string) string {
	if matchID != "" && strings.EqualFold(matchID, decodingMatchID) {
		return domain.AnalysisStateInProgress
	}
	return domain.AnalysisStateQueued
}
