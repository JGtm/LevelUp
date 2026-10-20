// Package domain — match_steps.go : les ÉTAPES MESURÉES du chemin d'un match, de sa fin à
// ses citations, et le rapport « délai du débrief » qu'on en tire.
//
// Le délai qui compte pour un joueur est celui qui sépare la fin de son match du moment où
// toutes ses données sont lisibles. Chaque étape ci-dessous est un INSTANT capté là où le
// travail se fait (watcher, post-sync, file de construction, dépôt d'ouvrier, dérivations,
// citations) ; le rapport les met bout à bout, match par match.
package domain

import "time"

// MatchStep nomme une étape mesurée. Liste FERMÉE : [MatchSteps].
type MatchStep string

// Les étapes, dans l'ordre où elles se produisent pour un match cuit sans incident.
const (
	// MatchStepEnd : fin du match selon l'API (`match_registry.end_time_utc`). Référence de tous
	// les délais.
	MatchStepEnd MatchStep = "match_end"
	// MatchStepDetected : le watcher a sorti le match de sa file pour lancer un sync. Absente
	// quand le match a été trouvé par un cycle planifié.
	MatchStepDetected MatchStep = "detected"
	// MatchStepStatsWritten : les stats API du match sont écrites (début du post-sync qui l'a
	// inséré).
	MatchStepStatsWritten MatchStep = "stats_written"
	// MatchStepKillSourceWritten : la passe de lecture des morts dans le film est écrite.
	MatchStepKillSourceWritten MatchStep = "killsource_written"
	// MatchStepReplayEnqueued : la construction du rejeu est en file (placement ouvrier).
	MatchStepReplayEnqueued MatchStep = "replay_enqueued"
	// MatchStepReplayBuildStarted : la construction locale du rejeu commence (placement local).
	MatchStepReplayBuildStarted MatchStep = "replay_build_started"
	// MatchStepReplayClaimed : un ouvrier a pris le job (placement ouvrier).
	MatchStepReplayClaimed MatchStep = "replay_claimed"
	// MatchStepArtifactStored : l'artefact du rejeu est rangé à sa place canonique.
	MatchStepArtifactStored MatchStep = "artifact_stored"
	// MatchStepDerivationsWritten : les dérivations du rejeu (usage, objectifs, véhicules,
	// positions…) sont écrites et marquées.
	MatchStepDerivationsWritten MatchStep = "derivations_written"
	// MatchStepCitationsWritten : les citations du match sont écrites pour UN joueur suivi
	// (son xuid est dans le détail). Plusieurs par match.
	MatchStepCitationsWritten MatchStep = "citations_written"
)

// MatchSteps : toutes les étapes, dans l'ordre d'affichage.
var MatchSteps = []MatchStep{
	MatchStepEnd, MatchStepDetected, MatchStepStatsWritten, MatchStepKillSourceWritten,
	MatchStepReplayEnqueued, MatchStepReplayBuildStarted, MatchStepReplayClaimed,
	MatchStepArtifactStored, MatchStepDerivationsWritten, MatchStepCitationsWritten,
}

// DebriefObjective : délai maximal visé entre la fin d'un match et la dernière de ses données
// (décision utilisateur du 2026-10-17).
const DebriefObjective = 5 * time.Minute

// MatchStepEvent : un instant mesuré.
type MatchStepEvent struct {
	TitleSlug string
	MatchID   string
	Step      MatchStep
	// At : l'instant de l'étape, UTC — capté au moment où elle se produit, pas à l'écriture.
	At time.Time
	// Detail : précision libre et courte (xuid pour les citations, lieu pour l'artefact).
	Detail string
}

// MatchDelayStep : une étape d'un match dans le rapport.
type MatchDelayStep struct {
	Step MatchStep `json:"step" enum:"match_end,detected,stats_written,killsource_written,replay_enqueued,replay_build_started,replay_claimed,artifact_stored,derivations_written,citations_written"`
	// At : RFC3339 (UTC).
	At string `json:"at"`
	// OffsetSeconds : secondes écoulées depuis la fin du match. Absent sans fin de match connue.
	OffsetSeconds *float64 `json:"offset_seconds,omitempty"`
	// Detail : précision de l'étape (cf. [MatchStepEvent.Detail]).
	Detail string `json:"detail,omitempty"`
}

// MatchDelay : le délai du débrief d'un match.
type MatchDelay struct {
	TitleSlug string `json:"title_slug"`
	MatchID   string `json:"match_id"`
	// EndedAt : fin du match, RFC3339 ; vide si inconnue.
	EndedAt string `json:"ended_at,omitempty"`
	// Steps : la PREMIÈRE occurrence de chaque étape, sauf les citations (la dernière écrite
	// après les dérivations, sinon la dernière tout court), dans l'ordre de [MatchSteps].
	Steps []MatchDelayStep `json:"steps"`
	// CitationsPlayers : joueurs suivis dont les citations ont été écrites après les dérivations.
	CitationsPlayers int `json:"citations_players"`
	// TotalSeconds : de la fin du match à la dernière de (dérivations, citations postérieures aux
	// dérivations). Absent tant que l'une des trois manque.
	TotalSeconds *float64 `json:"total_seconds,omitempty"`
	// ObjectiveMet : TotalSeconds <= DebriefObjective. Absent avec TotalSeconds.
	ObjectiveMet *bool `json:"objective_met,omitempty"`
}
