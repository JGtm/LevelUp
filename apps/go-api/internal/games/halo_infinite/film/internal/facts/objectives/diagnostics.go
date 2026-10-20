package objectives

import "levelup/go-api/internal/games/halo_infinite/film/internal/constat"

// diagnostics.go — CE QUE LES LECTURES D OBJECTIFS CONSTATENT, RENDU A L ORCHESTRATEUR (lot J12.3,
// ADR 0034 D-4).
//
// Ce paquet ne journalise plus. Les constats des lectures du statborg tombent dans le
// [constat.Diagnostics] que l appelant passe ([StatRecordsAvecReplis], [StatRecordsBornes]) ;
// ceux des derivations d evenements et de series tombent dans l enregistreur des consultations
// ([ReplisALaConsultation.Diagnostics]). L orchestrateur (`film/replay`, `replaybuild`) les releve
// et les journalise avec SON contexte (`replay.JournaliserDiagnostics`).

// Les diagnostics de `objectives`.
const (
	// DiagFilmSansManifeste : aucun chunk du film n est decrit par le manifeste.
	DiagFilmSansManifeste constat.Code = "objectives.film_sans_manifeste"
	// DiagStatborgTronque : le plafond d enregistrements du statborg est atteint.
	DiagStatborgTronque constat.Code = "objectives.statborg_tronque"
	// DiagDeroulageRejete : un deroulage aberrant a ete rejete par la borne par pas.
	DiagDeroulageRejete constat.Code = "objectives.deroulage_rejete"
	// DiagPlafondEvenements : le plafond d evenements du film est atteint.
	DiagPlafondEvenements constat.Code = "objectives.plafond_evenements"
	// DiagBornesAppliquees : resume de passe quand les bornes de deroulage ont coute.
	DiagBornesAppliquees constat.Code = "objectives.bornes_appliquees"
	// DiagSerieNonChronologique : une serie cumulee reculait dans le temps, points ecartes.
	DiagSerieNonChronologique constat.Code = "objectives.serie_non_chronologique"
)
