package domain

// Codes de résultat de match — miroir de l'enum Outcome de l'API Halo Infinite.
// Utilisés dans SquadMatchRow.Outcome, SynthesisMatchRow.Outcome, etc.
const (
	OutcomeUnknown = 0
	OutcomeDraw    = 1
	OutcomeWin     = 2
	OutcomeLoss    = 3
)
