package killsource

// types_alias.go — LES ALIAS DATES DES TYPES DE CONTRAT DE CETTE COUCHE (lot 2.6.2, 2026-09-16).
//
// Meme raison et meme date de retrait que `film/source/types_alias.go` : les declarations ont
// descendu dans `film/types` (feuille sans import du depot), et leurs consommateurs vivent pour
// partie dans `film/replay`, `replaybuild` et `internal/ops`, que d autres executeurs tiennent.
// Un alias de type est LE MEME type pour le compilateur — rien ne peut diverger.
//
// RETRAIT : volet grammaire / rejeu du lot 2.6, apres 2.5.e. Critere mesurable : plus aucun
// `killsource.{ApparStats,Assist,CoupleStats}` hors de ce paquet.

import "levelup/go-api/internal/games/halo_infinite/film/types"

type (
	// ApparStats : voir [types.ApparStats].
	ApparStats = types.ApparStats
	// Assist : voir [types.Assist].
	Assist = types.Assist
	// CoupleStats : voir [types.CoupleStats].
	CoupleStats = types.CoupleStats
)
