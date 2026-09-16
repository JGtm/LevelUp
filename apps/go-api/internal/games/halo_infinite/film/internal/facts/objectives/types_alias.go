package objectives

// types_alias.go — LES ALIAS DATES DES TYPES DE CONTRAT DE CETTE COUCHE (lot 2.6.2, 2026-09-16).
//
// Meme raison et meme date de retrait que `film/source/types_alias.go` : les declarations ont
// descendu dans `film/types` (feuille sans import du depot), et leurs consommateurs vivent pour
// partie dans `film/replay`, `replaybuild` et `internal/sync`, que d autres executeurs tiennent.
// Un alias de type est LE MEME type pour le compilateur — rien ne peut diverger.
//
// RETRAIT : volet grammaire / rejeu du lot 2.6, apres 2.5.e. Critere mesurable : plus aucun
// `objectives.{DeathInstant,FlagSpan,FlagTrack,FlagGrabsNetPlayer,PlayerLine,ScorePoint,
// StatValue,StatRecord}` hors de ce paquet.

import "levelup/go-api/internal/games/halo_infinite/film/types"

type (
	// DeathInstant : voir [types.DeathInstant].
	DeathInstant = types.DeathInstant
	// FlagSpan : voir [types.FlagSpan].
	FlagSpan = types.FlagSpan
	// FlagTrack : voir [types.FlagTrack].
	FlagTrack = types.FlagTrack
	// FlagGrabsNetPlayer : voir [types.FlagGrabsNetPlayer].
	FlagGrabsNetPlayer = types.FlagGrabsNetPlayer
	// PlayerLine : voir [types.PlayerLine].
	PlayerLine = types.PlayerLine
	// ScorePoint : voir [types.ScorePoint].
	ScorePoint = types.ScorePoint
	// StatValue : voir [types.StatValue].
	StatValue = types.StatValue
	// StatRecord : voir [types.StatRecord].
	StatRecord = types.StatRecord
)
