package analysis

import (
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

// IsAnalysisPending dit si l'analyse du film d'un match synchronisé n'est pas finie.
//
// Vrai si et seulement si le match a été synchronisé il y a MOINS de `AnalysisPendingWindow`
// ET qu'au moins un de ces trois points manque :
//   - la source du kill n'est pas traitée (`Probe.KillSourceDone`) ;
//   - les citations ne sont pas calculées (`Probe.HasCitationRows`) ;
//   - le rejeu est absent alors que le film n'est pas déclaré absent.
//
// La porte de titre (capability) n'est pas évaluée ici : un titre sans la capability n'a
// pas de résolveur câblé, donc jamais d'appel.
func IsAnalysisPending(in AnalysisPendingInput) bool {
	p := in.Probe
	if p.FirstSyncAt == nil {
		return false
	}
	if in.Now.Sub(*p.FirstSyncAt) >= AnalysisPendingWindow {
		return false
	}
	replayMissing := in.ReplayExpected && in.ReplayKnown && !in.HasReplay && !p.FilmAbsent
	return !p.KillSourceDone || !p.HasCitationRows || replayMissing
}
