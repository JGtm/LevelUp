package domain

// Codes de résultat de match — miroir de l'enum Outcome de l'API Halo Infinite.
// Utilisés dans SquadMatchRow.Outcome, SynthesisMatchRow.Outcome, etc.
//
// Convention : ces codes int sont la représentation Halo brute. La
// représentation canonique cross-titre est `canonical.Outcome` (string :
// "win", "loss", "tie", "dnf"). ADR 0002 et 0006 documentent la séparation
// entre code int (transit DB / DTOs Halo) et label canonique (cross-titre).
//
// Note revue 2026-04-29 axe 6 : OutcomeDraw (1) est l'équivalent int de
// canonical.OutcomeTie. La divergence "draw" vs "tie" relève du mapping TS
// (cf. apps/web/src/lib/outcomes ou outcome-color.ts).
const (
	OutcomeUnknown = 0
	OutcomeDraw    = 1 // Halo "Tied" — alias canonical.OutcomeTie
	OutcomeWin     = 2
	OutcomeLoss    = 3
	OutcomeDNF     = 4 // Halo "DidNotFinish" — match abandonné
)
